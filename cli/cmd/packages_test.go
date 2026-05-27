package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunPackages_ListsEmbeddedPackages(t *testing.T) {
	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)

	if err := runPackages(cmd, nil); err != nil {
		t.Fatalf("runPackages() error = %v", err)
	}

	out := output.String()
	for _, want := range []string{
		"Available fragment packages",
		"use with --package or --packages",
		"name: v1-interactions",
		"name: v2-request-evals",
		"name: v2-response-evals",
		"version: 1.0.0",
		"description:",
		"inbound:",
		"outbound:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("runPackages() output missing %q:\n%s", want, out)
		}
	}
}
