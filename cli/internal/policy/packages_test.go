package policy

import (
	"encoding/json"
	"testing"
)

func TestListPackages(t *testing.T) {
	names, err := ListPackages()
	if err != nil {
		t.Fatalf("ListPackages() error = %v", err)
	}
	if len(names) < 2 {
		t.Fatalf("expected at least 2 packages (v1-interactions, test-fixture), got %d: %v", len(names), names)
	}

	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	for _, want := range []string{"v1-interactions", "test-fixture"} {
		if !found[want] {
			t.Errorf("ListPackages() missing %q, got %v", want, names)
		}
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
	pkg, err := LoadPackage("test-fixture")
	if err != nil {
		t.Fatalf("LoadPackage() error = %v", err)
	}
	if pkg.Manifest.Name != "test-fixture" {
		t.Errorf("Manifest.Name = %q, want %q", pkg.Manifest.Name, "test-fixture")
	}
	if pkg.Manifest.Version != "0.0.1" {
		t.Errorf("Manifest.Version = %q, want %q", pkg.Manifest.Version, "0.0.1")
	}
	if len(pkg.Fragments) != 1 {
		t.Errorf("expected 1 fragment, got %d", len(pkg.Fragments))
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
