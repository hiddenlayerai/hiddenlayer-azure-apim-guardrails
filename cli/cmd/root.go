package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
	verbose bool

	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

var rootCmd = &cobra.Command{
	Use:   "hiddenlayer-apim",
	Short: "HiddenLayer APIM Integration CLI",
	Long: `Deploy and manage HiddenLayer AI security scanning 
in Azure API Management.

This CLI helps you:
  • Deploy HiddenLayer policy fragments to APIM
  • Apply security scanning to specific APIs
  • Monitor and debug the integration`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "init" || cmd.Name() == "version" || cmd.Name() == "export" || cmd.Name() == "bicep" {
			return nil
		}

		var err error
		cfg, err = config.LoadFrom(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .env)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func printSuccess(format string, a ...any) {
	fmt.Printf("%s %s\n", green("✓"), fmt.Sprintf(format, a...))
}

func printError(format string, a ...any) {
	fmt.Printf("%s %s\n", red("✗"), fmt.Sprintf(format, a...))
}

func printWarning(format string, a ...any) {
	fmt.Printf("%s %s\n", yellow("!"), fmt.Sprintf(format, a...))
}

func printInfo(format string, a ...any) {
	fmt.Printf("%s %s\n", cyan("→"), fmt.Sprintf(format, a...))
}

func printHeader(title string) {
	fmt.Println()
	fmt.Println(bold(title))
	fmt.Println(bold("─────────────────────────────────────────"))
}
