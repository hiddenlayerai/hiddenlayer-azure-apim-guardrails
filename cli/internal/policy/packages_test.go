package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPackages(t *testing.T) {
	names, err := ListPackages()
	if err != nil {
		t.Fatalf("ListPackages() error = %v", err)
	}
	if len(names) < 1 {
		t.Fatalf("expected at least 1 package, got %d: %v", len(names), names)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["v1-interactions"] {
		t.Errorf("ListPackages() missing %q, got %v", "v1-interactions", names)
	}
	if !found["v2-request-evals"] {
		t.Errorf("ListPackages() missing %q, got %v", "v2-request-evals", names)
	}
	if !found["v2-response-evals"] {
		t.Errorf("ListPackages() missing %q, got %v", "v2-response-evals", names)
	}
	if found["test-fixture"] {
		t.Errorf("ListPackages() should not contain test-fixture; test fixtures belong in testdata/")
	}
}

func TestSharedFragmentIDs_IncludesOAuth(t *testing.T) {
	shared, err := SharedFragmentIDs()
	if err != nil {
		t.Fatalf("SharedFragmentIDs() error = %v", err)
	}
	if !shared["hl-oauth-token-management"] {
		t.Fatal("expected hl-oauth-token-management to be shared across packages")
	}
}

func TestLoadPackage_V1Interactions(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	if pkg.Manifest.Name != "v1-interactions" {
		t.Errorf("Manifest.Name = %q, want %q", pkg.Manifest.Name, "v1-interactions")
	}
	if pkg.Manifest.Version != "1.0.0" {
		t.Errorf("Manifest.Version = %q, want %q", pkg.Manifest.Version, "1.0.0")
	}
	if len(pkg.Fragments) != 3 {
		t.Errorf("expected 3 fragments, got %d", len(pkg.Fragments))
	}
}

func TestLoadPackage_TestFixture(t *testing.T) {
	// Load the test fixture from testdata/ (not the embedded FS) to verify
	// manifest parsing and fragment loading without shipping test data in the binary.
	fixtureDir := filepath.Join("testdata", "test-fixture")

	data, err := os.ReadFile(filepath.Join(fixtureDir, "package.json"))
	if err != nil {
		t.Fatalf("reading test fixture manifest: %v", err)
	}
	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parsing test fixture manifest: %v", err)
	}
	if manifest.Name != "test-fixture" {
		t.Errorf("Manifest.Name = %q, want %q", manifest.Name, "test-fixture")
	}
	if manifest.Version != "0.0.1" {
		t.Errorf("Manifest.Version = %q, want %q", manifest.Version, "0.0.1")
	}
	if len(manifest.Inbound) != 1 {
		t.Errorf("expected 1 inbound fragment, got %d", len(manifest.Inbound))
	}

	// Load fragments from the fixture directory.
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("reading fixture directory: %v", err)
	}
	var fragments []Fragment
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		xmlData, err := os.ReadFile(filepath.Join(fixtureDir, e.Name()))
		if err != nil {
			t.Fatalf("reading fragment %q: %v", e.Name(), err)
		}
		id := strings.TrimSuffix(e.Name(), ".xml")
		fragments = append(fragments, Fragment{ID: id, XML: string(xmlData)})
	}
	if len(fragments) != 1 {
		t.Errorf("expected 1 fragment file, got %d", len(fragments))
	}
	if fragments[0].ID != "hl-test-fragment" {
		t.Errorf("fragment ID = %q, want %q", fragments[0].ID, "hl-test-fragment")
	}
}

func TestLoadPackage_NotFound(t *testing.T) {
	_, err := LoadPackage("does-not-exist")
	if err == nil {
		t.Error("LoadPackage() expected error for non-existent package")
	}
}

func TestLoadPackageManifest(t *testing.T) {
	m, err := LoadPackageManifest("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackageManifest() error = %v", err)
	}
	if m.Name != "v1-interactions" {
		t.Errorf("Name = %q, want %q", m.Name, "v1-interactions")
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", m.Version, "1.0.0")
	}
	if len(m.Inbound) != 2 {
		t.Errorf("expected 2 inbound fragments, got %d", len(m.Inbound))
	}
	if len(m.Outbound) != 1 {
		t.Errorf("expected 1 outbound fragment, got %d", len(m.Outbound))
	}
}

func TestPackage_AllFragmentIDs(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	ids := pkg.AllFragmentIDs()
	if len(ids) != 3 {
		t.Errorf("AllFragmentIDs() returned %d, want 3", len(ids))
	}
}

