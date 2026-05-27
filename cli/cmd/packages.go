package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var packagesCmd = &cobra.Command{
	Use:     "packages",
	Aliases: []string{"list-packages", "package-list"},
	Short:   "List available fragment packages",
	Long: `List embedded HiddenLayer fragment packages that can be used with
deploy/apply/status/export package flags.`,
	RunE: runPackages,
}

func init() {
	rootCmd.AddCommand(packagesCmd)
}

func runPackages(cmd *cobra.Command, args []string) error {
	names, err := policy.ListPackages()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return fmt.Errorf("no fragment packages found")
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Available fragment packages (use with --package or --packages):")
	for _, name := range names {
		m, err := policy.LoadPackageManifest(name)
		if err != nil {
			return fmt.Errorf("load package manifest %q: %w", name, err)
		}

		fmt.Fprintf(out, "  name: %s\n", m.Name)
		if m.Version != "" {
			fmt.Fprintf(out, "    version: %s\n", m.Version)
		}
		if m.Description != "" {
			fmt.Fprintf(out, "    description: %s\n", m.Description)
		}
		if len(m.Inbound) > 0 {
			fmt.Fprintf(out, "    inbound:  %s\n", strings.Join(m.Inbound, ", "))
		}
		if len(m.Outbound) > 0 {
			fmt.Fprintf(out, "    outbound: %s\n", strings.Join(m.Outbound, ", "))
		}
	}

	return nil
}
