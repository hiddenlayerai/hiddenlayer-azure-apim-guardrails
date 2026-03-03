package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNormalizeUniquePackageNames_DedupesAndErrors(t *testing.T) {
	if _, err := normalizeUniquePackageNames([]string{"v1-interactions", "v1-interactions"}); err == nil {
		t.Fatal("expected duplicate package error")
	}
	if got, err := normalizeUniquePackageNames([]string{"  v1-interactions  ", "", "v2-beta-request-evals"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if len(got) != 2 || got[0] != "v1-interactions" || got[1] != "v2-beta-request-evals" {
		t.Fatalf("unexpected normalized result: %v", got)
	}
}

func TestResolveApplyPackages_LoadsMultiple(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "v1-interactions,v2-beta-request-evals"); err != nil {
		t.Fatalf("set packages: %v", err)
	}

	pkgs, err := resolveApplyPackages(cmd)
	if err != nil {
		t.Fatalf("resolveApplyPackages error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Manifest.Name != "v1-interactions" {
		t.Fatalf("pkg[0] = %q, want %q", pkgs[0].Manifest.Name, "v1-interactions")
	}
	if pkgs[1].Manifest.Name != "v2-beta-request-evals" {
		t.Fatalf("pkg[1] = %q, want %q", pkgs[1].Manifest.Name, "v2-beta-request-evals")
	}
}

func TestResolveApplyPackages_ErrorsOnUnknownPackage(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "does-not-exist"); err != nil {
		t.Fatalf("set packages: %v", err)
	}

	if _, err := resolveApplyPackages(cmd); err == nil {
		t.Fatal("expected error for unknown package")
	}
}

func TestResolveApplyPackages_ErrorsOnDuplicatePackage(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "v1-interactions,v1-interactions"); err != nil {
		t.Fatalf("set packages: %v", err)
	}

	if _, err := resolveApplyPackages(cmd); err == nil {
		t.Fatal("expected error for duplicate package")
	}
}
