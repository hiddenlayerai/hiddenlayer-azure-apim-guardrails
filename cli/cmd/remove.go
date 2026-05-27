package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var removePackage string
var removePackages []string
var removeAll bool
var removeYes bool

var removeCmd = &cobra.Command{
	Use:   "remove [api-id]",
	Short: "Remove HiddenLayer policy from an API",
	Long: `Remove HiddenLayer security scanning fragments from a specific API.

This only removes the HiddenLayer fragments, preserving all other policy rules.

If no API ID is provided, uses HL_TARGET_API from configuration.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().StringVar(&removePackage, "package", "", "fragment package to remove (default: auto-select)")
	removeCmd.Flags().StringSliceVar(&removePackages, "packages", nil, "fragment packages to remove (comma-separated or repeated; cannot be used with --package)")
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Remove all detected HiddenLayer fragments from the API policy")
	removeCmd.Flags().BoolVar(&removeYes, "yes", false, "Remove policy fragments without confirmation")
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	var apiID string
	if len(args) > 0 {
		apiID = args[0]
	} else if cfg.TargetAPI != "" {
		apiID = cfg.TargetAPI
	} else {
		return fmt.Errorf("no API specified\n\nUsage: hiddenlayer-apim remove <api-id>\n\nOr set HL_TARGET_API")
	}

	printHeader("Removing HiddenLayer Policy")
	fmt.Printf("APIM Instance: %s\n", cfg.APIMName)
	fmt.Printf("Target API:    %s\n", apiID)
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

	printInfo("Reading existing policy...")
	existingPolicy, err := client.GetAPIPolicy(apiID)
	if err != nil {
		return fmt.Errorf("failed to read existing policy: %w", err)
	}

	if cmd.Flags().Changed("packages") && cmd.Flags().Changed("package") {
		return fmt.Errorf("--packages cannot be used with --package")
	}
	if removeAll && (cmd.Flags().Changed("packages") || cmd.Flags().Changed("package")) {
		return fmt.Errorf("--all cannot be used with --package or --packages")
	}

	pkgs, err := resolveRemovePackages(cmd)
	if err != nil {
		return err
	}
	idsToRemove, err := removeFragmentIDs(existingPolicy, pkgs, removeAll)
	if err != nil {
		return err
	}
	if len(idsToRemove) == 0 {
		printWarning("No removable HiddenLayer fragments found in policy")
		return nil
	}
	if removeAll {
		printInfo("Detected fragments: %s", fmt.Sprintf("%v", idsToRemove))
	}

	if !removeYes {
		if err := confirmRemovePolicy(cmd.InOrStdin(), cmd.OutOrStdout(), apiID, idsToRemove, removeAll); err != nil {
			return err
		}
	}

	newPolicy, err := policy.RemoveHiddenLayerFragmentsByIDs(existingPolicy, idsToRemove)
	if err != nil {
		return err
	}

	printInfo("Removing HiddenLayer fragments...")
	if err := client.SetAPIPolicy(apiID, newPolicy); err != nil {
		return err
	}
	printSuccess("Fragments removed successfully")

	printHeader("Complete")
	printRemoveComplete(apiID, pkgs, removeAll)
	fmt.Println()

	return nil
}

func printRemoveComplete(apiID string, pkgs []*policy.Package, all bool) {
	if all {
		fmt.Printf("API '%s' no longer uses HiddenLayer scanning.\n", apiID)
		fmt.Println()
		fmt.Printf("To re-enable: %s apply %s\n", cyan("hiddenlayer-apim"), apiID)
		return
	}

	fmt.Printf("Removed selected HiddenLayer package fragments from API '%s'.\n", apiID)
	fmt.Println()
	if len(pkgs) == 1 {
		fmt.Printf("To re-enable: %s apply %s --package %s\n", cyan("hiddenlayer-apim"), apiID, pkgs[0].Manifest.Name)
		return
	}

	names := make([]string, 0, len(pkgs))
	for _, pkg := range pkgs {
		names = append(names, pkg.Manifest.Name)
	}
	fmt.Printf("To re-enable: %s apply %s --packages %s\n", cyan("hiddenlayer-apim"), apiID, strings.Join(names, ","))
}

func resolveRemovePackages(cmd *cobra.Command) ([]*policy.Package, error) {
	if removeAll {
		return nil, nil
	}
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
			pkg, err := policy.LoadPackage(name)
			if err != nil {
				return nil, fmt.Errorf("package load %q: %w", name, err)
			}
			pkgs = append(pkgs, pkg)
		}
		return pkgs, nil
	}

	pkgFlag := removePackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return nil, fmt.Errorf("package selection: %w", err)
	}
	return []*policy.Package{pkg}, nil
}

func removeFragmentIDs(existingPolicy string, pkgs []*policy.Package, all bool) ([]string, error) {
	present := policy.DetectHiddenLayerFragmentIDs(existingPolicy)
	if all {
		return present, nil
	}

	requestedSet := map[string]bool{}
	var requested []string
	for _, pkg := range pkgs {
		for _, id := range pkg.AllFragmentIDs() {
			if requestedSet[id] {
				continue
			}
			requestedSet[id] = true
			requested = append(requested, id)
		}
	}

	presentSet := map[string]bool{}
	for _, id := range present {
		presentSet[id] = true
	}

	remaining := false
	for _, id := range present {
		if !requestedSet[id] {
			remaining = true
			break
		}
	}

	shared := map[string]bool{}
	if remaining {
		var err error
		shared, err = policy.SharedFragmentIDs()
		if err != nil {
			return nil, fmt.Errorf("compute shared fragments: %w", err)
		}
	}

	idsToRemove := []string{}
	for _, id := range requested {
		if !presentSet[id] {
			continue
		}
		if remaining && shared[id] {
			continue
		}
		idsToRemove = append(idsToRemove, id)
	}
	return idsToRemove, nil
}

func confirmRemovePolicy(r io.Reader, w io.Writer, apiID string, fragmentIDs []string, all bool) error {
	fmt.Fprintln(w)
	if all {
		fmt.Fprintf(w, "This will remove all detected HiddenLayer fragments from API %q:\n", apiID)
	} else {
		fmt.Fprintf(w, "This will update the APIM policy for API %q and remove HiddenLayer fragments:\n", apiID)
	}
	for _, id := range fragmentIDs {
		fmt.Fprintf(w, "  - %s\n", id)
	}
	fmt.Fprint(w, "Type 'yes' to continue: ")

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return fmt.Errorf("remove cancelled")
	}
	if strings.TrimSpace(scanner.Text()) != "yes" {
		return fmt.Errorf("remove cancelled")
	}
	return nil
}
