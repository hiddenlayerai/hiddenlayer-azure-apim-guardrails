package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var removePackage string
var removePackages []string

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
	removeCmd.Flags().StringSliceVar(&removePackages, "packages", nil, "fragment packages to remove (comma-separated or repeated; cannot be used with --package)")
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

	pkgFlag := removePackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}

	var newPolicy string
	if cmd.Flags().Changed("packages") {
		raw, err := cmd.Flags().GetStringSlice("packages")
		if err != nil {
			return fmt.Errorf("read --packages: %w", err)
		}
		names, err := normalizeUniquePackageNames(raw)
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return fmt.Errorf("no packages specified")
		}

		var pkgs []*policy.Package
		for _, name := range names {
			pkg, err := policy.LoadPackage(name)
			if err != nil {
				return fmt.Errorf("package load %q: %w", name, err)
			}
			pkgs = append(pkgs, pkg)
		}

		present := policy.DetectHiddenLayerFragmentIDs(existingPolicy)
		idsToRemoveSet := map[string]bool{}
		for _, pkg := range pkgs {
			for _, id := range pkg.AllFragmentIDs() {
				idsToRemoveSet[id] = true
			}
		}

		var idsToRemove []string
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for _, id := range pkg.AllFragmentIDs() {
				if idsToRemoveSet[id] && !seen[id] {
					idsToRemove = append(idsToRemove, id)
					seen[id] = true
				}
			}
		}

		anyPresent := false
		for _, id := range present {
			if idsToRemoveSet[id] {
				anyPresent = true
				break
			}
		}
		if !anyPresent {
			printWarning("No HiddenLayer fragments found in policy for specified packages")
			return nil
		}

		remaining := false
		for _, id := range present {
			if !idsToRemoveSet[id] {
				remaining = true
				break
			}
		}

		if remaining {
			shared, err := policy.SharedFragmentIDs()
			if err != nil {
				return fmt.Errorf("compute shared fragments: %w", err)
			}
			filtered := idsToRemove[:0]
			for _, id := range idsToRemove {
				if shared[id] {
					continue
				}
				filtered = append(filtered, id)
			}
			idsToRemove = filtered
		}

		newPolicy, err = policy.RemoveHiddenLayerFragmentsByIDs(existingPolicy, idsToRemove)
		if err != nil {
			return err
		}
	} else if pkgFlag != "" {
		pkg, err := policy.LoadPackage(pkgFlag)
		if err != nil {
			return fmt.Errorf("package load: %w", err)
		}
		if !policy.HasHiddenLayerFragments(existingPolicy, pkg) {
			printWarning("No HiddenLayer fragments found in policy")
			return nil
		}

		idsToRemove := pkg.AllFragmentIDs()
		present := policy.DetectHiddenLayerFragmentIDs(existingPolicy)
		pkgIDs := map[string]bool{}
		for _, id := range idsToRemove {
			pkgIDs[id] = true
		}
		otherFragmentsRemain := false
		for _, id := range present {
			if !pkgIDs[id] {
				otherFragmentsRemain = true
				break
			}
		}
		if otherFragmentsRemain {
			// Preserve shared fragments (e.g., hl-oauth-token-management) so removing one
			// package does not break other installed HiddenLayer fragments.
			shared, err := policy.SharedFragmentIDs()
			if err != nil {
				return fmt.Errorf("compute shared fragments: %w", err)
			}
			filtered := make([]string, 0, len(idsToRemove))
			for _, id := range idsToRemove {
				if shared[id] {
					continue
				}
				filtered = append(filtered, id)
			}
			idsToRemove = filtered
		}

		newPolicy, err = policy.RemoveHiddenLayerFragmentsByIDs(existingPolicy, idsToRemove)
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
