package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

func TestRemoveFragmentIDsPreservesSharedFragmentsWhenOthersRemain(t *testing.T) {
	requestPkg, err := policy.LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-request-evals) error = %v", err)
	}
	responsePkg, err := policy.LoadPackage("v2-response-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-response-evals) error = %v", err)
	}
	existingPolicy, err := policy.InjectHiddenLayerFragments(policy.BasePolicy, requestPkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments(request) error = %v", err)
	}
	existingPolicy, err = policy.InjectHiddenLayerFragments(existingPolicy, responsePkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments(response) error = %v", err)
	}

	ids, err := removeFragmentIDs(existingPolicy, []*policy.Package{requestPkg}, false)
	if err != nil {
		t.Fatalf("removeFragmentIDs error = %v", err)
	}

	if containsString(ids, "hl-oauth-token-management") {
		t.Fatalf("ids = %v, shared oauth fragment should be preserved", ids)
	}
	for _, want := range []string{"hl-v2-request-evaluations", "hl-v2-surface-runtime-action"} {
		if !containsString(ids, want) {
			t.Fatalf("ids = %v, missing %s", ids, want)
		}
	}
}

func TestRemoveFragmentIDsAllRemovesEveryDetectedFragment(t *testing.T) {
	requestPkg, err := policy.LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-request-evals) error = %v", err)
	}
	existingPolicy, err := policy.InjectHiddenLayerFragments(policy.BasePolicy, requestPkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments error = %v", err)
	}

	ids, err := removeFragmentIDs(existingPolicy, nil, true)
	if err != nil {
		t.Fatalf("removeFragmentIDs error = %v", err)
	}

	for _, want := range requestPkg.AllFragmentIDs() {
		if !containsString(ids, want) {
			t.Fatalf("ids = %v, missing %s", ids, want)
		}
	}
}

func TestConfirmRemovePolicyRequiresYes(t *testing.T) {
	var output bytes.Buffer
	ids := []string{"hl-oauth-token-management", "hl-interactions-input"}

	if err := confirmRemovePolicy(strings.NewReader("no\n"), &output, "openai-proxy", ids, false); err == nil {
		t.Fatal("expected cancellation")
	}
	out := output.String()
	for _, want := range []string{
		`API "openai-proxy"`,
		"hl-oauth-token-management",
		"hl-interactions-input",
		"Type 'yes' to continue",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("confirm output missing %q:\n%s", want, out)
		}
	}

	output.Reset()
	if err := confirmRemovePolicy(strings.NewReader("yes\n"), &output, "openai-proxy", ids, true); err != nil {
		t.Fatalf("confirmRemovePolicy error = %v, want nil", err)
	}
	if !strings.Contains(output.String(), "all detected HiddenLayer fragments") {
		t.Fatalf("all confirmation output missing --all wording:\n%s", output.String())
	}
}

func TestPrintRemoveCompleteMessages(t *testing.T) {
	v1, err := policy.LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage(v1-interactions) error = %v", err)
	}
	v2Request, err := policy.LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-request-evals) error = %v", err)
	}

	output := captureStdout(t, func() {
		printRemoveComplete("openai-proxy", []*policy.Package{v1}, false)
	})
	if !strings.Contains(output, "Removed selected HiddenLayer package fragments") {
		t.Fatalf("package-specific output missing selected wording:\n%s", output)
	}
	if !strings.Contains(output, "--package v1-interactions") {
		t.Fatalf("package-specific output missing re-enable package hint:\n%s", output)
	}
	if strings.Contains(output, "no longer uses HiddenLayer scanning") {
		t.Fatalf("package-specific output should not claim all scanning removed:\n%s", output)
	}

	output = captureStdout(t, func() {
		printRemoveComplete("openai-proxy", []*policy.Package{v1, v2Request}, false)
	})
	if !strings.Contains(output, "--packages v1-interactions,v2-request-evals") {
		t.Fatalf("multi-package output missing re-enable packages hint:\n%s", output)
	}

	output = captureStdout(t, func() {
		printRemoveComplete("openai-proxy", nil, true)
	})
	if !strings.Contains(output, "no longer uses HiddenLayer scanning") {
		t.Fatalf("--all output missing full removal wording:\n%s", output)
	}
	if strings.Contains(output, "--package") {
		t.Fatalf("--all output should not include package-specific hint:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(data)
}