func TestPackage_InboundOutboundIDs(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	if len(pkg.InboundIDs()) != 2 {
		t.Errorf("InboundIDs() returned %d, want 2", len(pkg.InboundIDs()))
	}
	if len(pkg.OutboundIDs()) != 1 {
		t.Errorf("OutboundIDs() returned %d, want 1", len(pkg.OutboundIDs()))
	}
}

func TestPackage_GetFragmentXML(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}

	xml, ok := pkg.GetFragmentXML("hl-oauth-token-management")
	if !ok {
		t.Fatal("GetFragmentXML() returned false for hl-oauth-token-management")
	}
	if xml == "" {
		t.Error("GetFragmentXML() returned empty XML")
	}

	_, ok = pkg.GetFragmentXML("nonexistent")
	if ok {
		t.Error("GetFragmentXML() should return false for non-existent fragment")
	}
}

func TestPackage_NoDuplicateIDs(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	ids := pkg.AllFragmentIDs()
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate fragment ID: %s", id)
		}
		seen[id] = true
	}
}

func TestPackage_FragmentDescription(t *testing.T) {
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	got := pkg.FragmentDescription("hl-interactions-input")
	want := "HiddenLayer v1-interactions@1.0.0 fragment hl-interactions-input"
	if got != want {
		t.Fatalf("FragmentDescription() = %q, want %q", got, want)
	}
}

func TestPackage_FragmentDescription_WithoutVersion(t *testing.T) {
	pkg := &Package{
		Manifest: PackageManifest{Name: "my-package"},
	}
	got := pkg.FragmentDescription("frag-id")
	want := "HiddenLayer my-package fragment frag-id"
	if got != want {
		t.Fatalf("FragmentDescription() = %q, want %q", got, want)
	}
}

func TestPackageManifestJSON_WithVersion(t *testing.T) {
	const raw = `{"name":"pkg-a","version":"2.3.4","description":"d","inbound":[],"outbound":[]}`
	var m PackageManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if m.Version != "2.3.4" {
		t.Errorf("Version = %q, want %q", m.Version, "2.3.4")
	}
}

func TestPackageManifestJSON_WithoutVersion(t *testing.T) {
	const raw = `{"name":"pkg-a","description":"d","inbound":[],"outbound":[]}`
	var m PackageManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if m.Version != "" {
		t.Errorf("Version = %q, want empty string", m.Version)
	}
}

func mustGetFragmentXML(t *testing.T, packageName, fragmentID string) string {
	t.Helper()

	pkg, err := LoadPackage(packageName)
	if err != nil {
		t.Fatalf("LoadPackage(%q) error = %v", packageName, err)
	}
	xml, ok := pkg.GetFragmentXML(fragmentID)
	if !ok {
		t.Fatalf("GetFragmentXML(%q) returned false", fragmentID)
	}
	return xml
}

func assertContainsAll(t *testing.T, xml, fragmentID string, values []string) {
	t.Helper()

	for _, value := range values {
		if !strings.Contains(xml, value) {
			t.Errorf("fragment %q missing %q", fragmentID, value)
		}
	}
}

func assertOrder(t *testing.T, xml, fragmentID, first, second string) {
	t.Helper()

	firstIdx := strings.Index(xml, first)
	if firstIdx == -1 {
		t.Fatalf("fragment %q missing %q", fragmentID, first)
	}
	secondIdx := strings.Index(xml, second)
	if secondIdx == -1 {
		t.Fatalf("fragment %q missing %q", fragmentID, second)
	}
	if firstIdx >= secondIdx {
		t.Errorf("fragment %q should place %q before %q", fragmentID, first, second)
	}
}

func TestEvalFragments_ContainProviderVersionHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v2-response-evals", "hl-v2-response-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
		{"v3-response-evals", "hl-v3-response-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			assertContainsAll(t, xml, tt.fragmentID, []string{
				"HL-Runtime-Edge-Provider-Version",
				"<value>0.1</value>",
			})
		})
	}
}

func TestEvalFragments_ContainRuntimeEdgeProviderHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v2-response-evals", "hl-v2-response-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
		{"v3-response-evals", "hl-v3-response-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			assertContainsAll(t, xml, tt.fragmentID, []string{
				`<set-header name="HL-Runtime-Edge-Provider" exists-action="override">`,
				`<value>@((string)context.Variables.GetValueOrDefault("hl_runtime_edge_provider", "azure-apim"))</value>`,
			})
			if strings.Contains(xml, "hl-runtime-edge-provider") {
				t.Errorf("fragment %q should not use lowercase hl-runtime-edge-provider header name", tt.fragmentID)
			}
			if strings.Contains(xml, "HL-Provider") {
				t.Errorf("fragment %q should not reference HL-Provider", tt.fragmentID)
			}
		})
	}
}

