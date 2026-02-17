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

func TestSelectPackage_InteractiveMenu(t *testing.T) {
	// Simulate selecting option 1 from the menu
	input := strings.NewReader("1\n")
	var output bytes.Buffer

	pkg, err := SelectPackage("", input, &output)
	if err != nil {
		t.Fatalf("SelectPackage() error = %v", err)
	}
	if pkg == nil {
		t.Fatal("SelectPackage() returned nil package")
	}

	// Verify menu was displayed
	out := output.String()
	if !strings.Contains(out, "Available fragment packages") {
		t.Error("expected menu header in output")
	}
	if !strings.Contains(out, "@1.0.0") {
		t.Error("expected version metadata in package menu output")
	}
}

func TestSelectPackage_InteractiveDefaultSelection(t *testing.T) {
	// Empty input should default to "1"
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

func TestSelectPackage_InvalidInput(t *testing.T) {
	input := strings.NewReader("abc\n")
	var output bytes.Buffer

	_, err := SelectPackage("", input, &output)
	if err == nil {
		t.Error("SelectPackage() expected error for invalid input")
	}
}

func TestSelectPackage_OutOfRange(t *testing.T) {
	input := strings.NewReader("999\n")
	var output bytes.Buffer

	_, err := SelectPackage("", input, &output)
	if err == nil {
		t.Error("SelectPackage() expected error for out-of-range selection")
	}
}
