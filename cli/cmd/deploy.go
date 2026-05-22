package cmd

import (
	"fmt"
	"html"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var deployPackage string
var deployPackages []string
var deployOverwrite bool

type namedValueDef struct {
	name        string
	displayName string
	value       string
	secret      bool
}

type fragDef struct {
	id   string
	xml  string
	desc string
}

type deployClient interface {
	GetNamedValueInfo(name string) (*azure.NamedValue, error)
	CreateOrUpdateNamedValue(name, displayName, value string, secret bool) error
	GetPolicyFragmentContent(name string) (string, bool, error)
	CreateOrUpdatePolicyFragment(name, xmlContent, description string) error
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy HiddenLayer policy fragments to APIM",
	Long: `Deploy HiddenLayer policy fragments and named values to your
Azure API Management instance.

This creates:
  • Named values for HiddenLayer credentials
  • Policy fragments for OAuth and HiddenLayer evaluation (package-specific)`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&deployPackage, "package", "", "fragment package to deploy (default: auto-select)")
	deployCmd.Flags().StringSliceVar(&deployPackages, "packages", nil, "fragment packages to deploy (comma-separated or repeated; cannot be used with --package)")
	deployCmd.Flags().BoolVar(&deployOverwrite, "overwrite", false, "Overwrite existing HiddenLayer named values and policy fragments")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if err := cfg.ValidateHLCredentials(); err != nil {
		return err
	}

	if cmd.Flags().Changed("packages") && cmd.Flags().Changed("package") {
		return fmt.Errorf("--packages cannot be used with --package")
	}

	pkgs, err := resolveDeployPackages(cmd)
	if err != nil {
		return err
	}

	printHeader("Deploying HiddenLayer to APIM")
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
	fmt.Printf("Resource Group: %s\n", cfg.ResourceGroup)
	fmt.Printf("APIM Instance:  %s\n", cfg.APIMName)
	fmt.Printf("Package:        %s\n", packageLabel)
	fmt.Println()

	printInfo("Connecting to Azure...")
	client, err := azure.NewClient(cfg.SubscriptionID, cfg.ResourceGroup, cfg.APIMName, verbose)
	if err != nil {
		return err
	}
	printSuccess("Connected to Azure")

	printInfo("Deploying named values...")

	namedValues := []namedValueDef{
		{"hl-client-id", "hl-client-id", cfg.HLClientID, false},
		{"hl-client-secret", "hl-client-secret", cfg.HLClientSecret, true},
		{"hl-project-id", "hl-project-id", cfg.HLProjectID, false},
		{"hl-host", "hl-host", cfg.HLHost, false},
		{"hl-tenant-id", "hl-tenant-id", cfg.HLTenantID, false},
		{"hl-oauth-cache-seconds", "hl-oauth-cache-seconds", cfg.HLOAuthCacheSeconds, false},
	}

	for _, nv := range namedValues {
		action, err := deployNamedValue(client, nv, deployOverwrite)
		if err != nil {
			return err
		}
		if action == "skipped" {
			printWarning("Skipped named value: %s (secret exists; use --overwrite to update)", nv.name)
		} else if nv.secret {
			printSuccess("%s named value: %s (secret)", actionLabel(action), nv.name)
		} else {
			printSuccess("%s named value: %s", actionLabel(action), nv.name)
		}
	}

	printInfo("Deploying policy fragments...")

	fragments := map[string]fragDef{}
	for _, pkg := range pkgs {
		for _, frag := range pkg.Fragments {
			desc := pkg.FragmentDescription(frag.ID)
			if existing, ok := fragments[frag.ID]; ok {
				if existing.xml != frag.XML {
					return fmt.Errorf("fragment ID %q appears in multiple packages with different XML (%s vs %s)", frag.ID, existing.desc, desc)
				}
				continue
			}
			fragments[frag.ID] = fragDef{id: frag.ID, xml: frag.XML, desc: desc}
		}
	}

	for _, pkg := range pkgs {
		for _, fragID := range pkg.AllFragmentIDs() {
			def, ok := fragments[fragID]
			if !ok {
				continue
			}
			action, err := deployPolicyFragment(client, def, deployOverwrite)
			if err != nil {
				return err
			}
			printSuccess("%s fragment: %s (%s)", actionLabel(action), def.id, def.desc)
			delete(fragments, fragID)
		}
	}

	printHeader("Deployment Complete")
	fmt.Println("HiddenLayer policy fragments are now available in your APIM instance.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  %s list              # List available APIs\n", cyan("hiddenlayer-apim"))
	fmt.Printf("  %s apply <api-id>    # Apply policy to an API\n", cyan("hiddenlayer-apim"))
	fmt.Println()

	return nil
}