func TestEvalFragments_ContainMetadataHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v2-response-evals", "hl-v2-response-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
		{"v3-response-evals", "hl-v3-response-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			expectedFields := []string{
				"HL-Runtime-Edge-Provider-Metadata",
				"context.Api.Name",
				"context.Api.Version",
				"context.Deployment.ServiceName",
				"context.Deployment.Region",
				"context.Api.Id",
				"context.Api.Revision",
				"context.Operation",
			}
			assertContainsAll(t, xml, tt.fragmentID, expectedFields)
		})
	}
}

func TestEvalFragments_ContainSessionIdHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v2-response-evals", "hl-v2-response-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
		{"v3-response-evals", "hl-v3-response-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			assertContainsAll(t, xml, tt.fragmentID, []string{
				`<set-header name="Hl-Runtime-Session-Id" exists-action="override">`,
				`<value>@((string)context.Variables.GetValueOrDefault("hl_runtime_session_id", ""))</value>`,
			})
			if strings.Contains(xml, "context.RequestId") {
				t.Errorf("fragment %q should not fall back to context.RequestId", tt.fragmentID)
			}
		})
	}
}

func TestRequestEvalFragments_CaptureThenDeleteSessionIdHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			capture := `context.Variables.GetValueOrDefault("hl_runtime_session_id", "")`
			headerFallback := `context.Request.Headers.GetValueOrDefault("Hl-Runtime-Session-Id", "")`
			deleteHeader := `<set-header name="Hl-Runtime-Session-Id" exists-action="delete" />`
			assertContainsAll(t, xml, tt.fragmentID, []string{capture, headerFallback, deleteHeader})
			assertOrder(t, xml, tt.fragmentID, capture, headerFallback)
		})
	}
}

func TestRequestEvalFragments_CaptureThenDeleteRuntimeEdgeProviderHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-request-evals", "hl-v2-request-evaluations"},
		{"v3-request-evals", "hl-v3-request-evaluations"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			capture := `context.Variables.GetValueOrDefault("hl_runtime_edge_provider", "")`
			headerFallback := `context.Request.Headers.GetValueOrDefault("HL-Runtime-Edge-Provider", "azure-apim")`
			deleteHeader := `<set-header name="HL-Runtime-Edge-Provider" exists-action="delete" />`
			assertContainsAll(t, xml, tt.fragmentID, []string{capture, headerFallback, deleteHeader})
			assertOrder(t, xml, tt.fragmentID, capture, headerFallback)
		})
	}
}

func TestV3RequestEvalFragment_PreservesCapturedProjectID(t *testing.T) {
	xml := mustGetFragmentXML(t, "v3-request-evals", "hl-v3-request-evaluations")
	assertContainsAll(t, xml, "hl-v3-request-evaluations", []string{
		`context.Variables.GetValueOrDefault("hl_project_id", "")`,
		`context.Request.Headers.GetValueOrDefault("HL-Project-Id", "")`,
	})
	assertOrder(t, xml, "hl-v3-request-evaluations", `context.Variables.GetValueOrDefault("hl_project_id", "")`, `context.Request.Headers.GetValueOrDefault("HL-Project-Id", "")`)
}

func TestResponseEvalInboundFragments_CaptureThenDeleteSessionIdHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-response-evals", "hl-v2-response-evals-inbound"},
		{"v3-response-evals", "hl-v3-response-evals-inbound"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			capture := `<set-variable name="hl_runtime_session_id" value='@(context.Request.Headers.GetValueOrDefault("Hl-Runtime-Session-Id", ""))' />`
			deleteHeader := `<set-header name="Hl-Runtime-Session-Id" exists-action="delete" />`
			assertContainsAll(t, xml, tt.fragmentID, []string{capture, deleteHeader})
			assertOrder(t, xml, tt.fragmentID, capture, deleteHeader)
		})
	}
}

func TestResponseEvalInboundFragments_CaptureThenDeleteRuntimeEdgeProviderHeader(t *testing.T) {
	tests := []struct {
		pkg        string
		fragmentID string
	}{
		{"v2-response-evals", "hl-v2-response-evals-inbound"},
		{"v3-response-evals", "hl-v3-response-evals-inbound"},
	}
	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			xml := mustGetFragmentXML(t, tt.pkg, tt.fragmentID)
			capture := `<set-variable name="hl_runtime_edge_provider" value='@(context.Request.Headers.GetValueOrDefault("HL-Runtime-Edge-Provider", "azure-apim"))' />`
			deleteHeader := `<set-header name="HL-Runtime-Edge-Provider" exists-action="delete" />`
			assertContainsAll(t, xml, tt.fragmentID, []string{capture, deleteHeader})
			assertOrder(t, xml, tt.fragmentID, capture, deleteHeader)
		})
	}
}
