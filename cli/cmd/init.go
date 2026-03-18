package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a .env configuration file template in the current directory.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing .env file")
	rootCmd.AddCommand(initCmd)
}

const envTemplate = `# HiddenLayer APIM Integration Configuration

# ===== Azure Configuration =====
# Your Azure resource group name
RG=your-resource-group

# Your Azure API Management instance name
APIM_NAME=your-apim-instance

# ===== HiddenLayer Credentials =====
# Get these from the HiddenLayer console (https://console.hiddenlayer.ai)

# HiddenLayer OAuth Client ID
HL_CLIENT=your-client-id

# HiddenLayer OAuth Client Secret
HL_SECRET=your-client-secret

# HiddenLayer Project ID (used by HiddenLayer evaluation endpoints)
HL_PROJECT_ID=your-project-id

# HiddenLayer Tenant ID
HL_TENANT_ID=your-tenant-id

# Optional: OAuth token cache duration (seconds) for APIM internal cache
# Default: 5
# HL_OAUTH_CACHE_SECONDS=5

# ===== Optional =====
# HiddenLayer host suffix for all endpoints (auth.<host>, api.<host>)
# HL_HOST=hiddenlayer.ai

# Default API ID to apply HiddenLayer policy to
# HL_TARGET_API=your-api-id

# Fragment package to use (see available packages with 'deploy --help')
# HL_PACKAGE=v1-interactions
# HL_PACKAGE=v2-request-evals
# HL_PACKAGE=v2-response-evals
`

func runInit(cmd *cobra.Command, args []string) error {
	envFile := ".env"

	if _, err := os.Stat(envFile); err == nil {
		if !initForce {
			return fmt.Errorf(".env file already exists\n\nUse --force to overwrite")
		}
		printWarning("Overwriting existing .env file")
	}

	if err := os.WriteFile(envFile, []byte(envTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	printSuccess("Created .env file")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit .env with your Azure and HiddenLayer configuration")
	fmt.Printf("  2. Run: %s deploy\n", cyan("hiddenlayer-apim"))
	fmt.Printf("  3. Run: %s apply <api-id>\n", cyan("hiddenlayer-apim"))
	fmt.Println()

	return nil
}
