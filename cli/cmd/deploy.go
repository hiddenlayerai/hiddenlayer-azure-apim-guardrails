package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var deployPackage string
var deployPackages []string

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

	envVarToValue := map[string]string{
		"HL_REQ_EVALS_POLICY_ID":  cfg.HLReqEvalsPolicyID,
		"HL_RESP_EVALS_POLICY_ID": cfg.HLRespEvalsPolicyID,
	}

	missingEnvVars := map[string][]string{}
	for _, p := range pkgs {
		req, ok := policy.PolicyDefinitionIDRequirementForPackage(p.Manifest.Name)
		if !ok {
			continue
		}
		if envVarToValue[req.EnvVar] == "" {
			missingEnvVars[req.EnvVar] = append(missingEnvVars[req.EnvVar], p.Manifest.Name)
		}
	}
	if len(missingEnvVars) > 0 {
		var missingKeys []string
		for k := range missingEnvVars {
			missingKeys = append(missingKeys, k)
		}
		sort.Strings(missingKeys)

		var parts []string
		for _, envVar := range missingKeys {
			pkgNames := missingEnvVars[envVar]
			sort.Strings(pkgNames)
			parts = append(parts, fmt.Sprintf("%s (for packages: %s)", envVar, strings.Join(pkgNames, ", ")))
		}
		return fmt.Errorf("missing required env var(s): %s", strings.Join(parts, "; "))
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

	type namedValueDef struct {
		name        string
		displayName string
		value       string
		secret      bool
	}

	namedValues := []namedValueDef{
		{"hl-client-id", "hl-client-id", cfg.HLClientID, false},
		{"hl-client-secret", "hl-client-secret", cfg.HLClientSecret, true},
		{"hl-project-id", "hl-project-id", cfg.HLProjectID, false},
		{"hl-host", "hl-host", cfg.HLHost, false},
		{"hl-tenant-id", "hl-tenant-id", cfg.HLTenantID, false},
	}

	policyNamedValues := map[string]namedValueDef{}
	for _, p := range pkgs {
		req, ok := policy.PolicyDefinitionIDRequirementForPackage(p.Manifest.Name)
		if !ok {
			continue
		}
		policyNamedValues[req.NamedValue] = namedValueDef{
			name:        req.NamedValue,
			displayName: req.NamedValue,
			value:       envVarToValue[req.EnvVar],
			secret:      false,
		}
	}
	if len(policyNamedValues) > 0 {
		var keys []string
		for k := range policyNamedValues {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			namedValues = append(namedValues, policyNamedValues[k])
		}
	}

	for _, nv := range namedValues {
		if err := client.CreateOrUpdateNamedValue(nv.name, nv.displayName, nv.value, nv.secret); err != nil {
			return fmt.Errorf("failed to create named value '%s': %w", nv.name, err)
		}
		if nv.secret {
			printSuccess("Created named value: %s (secret)", nv.name)
		} else {
			printSuccess("Created named value: %s", nv.name)
		}
	}

	printInfo("Deploying policy fragments...")

	type fragDef struct {
		id   string
		xml  string
		desc string
	}

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
			if err := client.CreateOrUpdatePolicyFragment(def.id, def.xml, def.desc); err != nil {
				return fmt.Errorf("failed to create fragment '%s': %w", def.id, err)
			}
			printSuccess("Created fragment: %s (%s)", def.id, def.desc)
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
