package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	// Azure configuration
	ResourceGroup  string
	APIMName       string
	SubscriptionID string

	// HiddenLayer configuration
	HLClientID          string
	HLClientSecret      string
	HLProjectID         string
	HLTenantID          string
	HLHost              string
	HLOAuthCacheSeconds string

	TargetAPI string
	HLPackage string
}

func Load() (*Config, error) {
	envPaths := []string{
		".env",
		filepath.Join(os.Getenv("HOME"), ".hiddenlayer", ".env"),
	}

	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			break
		}
	}

	cfg := buildConfigFromEnv()
	cfg.normalizeConfig()
	return cfg, nil
}

func LoadFrom(path string) (*Config, error) {
	if path == "" {
		return Load()
	}

	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	if err := godotenv.Overload(path); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	cfg := buildConfigFromEnv()
	cfg.normalizeConfig()
	return cfg, nil
}

func buildConfigFromEnv() *Config {
	return &Config{
		ResourceGroup:       os.Getenv("RG"),
		APIMName:            os.Getenv("APIM_NAME"),
		SubscriptionID:      os.Getenv("AZURE_SUBSCRIPTION_ID"),
		HLClientID:          os.Getenv("HL_CLIENT"),
		HLClientSecret:      os.Getenv("HL_SECRET"),
		HLProjectID:         os.Getenv("HL_PROJECT_ID"),
		HLTenantID:          os.Getenv("HL_TENANT_ID"),
		HLHost:              os.Getenv("HL_HOST"),
		HLOAuthCacheSeconds: os.Getenv("HL_OAUTH_CACHE_SECONDS"),
		TargetAPI:           os.Getenv("HL_TARGET_API"),
		HLPackage:           os.Getenv("HL_PACKAGE"),
	}
}

func (c *Config) normalizeConfig() {
	if c.HLHost == "" {
		c.HLHost = "hiddenlayer.ai"
	}
	if c.HLOAuthCacheSeconds == "" {
		c.HLOAuthCacheSeconds = "5"
	}
}

func (c *Config) Validate() error {
	if c.ResourceGroup == "" {
		return fmt.Errorf("resource_group (RG) is required")
	}
	if c.APIMName == "" {
		return fmt.Errorf("apim_name (APIM_NAME) is required")
	}
	return nil
}

func (c *Config) ValidateHLCredentials() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.HLClientID == "" {
		return fmt.Errorf("hl_client_id (HL_CLIENT) is required")
	}
	if c.HLClientSecret == "" {
		return fmt.Errorf("hl_client_secret (HL_SECRET) is required")
	}
	if c.HLProjectID == "" {
		return fmt.Errorf("hl_project_id (HL_PROJECT_ID) is required")
	}
	if c.HLTenantID == "" {
		return fmt.Errorf("hl_tenant_id (HL_TENANT_ID) is required")
	}
	return nil
}
