package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hiddenlayer/apim-cli/internal/policy"
)

var (
	exportBicepPackage string
	exportBicepGroup   string
	exportBicepOut     string
)

var exportBicepCmd = &cobra.Command{
	Use:   "bicep",
	Short: "Export package fragments as a Bicep deployment bundle",
	Long: `Export a package as a Bicep bundle that includes:
  • main.bicep
  • main.bicepparam
  • fragments/*.xml

The generated bundle can be committed to source control and deployed with Azure CLI.`,
	RunE: runExportBicep,
}

func init() {
	exportBicepCmd.Flags().StringVar(&exportBicepPackage, "package", "", "fragment package to export (default: auto-select)")
	exportBicepCmd.Flags().StringVar(&exportBicepGroup, "group", policy.ExportGroupAll, "fragment group to export: inbound, outbound, or all")
	exportBicepCmd.Flags().StringVar(&exportBicepOut, "out", "", "output directory (default: ./export/<package>/<group>)")
	exportCmd.AddCommand(exportBicepCmd)
}

func runExportBicep(cmd *cobra.Command, args []string) error {
	pkg, err := policy.SelectPackageStdin(exportBicepPackage)
	if err != nil {
		return fmt.Errorf("package selection: %w", err)
	}

	group, err := policy.ResolveExportGroup(exportBicepGroup)
	if err != nil {
		return err
	}

	outDir := exportBicepOut
	if outDir == "" {
		outDir = filepath.Join("export", pkg.Manifest.Name, group)
	}

	if err := policy.ExportPackageBicep(pkg, outDir, group); err != nil {
		return err
	}

	printHeader("Export Complete")
	fmt.Printf("Package:  %s", pkg.Manifest.Name)
	if pkg.Manifest.Version != "" {
		fmt.Printf("@%s", pkg.Manifest.Version)
	}
	fmt.Println()
	fmt.Printf("Group:    %s\n", group)
	fmt.Printf("Output:   %s\n", outDir)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit fragments/*.xml as needed")
	fmt.Println("  2. Deploy with: az deployment group create --resource-group <rg> --template-file main.bicep --parameters main.bicepparam")
	fmt.Println()

	return nil
}
