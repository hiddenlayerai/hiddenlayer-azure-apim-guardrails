package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var (
	applyDryRun   bool
	applyPackage  string
	applyPackages []string
	applyYes      bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [api-id]",
	Short: "Apply HiddenLayer policy to an API",
	Long: `Apply HiddenLayer security scanning policy to a specific API.

If no API ID is provided, uses HL_TARGET_API from configuration.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview policy without applying")
	applyCmd.Flags().BoolVar(&applyYes, "yes", false, "Apply policy changes without confirmation")
	applyCmd.Flags().StringVar(&applyPackage, "package", "", "fragment package to use (default: auto-select)")
	applyCmd.Flags().StringSliceVar(&applyPackages, "packages", nil, "fragment packages to apply (comma-separated or repeated; cannot be used with --package)")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	var apiID string
	if len(args) > 0 {
		apiID = args[0]
	} else if cfg.TargetAPI != "" {
		apiID = cfg.TargetAPI
	} else {
		return fmt.Errorf("no API specified\n\nUsage: hiddenlayer-apim apply <api-id>\n\nOr set HL_TARGET_API")
	}

	if cmd.Flags().Changed("packages") && cmd.Flags().Changed("package") {
		return fmt.Errorf("--packages cannot be used with --package")
	}

	pkgs, err := resolveApplyPackages(cmd)
	if err != nil {
		return err
	}

	printHeader("Applying HiddenLayer Policy")
	packageLabel := ""
	if len(pkgs) == 1 {
		packageLabel = pkgs[0].Manifest.Name
		if pkgs[0].Manifest.Version != "" {
			packageLabel = fmt.Sprintf("%s@%s", pkgs[0].Manifest.Name, pkgs[0].Manifest.Version)
		}
	} else {
		labels := make([]string, 0, len(pkgs))
		for _, p := range pkgs {
			if p.Manifest.Version != "" {
				labels = append(labels, fmt.Sprintf("%s@%s", p.Manifest.Name, p.Manifest.Version))
			} else {
				labels = append(labels, p.Manifest.Name)
			}
		}
		packageLabel = fmt.Sprintf("%v", labels)
	}
	fmt.Printf("APIM Instance: %s\n", cfg.APIMName)
	fmt.Printf("Target API:    %s\n", apiID)
	fmt.Printf("Package:       %s\n", packageLabel)
	fmt.Printf("Dry Run:       %v\n", applyDryRun)
	fmt.Println()

	client, err := azure.NewClient(cfg.SubscriptionID, cfg.ResourceGroup, cfg.APIMName, verbose)
	if err != nil {
		return err
	}

	printInfo("Verifying API exists...")
	api, err := client.GetAPI(apiID)
	if err != nil {
		return err
	}
	printSuccess("Found API: %s (%s)", api.DisplayName, api.Name)

	printInfo("Verifying policy fragments...")
	verified := map[string]bool{}
	for _, pkg := range pkgs {
		for _, frag := range pkg.Fragments {
			if verified[frag.ID] {
				continue
			}
			verified[frag.ID] = true
			exists, err := waitForPolicyFragment(client, frag.ID, 5, 2*time.Second)
			if err != nil {
				return fmt.Errorf("error checking fragment '%s': %w", frag.ID, err)
			}
			if !exists {
				return fmt.Errorf("missing fragment '%s'\n\nRun 'hiddenlayer-apim deploy' first", frag.ID)
			}
		}
	}
	printSuccess("All policy fragments found")

	printInfo("Reading existing policy...")
	existingPolicy, err := client.GetAPIPolicy(apiID)
	if err != nil {
		return fmt.Errorf("failed to read existing policy: %w", err)
	}
	if existingPolicy == "" {
		existingPolicy = policy.BasePolicy
	}
	if allApplyFragmentsPresent(existingPolicy, pkgs) {
		printWarning("All requested fragments are already present in policy")
		return nil
	}

	newPolicy := existingPolicy
	for _, pkg := range pkgs {
		var err error
		newPolicy, err = policy.InjectHiddenLayerFragments(newPolicy, pkg)
		if err != nil {
			return err
		}
	}

	if newPolicy == existingPolicy {
		printWarning("All requested fragments are already present in policy")
		return nil
	}

	if applyDryRun {
		printHeader("Policy Preview (Dry Run)")
		fmt.Println(newPolicy)
		fmt.Println()
		printInfo("To apply this policy, run without --dry-run")
		return nil
	}

	if !applyYes {
		if err := confirmApplyPolicy(cmd.InOrStdin(), cmd.OutOrStdout(), apiID, pkgs); err != nil {
			return err
		}
	}

	printInfo("Injecting HiddenLayer fragments...")
	if err := client.SetAPIPolicy(apiID, newPolicy); err != nil {
		return err
	}
	printSuccess("Fragments injected successfully")

	printHeader("Complete")
	fmt.Printf("API '%s' is now protected by HiddenLayer.\n", apiID)
	fmt.Println()

	printHeader("Test Command")
	gatewayURL, err := client.GetGatewayURL()
	if err != nil {
		gatewayURL = fmt.Sprintf("https://%s.azure-api.net", cfg.APIMName)
	}
	gatewayURL = fmt.Sprintf("%s/%s", gatewayURL, api.Path)
	fmt.Println("Test your API with this curl command:")
	fmt.Println()
	fmt.Printf("curl -s %s/v1/chat/completions \\\n", gatewayURL)
	fmt.Println("  -H \"Content-Type: application/json\" \\")
	if api.SubscriptionRequired {
		fmt.Println("  -H \"Ocp-Apim-Subscription-Key: <your-subscription-key>\" \\")
	}
	fmt.Println("  -d '{")
	fmt.Println("    \"model\": \"gpt-4o-mini\",")
	fmt.Println("    \"messages\": [{\"role\": \"user\", \"content\": \"Say hello in one sentence.\"}]")
	fmt.Println("  }'")
	fmt.Println()
	fmt.Println("Add -i to see response headers (HL-Runtime-Action: BLOCK or empty)")
	fmt.Println()
	fmt.Printf("To check status: %s status\n", cyan("hiddenlayer-apim"))
	fmt.Printf("To remove:       %s remove %s\n", cyan("hiddenlayer-apim"), apiID)
	fmt.Println()

	return nil
}

func allApplyFragmentsPresent(policyXML string, pkgs []*policy.Package) bool {
	for _, pkg := range pkgs {
		if !policy.HasHiddenLayerFragments(policyXML, pkg) {
			return false
		}
	}
	return true
}

func confirmApplyPolicy(r io.Reader, w io.Writer, apiID string, pkgs []*policy.Package) error {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "This will update the APIM policy for API %q and add HiddenLayer fragments:\n", apiID)
	inbound, outbound := summarizeApplyFragments(pkgs)
	if len(inbound) > 0 {
		fmt.Fprintf(w, "  inbound:  %s\n", strings.Join(inbound, ", "))
	}
	if len(outbound) > 0 {
		fmt.Fprintf(w, "  outbound: %s\n", strings.Join(outbound, ", "))
	}
	fmt.Fprint(w, "Type 'yes' to continue: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return fmt.Errorf("apply cancelled")
	}
	if strings.TrimSpace(scanner.Text()) != "yes" {
		return fmt.Errorf("apply cancelled")
	}
	return nil
}

func summarizeApplyFragments(pkgs []*policy.Package) ([]string, []string) {
	inbound := []string{}
	outbound := []string{}
	seenInbound := map[string]bool{}
	seenOutbound := map[string]bool{}

	for _, pkg := range pkgs {
		for _, id := range pkg.InboundIDs() {
			if seenInbound[id] {
				continue
			}
			seenInbound[id] = true
			inbound = append(inbound, id)
		}
		for _, id := range pkg.OutboundIDs() {
			if seenOutbound[id] {
				continue
			}
			seenOutbound[id] = true
			outbound = append(outbound, id)
		}
	}

	return inbound, outbound
}

func resolveApplyPackages(cmd *cobra.Command) ([]*policy.Package, error) {
	if cmd.Flags().Changed("packages") {
		raw, err := cmd.Flags().GetStringSlice("packages")
		if err != nil {
			return nil, fmt.Errorf("read --packages: %w", err)
		}
		names, err := normalizeUniquePackageNames(raw)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("no packages specified")
		}
		pkgs := make([]*policy.Package, 0, len(names))
		for _, name := range names {
			p, err := policy.LoadPackage(name)
			if err != nil {
				return nil, fmt.Errorf("package load %q: %w", name, err)
			}
			pkgs = append(pkgs, p)
		}
		return pkgs, nil
	}

	pkgFlag := applyPackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return nil, fmt.Errorf("package selection: %w", err)
	}
	return []*policy.Package{pkg}, nil
}

func normalizeUniquePackageNames(in []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, raw := range in {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate package %q in --packages", name)
		}
		seen[name] = true
		out = append(out, name)
	}
	return out, nil
}

func waitForPolicyFragment(client *azure.Client, fragmentID string, attempts int, delay time.Duration) (bool, error) {
	for i := range attempts {
		exists, err := client.GetPolicyFragment(fragmentID)
		if err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return false, nil
}
