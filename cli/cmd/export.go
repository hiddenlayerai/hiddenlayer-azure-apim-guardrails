package cmd

import "github.com/spf13/cobra"

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export package artifacts for external deployment",
	Long: `Export package artifacts that can be edited and deployed outside the CLI.

Use subcommands to choose an export format, such as Bicep.`,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
