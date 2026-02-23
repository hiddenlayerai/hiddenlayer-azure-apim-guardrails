package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var deployPackage string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy HiddenLayer policy fragments to APIM",
	Long: `Deploy HiddenLayer policy fragments and named values to your
Azure API Management instance.

This creates:
  • Named values for HiddenLayer credentials
  • Policy fragments for OAuth and v1 Interactions input/output evaluation`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().StringVar(&deployPackage, "package", "", "fragment package to deploy (default: auto-select)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if err := cfg.ValidateHLCredentials(); err != nil {
		return err
	}

	pkgFlag := deployPackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return fmt.Errorf("package selection: %w", err)
	}

	printHeader("Deploying HiddenLayer to APIM")
	packageLabel := pkg.Manifest.Name
	if pkg.Manifest.Version != "" {
		packageLabel = fmt.Sprintf("%s@%s", pkg.Manifest.Name, pkg.Manifest.Version)
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

	namedValues := []struct {
		name        string
		displayName string
		value       string
		secret      bool
	}{
		{"hl-client-id", "hl-client-id", cfg.HLClientID, false},
		{"hl-client-secret", "hl-client-secret", cfg.HLClientSecret, true},
		{"hl-project-id", "hl-project-id", cfg.HLProjectID, false},
		{"hl-host", "hl-host", cfg.HLHost, false},
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

	for _, frag := range pkg.Fragments {
		desc := pkg.FragmentDescription(frag.ID)
		if err := client.CreateOrUpdatePolicyFragment(frag.ID, frag.XML, desc); err != nil {
			return fmt.Errorf("failed to create fragment '%s': %w", frag.ID, err)
		}
		printSuccess("Created fragment: %s (%s)", frag.ID, desc)
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
