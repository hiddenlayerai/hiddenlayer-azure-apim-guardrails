package policy

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectPackage_ExplicitFlag(t *testing.T) {
	pkg, err := SelectPackage("v1-interactions", nil, nil)
	if err != nil {
		t.Fatalf("SelectPackage() error = %v", err)
	}
	if pkg.Manifest.Name != "v1-interactions" {
		t.Errorf("expected v1-interactions, got %s", pkg.Manifest.Name)
	}
}

func TestSelectPackage_InvalidFlag(t *testing.T) {
	_, err := SelectPackage("nonexistent", nil, nil)
	if err == nil {
		t.Error("SelectPackage() expected error for non-existent package")
	}
}

func TestSelectFromMenu_ValidChoice(t *testing.T) {
	entries := []menuEntry{
		{Name: "pkg-a", Version: "1.0.0", Desc: "Package A"},
		{Name: "pkg-b", Version: "2.0.0", Desc: "Package B"},
	}
	input := strings.NewReader("1\n")
	var output bytes.Buffer

	name, err := selectFromMenu(entries, input, &output)
	if err != nil {
		t.Fatalf("selectFromMenu() error = %v", err)
	}
	if name != "pkg-a" {
		t.Errorf("expected %q, got %q", "pkg-a", name)
	}

	out := output.String()
	if !strings.Contains(out, "Available fragment packages") {
		t.Error("expected menu header in output")
	}
	if !strings.Contains(out, "@1.0.0") {
		t.Error("expected version metadata in package menu output")
	}
}

func TestSelectFromMenu_SecondChoice(t *testing.T) {
	entries := []menuEntry{
		{Name: "pkg-a", Version: "1.0.0", Desc: "Package A"},
		{Name: "pkg-b", Version: "2.0.0", Desc: "Package B"},
	}
	input := strings.NewReader("2\n")
	var output bytes.Buffer

	name, err := selectFromMenu(entries, input, &output)
	if err != nil {
		t.Fatalf("selectFromMenu() error = %v", err)
	}
	if name != "pkg-b" {
		t.Errorf("expected %q, got %q", "pkg-b", name)
	}
}

func TestSelectFromMenu_DefaultSelection(t *testing.T) {
	entries := []menuEntry{
		{Name: "pkg-a", Version: "1.0.0", Desc: "Package A"},
		{Name: "pkg-b", Version: "2.0.0", Desc: "Package B"},
	}
	input := strings.NewReader("\n")
	var output bytes.Buffer

	name, err := selectFromMenu(entries, input, &output)
	if err != nil {
		t.Fatalf("selectFromMenu() error = %v", err)
	}
	if name != "pkg-a" {
		t.Errorf("expected default %q, got %q", "pkg-a", name)
	}
}

func TestSelectPackage_InteractiveDefaultSelection(t *testing.T) {
	// With a single embedded package, SelectPackage auto-selects it.
	input := strings.NewReader("\n")
	var output bytes.Buffer

	pkg, err := SelectPackage("", input, &output)
	if err != nil {
		t.Fatalf("SelectPackage() error = %v", err)
	}
	if pkg == nil {
		t.Fatal("SelectPackage() returned nil package")
	}
}

func TestSelectFromMenu_InvalidInput(t *testing.T) {
	entries := []menuEntry{
		{Name: "pkg-a", Version: "1.0.0", Desc: "Package A"},
	}
	input := strings.NewReader("abc\n")
	var output bytes.Buffer

	_, err := selectFromMenu(entries, input, &output)
	if err == nil {
		t.Error("selectFromMenu() expected error for invalid input")
	}
}

func TestSelectFromMenu_OutOfRange(t *testing.T) {
	entries := []menuEntry{
		{Name: "pkg-a", Version: "1.0.0", Desc: "Package A"},
	}
	input := strings.NewReader("999\n")
	var output bytes.Buffer

	_, err := selectFromMenu(entries, input, &output)
	if err == nil {
		t.Error("selectFromMenu() expected error for out-of-range selection")
	}
}
