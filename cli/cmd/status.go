package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

var statusPackage string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check deployment status and verify configuration",
	Long: `Check the status of HiddenLayer deployment in your APIM instance.

This verifies:
  • Azure connectivity
  • HiddenLayer credentials
  • Named values in APIM
  • Policy fragments deployment`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVar(&statusPackage, "package", "", "fragment package to check (default: auto-select)")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	pkgFlag := statusPackage
	if pkgFlag == "" {
		pkgFlag = cfg.HLPackage
	}
	pkg, err := policy.SelectPackageStdin(pkgFlag)
	if err != nil {
		return fmt.Errorf("package selection: %w", err)
	}

	printHeader("HiddenLayer APIM Status")
	packageLabel := pkg.Manifest.Name
	if pkg.Manifest.Version != "" {
		packageLabel = fmt.Sprintf("%s@%s", pkg.Manifest.Name, pkg.Manifest.Version)
	}
	fmt.Printf("Resource Group: %s\n", cfg.ResourceGroup)
	fmt.Printf("APIM Instance:  %s\n", cfg.APIMName)
	fmt.Printf("Package:        %s\n", packageLabel)
	fmt.Println()

	allOk := true

	printInfo("Checking Azure connectivity...")
	client, err := azure.NewClient(cfg.SubscriptionID, cfg.ResourceGroup, cfg.APIMName, verbose)
	if err != nil {
		printError("Azure connection failed: %v", err)
		return nil
	}
	printSuccess("Connected to Azure")

	printInfo("Checking HiddenLayer credentials...")
	if cfg.HLClientID == "" || cfg.HLClientSecret == "" || cfg.HLProjectID == "" {
		printWarning("HiddenLayer credentials not fully configured")
		if cfg.HLClientID == "" {
			fmt.Println("  Missing: HL_CLIENT")
		}
		if cfg.HLClientSecret == "" {
			fmt.Println("  Missing: HL_SECRET")
		}
		if cfg.HLProjectID == "" {
			fmt.Println("  Missing: HL_PROJECT_ID")
		}
		allOk = false
	} else {
		if token, err := testHiddenLayerAuth(cfg.HLHost, cfg.HLClientID, cfg.HLClientSecret); err != nil {
			printError("HiddenLayer auth failed: %v", err)
			allOk = false
		} else {
			printSuccess("HiddenLayer credentials valid (token length: %d)", len(token))
		}
	}

	printInfo("Checking APIM named values...")
	namedValues := []string{"hl-client-id", "hl-client-secret", "hl-project-id", "hl-host"}
	for _, nv := range namedValues {
		value, err := client.GetNamedValue(nv)
		if err != nil {
			printError("Error checking '%s': %v", nv, err)
			allOk = false
		} else if value == "" {
			printWarning("Named value '%s' not found", nv)
			allOk = false
		} else {
			printSuccess("Named value '%s' exists", nv)
		}
	}

	printInfo("Checking policy fragments...")
	fragmentsOk := true
	for _, frag := range pkg.Fragments {
		exists, err := client.GetPolicyFragment(frag.ID)
		if err != nil {
			printError("Error checking '%s': %v", frag.ID, err)
			allOk = false
			fragmentsOk = false
		} else if !exists {
			printWarning("Fragment '%s' not found", frag.ID)
			allOk = false
			fragmentsOk = false
		} else {
			printSuccess("Fragment '%s' exists", frag.ID)
		}
	}

	if fragmentsOk {
		printInfo("Checking API policies...")
		apis, err := client.ListAPIs()
		if err != nil {
			printWarning("Could not list APIs: %v", err)
		} else if len(apis) == 0 {
			printWarning("No APIs found in APIM instance")
		} else {
			protectedAPIs := []string{}
			unprotectedAPIs := []string{}

			for _, api := range apis {
				policyXML, err := client.GetAPIPolicy(api.Name)
				if err != nil {
					continue
				}
				if policy.HasHiddenLayerFragments(policyXML, pkg) {
					protectedAPIs = append(protectedAPIs, api.Name)
				} else {
					unprotectedAPIs = append(unprotectedAPIs, api.Name)
				}
			}

			if len(protectedAPIs) > 0 {
				printSuccess("Protected APIs (%d):", len(protectedAPIs))
				for _, api := range protectedAPIs {
					fmt.Printf("    • %s\n", api)
				}
			}
			if len(unprotectedAPIs) > 0 {
				printWarning("Unprotected APIs (%d):", len(unprotectedAPIs))
				for _, api := range unprotectedAPIs {
					fmt.Printf("    • %s\n", api)
				}
			}
		}
	}

	printHeader("Summary")
	if allOk {
		printSuccess("All checks passed!")
		fmt.Println()
		fmt.Println("HiddenLayer is ready. Apply to an API with:")
		fmt.Printf("  %s apply <api-id>\n", cyan("hiddenlayer-apim"))
	} else {
		printWarning("Some checks failed")
		fmt.Println()
		fmt.Println("To fix, run:")
		fmt.Printf("  %s deploy\n", cyan("hiddenlayer-apim"))
	}
	fmt.Println()

	return nil
}

func testHiddenLayerAuth(host, clientID, clientSecret string) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	if host == "" {
		host = "hiddenlayer.ai"
	}
	resp, err := client.Post(
		fmt.Sprintf("https://auth.%s/oauth2/token", host),
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("authentication failed (status %d)", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return result.AccessToken, nil
}