func actionLabel(action string) string {
	switch action {
	case "created":
		return "Created"
	case "unchanged":
		return "Unchanged"
	case "overwritten":
		return "Overwrote"
	default:
		if action == "" {
			return "Processed"
		}
		return strings.ToUpper(action[:1]) + action[1:]
	}
}

func deployNamedValue(client deployClient, nv namedValueDef, overwrite bool) (string, error) {
	existing, err := client.GetNamedValueInfo(nv.name)
	if err != nil {
		return "", fmt.Errorf("failed to read named value '%s': %w", nv.name, err)
	}
	if existing == nil {
		if err := client.CreateOrUpdateNamedValue(nv.name, nv.displayName, nv.value, nv.secret); err != nil {
			return "", fmt.Errorf("failed to create named value '%s': %w", nv.name, err)
		}
		return "created", nil
	}

	if existing.Secret {
		if !overwrite {
			return "skipped", nil
		}
		if err := client.CreateOrUpdateNamedValue(nv.name, nv.displayName, nv.value, nv.secret); err != nil {
			return "", fmt.Errorf("failed to overwrite named value '%s': %w", nv.name, err)
		}
		return "overwritten", nil
	}

	if existing.Value == nv.value && existing.Secret == nv.secret {
		return "unchanged", nil
	}
	if !overwrite {
		return "", fmt.Errorf("named value '%s' already exists with a different value; rerun with --overwrite to update it", nv.name)
	}
	if err := client.CreateOrUpdateNamedValue(nv.name, nv.displayName, nv.value, nv.secret); err != nil {
		return "", fmt.Errorf("failed to overwrite named value '%s': %w", nv.name, err)
	}
	return "overwritten", nil
}

func deployPolicyFragment(client deployClient, def fragDef, overwrite bool) (string, error) {
	existingXML, exists, err := client.GetPolicyFragmentContent(def.id)
	if err != nil {
		return "", fmt.Errorf("failed to read fragment '%s': %w", def.id, err)
	}
	if !exists {
		if err := client.CreateOrUpdatePolicyFragment(def.id, def.xml, def.desc); err != nil {
			return "", fmt.Errorf("failed to create fragment '%s': %w", def.id, err)
		}
		return "created", nil
	}
	if normalizePolicyFragmentXML(existingXML) == normalizePolicyFragmentXML(def.xml) {
		return "unchanged", nil
	}
	if !overwrite {
		return "", fmt.Errorf("fragment '%s' already exists with different XML; rerun with --overwrite to replace it", def.id)
	}
	if err := client.CreateOrUpdatePolicyFragment(def.id, def.xml, def.desc); err != nil {
		return "", fmt.Errorf("failed to overwrite fragment '%s': %w", def.id, err)
	}
	return "overwritten", nil
}

func normalizePolicyFragmentXML(xml string) string {
	if strings.Contains(xml, "&lt;") || strings.Contains(xml, "&gt;") {
		xml = html.UnescapeString(xml)
	}
	xml = strings.ReplaceAll(xml, "\r\n", "\n")
	xml = strings.ReplaceAll(xml, "\r", "\n")

	lines := strings.Split(xml, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if start >= end {
		return ""
	}

	return strings.Join(lines[start:end], "\n")
}

func resolveDeployPackages(cmd *cobra.Command) ([]*policy.Package, error) {
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

	pkgFlag := deployPackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return nil, fmt.Errorf("package selection: %w", err)
	}
	return []*policy.Package{pkg}, nil
}
