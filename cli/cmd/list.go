package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List APIs in the APIM instance",
	Long:  `List all APIs available in your Azure API Management instance.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	printHeader("APIs in " + cfg.APIMName)

	client, err := azure.NewClient(cfg.SubscriptionID, cfg.ResourceGroup, cfg.APIMName, verbose)
	if err != nil {
		return err
	}

	apis, err := client.ListAPIs()
	if err != nil {
		return err
	}

	if len(apis) == 0 {
		printWarning("No APIs found in this APIM instance")
		return nil
	}

	// Find max lengths for formatting
	maxNameLen := 10
	maxDisplayLen := 12
	for _, api := range apis {
		if len(api.Name) > maxNameLen {
			maxNameLen = len(api.Name)
		}
		if len(api.DisplayName) > maxDisplayLen {
			maxDisplayLen = len(api.DisplayName)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %s\n", 
		maxNameLen, "API ID", 
		maxDisplayLen, "Display Name",
		"Path")
	fmt.Printf("%s  %s  %s\n",
		strings.Repeat("─", maxNameLen),
		strings.Repeat("─", maxDisplayLen),
		strings.Repeat("─", 20))

	// Print APIs
	for _, api := range apis {
		fmt.Printf("%-*s  %-*s  /%s\n",
			maxNameLen, api.Name,
			maxDisplayLen, api.DisplayName,
			api.Path)
	}

	fmt.Println()
	fmt.Printf("Total: %d APIs\n", len(apis))
	fmt.Println()
	fmt.Println("To apply HiddenLayer policy:")
	fmt.Printf("  %s apply <api-id>\n", cyan("hiddenlayer-apim"))
	fmt.Println()

	return nil
}
