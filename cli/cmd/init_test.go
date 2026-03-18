package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit_WritesTenantIDToTemplate(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	initForce = false
	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempDir, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}

	if !strings.Contains(string(data), "HL_TENANT_ID=your-tenant-id") {
		t.Fatal("expected generated .env template to include HL_TENANT_ID")
	}
}
