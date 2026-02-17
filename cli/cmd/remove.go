package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hiddenlayer/apim-cli/internal/azure"
	"github.com/hiddenlayer/apim-cli/internal/policy"
)

var removePackage string

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
	removeCmd.Flags().StringVar(&removePackage, "package", "", "fragment package to remove (omit to auto-detect)")
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

	pkgFlag := removePackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}

	var newPolicy string
	if pkgFlag != "" {
		pkg, err := policy.LoadPackage(pkgFlag)
		if err != nil {
			return fmt.Errorf("package load: %w", err)
		}
		if !policy.HasHiddenLayerFragments(existingPolicy, pkg) {
			printWarning("No HiddenLayer fragments found in policy")
			return nil
		}
		newPolicy, err = policy.RemoveHiddenLayerFragments(existingPolicy, pkg)
		if err != nil {
			return err
		}
	} else {
		detectedIDs := policy.DetectHiddenLayerFragmentIDs(existingPolicy)
		if len(detectedIDs) == 0 {
			printWarning("No HiddenLayer fragments found in policy")
			return nil
		}
		printInfo("Detected fragments: %s", fmt.Sprintf("%v", detectedIDs))
		newPolicy, err = policy.RemoveHiddenLayerFragmentsByIDs(existingPolicy, detectedIDs)
		if err != nil {
			return err
		}
	}

	printInfo("Removing HiddenLayer fragments...")
	if err := client.SetAPIPolicy(apiID, newPolicy); err != nil {
		return err
	}
	printSuccess("Fragments removed successfully")

	printHeader("Complete")
	fmt.Printf("API '%s' no longer uses HiddenLayer scanning.\n", apiID)
	fmt.Println()
	fmt.Printf("To re-enable: %s apply %s\n", cyan("hiddenlayer-apim"), apiID)
	fmt.Println()

	return nil
}
