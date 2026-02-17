package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/hiddenlayer/apim-cli/internal/azure"
	"github.com/hiddenlayer/apim-cli/internal/policy"
)

var (
	applyDryRun  bool
	applyPackage string
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
	applyCmd.Flags().StringVar(&applyPackage, "package", "", "fragment package to use (default: auto-select)")
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

	pkgFlag := applyPackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return fmt.Errorf("package selection: %w", err)
	}

	printHeader("Applying HiddenLayer Policy")
	packageLabel := pkg.Manifest.Name
	if pkg.Manifest.Version != "" {
		packageLabel = fmt.Sprintf("%s@%s", pkg.Manifest.Name, pkg.Manifest.Version)
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
	for _, frag := range pkg.Fragments {
		exists, err := waitForPolicyFragment(client, frag.ID, 5, 2*time.Second)
		if err != nil {
			return fmt.Errorf("error checking fragment '%s': %w", frag.ID, err)
		}
		if !exists {
			return fmt.Errorf("missing fragment '%s'\n\nRun 'hiddenlayer-apim deploy' first", frag.ID)
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

	if policy.HasHiddenLayerFragments(existingPolicy, pkg) {
		printWarning("HiddenLayer fragments already present in policy")
		printInfo("Use 'remove' first if you want to re-apply")
		return nil
	}

	newPolicy, err := policy.InjectHiddenLayerFragments(existingPolicy, pkg)
	if err != nil {
		return err
	}

	if applyDryRun {
		printHeader("Policy Preview (Dry Run)")
		fmt.Println(newPolicy)
		fmt.Println()
		printInfo("To apply this policy, run without --dry-run")
		return nil
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
