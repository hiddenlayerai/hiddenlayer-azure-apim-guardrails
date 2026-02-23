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
	if found["test-fixture"] {
		t.Errorf("ListPackages() should not contain test-fixture; test fixtures belong in testdata/")
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
