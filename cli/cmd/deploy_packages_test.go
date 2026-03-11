package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveDeployPackages_LoadsMultiple(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "v1-interactions,v2-request-evals"); err != nil {
		t.Fatalf("set packages: %v", err)
	}

	pkgs, err := resolveDeployPackages(cmd)
	if err != nil {
		t.Fatalf("resolveDeployPackages error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Manifest.Name != "v1-interactions" {
		t.Fatalf("pkg[0] = %q, want %q", pkgs[0].Manifest.Name, "v1-interactions")
	}
	if pkgs[1].Manifest.Name != "v2-request-evals" {
		t.Fatalf("pkg[1] = %q, want %q", pkgs[1].Manifest.Name, "v2-request-evals")
	}
}

func TestResolveDeployPackages_ErrorsOnUnknownPackage(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "does-not-exist"); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if _, err := resolveDeployPackages(cmd); err == nil {
		t.Fatal("expected error for unknown package")
	}
}

func TestResolveDeployPackages_ErrorsOnDuplicatePackage(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("packages", nil, "")
	if err := cmd.Flags().Set("packages", "v1-interactions,v1-interactions"); err != nil {
		t.Fatalf("set packages: %v", err)
	}
	if _, err := resolveDeployPackages(cmd); err == nil {
		t.Fatal("expected error for duplicate package")
	}
}
