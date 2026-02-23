package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("hiddenlayer-apim %s\n", Version)
		if verbose {
			fmt.Printf("  Git Commit: %s\n", GitCommit)
			fmt.Printf("  Build Date: %s\n", BuildDate)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
