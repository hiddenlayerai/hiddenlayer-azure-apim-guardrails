package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveExportGroup(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default empty", input: "", want: ExportGroupAll},
		{name: "inbound", input: "inbound", want: ExportGroupInbound},
		{name: "outbound uppercase", input: "OUTBOUND", want: ExportGroupOutbound},
		{name: "all", input: "all", want: ExportGroupAll},
		{name: "invalid", input: "foo", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveExportGroup(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("ResolveExportGroup() expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveExportGroup() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ResolveExportGroup() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFragmentIDsForGroup(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	inboundIDs, err := FragmentIDsForGroup(pkg, ExportGroupInbound)
	if err != nil {
		t.Fatalf("FragmentIDsForGroup() error = %v", err)
	}
	if len(inboundIDs) != 2 {
		t.Fatalf("expected 2 inbound ids, got %d", len(inboundIDs))
	}

	outboundIDs, err := FragmentIDsForGroup(pkg, ExportGroupOutbound)
	if err != nil {
		t.Fatalf("FragmentIDsForGroup() error = %v", err)
	}
	if len(outboundIDs) != 1 {
		t.Fatalf("expected 1 outbound id, got %d", len(outboundIDs))
	}
}

func TestExportPackageBicep_All(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	outDir := t.TempDir()

	if err := ExportPackageBicep(pkg, outDir, ExportGroupAll); err != nil {
		t.Fatalf("ExportPackageBicep() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "main.bicep")); err != nil {
		t.Fatalf("main.bicep not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "main.bicepparam")); err != nil {
		t.Fatalf("main.bicepparam not found: %v", err)
	}

	fragmentFiles, err := os.ReadDir(filepath.Join(outDir, "fragments"))
	if err != nil {
		t.Fatalf("failed to read fragments dir: %v", err)
	}
	if len(fragmentFiles) != 3 {
		t.Fatalf("expected 3 fragment files, got %d", len(fragmentFiles))
	}

	bicepRaw, err := os.ReadFile(filepath.Join(outDir, "main.bicep"))
	if err != nil {
		t.Fatalf("failed to read main.bicep: %v", err)
	}
	bicep := string(bicepRaw)
	for _, want := range []string{
		"Package: v1-interactions@1.0.0",
		"loadTextContent('fragments/hl-oauth-token-management.xml')",
		"loadTextContent('fragments/hl-interactions-input.xml')",
		"loadTextContent('fragments/hl-interactions-output.xml')",
	} {
		if !strings.Contains(bicep, want) {
			t.Fatalf("main.bicep missing %q", want)
		}
	}
}

func TestExportPackageBicep_GroupFiltering(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	outDir := t.TempDir()

	if err := ExportPackageBicep(pkg, outDir, ExportGroupInbound); err != nil {
		t.Fatalf("ExportPackageBicep() error = %v", err)
	}

	fragmentFiles, err := os.ReadDir(filepath.Join(outDir, "fragments"))
	if err != nil {
		t.Fatalf("failed to read fragments dir: %v", err)
	}
	if len(fragmentFiles) != 2 {
		t.Fatalf("expected 2 fragment files, got %d", len(fragmentFiles))
	}

	bicepRaw, err := os.ReadFile(filepath.Join(outDir, "main.bicep"))
	if err != nil {
		t.Fatalf("failed to read main.bicep: %v", err)
	}
	bicep := string(bicepRaw)
	if strings.Contains(bicep, "hl-interactions-output.xml") {
		t.Fatal("main.bicep should not contain outbound fragment in inbound export")
	}
}

func TestExportPackageBicep_MissingVersion(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	pkg.Manifest.Version = ""

	err = ExportPackageBicep(pkg, t.TempDir(), ExportGroupAll)
	if err == nil {
		t.Fatal("ExportPackageBicep() expected error when version is missing")
	}
}

func TestFragmentDescription_WithoutVersion(t *testing.T) {
	got := fragmentDescription("my-package", "", "frag-id")
	want := "HiddenLayer my-package fragment frag-id"
	if got != want {
		t.Fatalf("fragmentDescription() = %q, want %q", got, want)
	}
}
