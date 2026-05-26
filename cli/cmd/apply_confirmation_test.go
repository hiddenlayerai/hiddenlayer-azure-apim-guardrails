package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/policy"
)

func TestSummarizeApplyFragmentsDedupesAcrossPackages(t *testing.T) {
	v2Request, err := policy.LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-request-evals) error = %v", err)
	}
	v2Response, err := policy.LoadPackage("v2-response-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-response-evals) error = %v", err)
	}

	inbound, outbound := summarizeApplyFragments([]*policy.Package{v2Request, v2Response})

	if countString(inbound, "hl-oauth-token-management") != 1 {
		t.Fatalf("inbound = %v, want one shared oauth fragment", inbound)
	}
	for _, want := range []string{"hl-v2-request-evaluations", "hl-v2-response-evals-inbound"} {
		if !containsString(inbound, want) {
			t.Fatalf("inbound = %v, missing %s", inbound, want)
		}
	}
	for _, want := range []string{"hl-v2-surface-runtime-action", "hl-v2-response-evaluations"} {
		if !containsString(outbound, want) {
			t.Fatalf("outbound = %v, missing %s", outbound, want)
		}
	}
}

func TestAllApplyFragmentsPresent(t *testing.T) {
	v1, err := policy.LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage(v1-interactions) error = %v", err)
	}
	v2Request, err := policy.LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("LoadPackage(v2-request-evals) error = %v", err)
	}

	policyXML := `<policies>
	<inbound>
		<base />
		<include-fragment fragment-id="hl-oauth-token-management" />
		<include-fragment fragment-id="hl-v2-request-evaluations" />
		<include-fragment fragment-id="hl-interactions-input" />
	</inbound>
	<backend><base /></backend>
	<outbound>
		<base />
		<include-fragment fragment-id="hl-v2-surface-runtime-action" />
		<include-fragment fragment-id="hl-interactions-output" />
	</outbound>
	<on-error><base /></on-error>
</policies>`

	if !allApplyFragmentsPresent(policyXML, []*policy.Package{v1}) {
		t.Fatal("expected v1 package fragments to be present")
	}
	if !allApplyFragmentsPresent(policyXML, []*policy.Package{v2Request}) {
		t.Fatal("expected v2 request package fragments to be present")
	}
}

func TestAllApplyFragmentsPresentDetectsMissingPackageFragment(t *testing.T) {
	v1, err := policy.LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage(v1-interactions) error = %v", err)
	}

	policyXML := `<policies>
	<inbound>
		<base />
		<include-fragment fragment-id="hl-oauth-token-management" />
	</inbound>
	<backend><base /></backend>
	<outbound><base /></outbound>
	<on-error><base /></on-error>
</policies>`

	if allApplyFragmentsPresent(policyXML, []*policy.Package{v1}) {
		t.Fatal("expected v1 package to be missing fragments")
	}
}

func TestConfirmApplyPolicyRequiresYes(t *testing.T) {
	pkg, err := policy.LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("LoadPackage(v1-interactions) error = %v", err)
	}
	entries := []*policy.Package{pkg}

	var output bytes.Buffer
	if err := confirmApplyPolicy(strings.NewReader("no\n"), &output, "openai-proxy", entries); err == nil {
		t.Fatal("expected cancellation")
	}
	out := output.String()
	for _, want := range []string{
		`API "openai-proxy"`,
		"hl-oauth-token-management",
		"hl-interactions-input",
		"hl-interactions-output",
		"Type 'yes' to continue",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("confirm output missing %q:\n%s", want, out)
		}
	}

	output.Reset()
	if err := confirmApplyPolicy(strings.NewReader("yes\n"), &output, "openai-proxy", entries); err != nil {
		t.Fatalf("confirmApplyPolicy error = %v, want nil", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countString(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}
