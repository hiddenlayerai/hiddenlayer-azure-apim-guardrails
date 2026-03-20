package policy

import (
	"strings"
	"testing"
)

// Test policies for testing
const emptyPolicy = ``

const basePolicyOnly = `<policies>
    <inbound>
        <base />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

const policyWithCustomElements = `<policies>
    <inbound>
        <base />
        <rate-limit calls="100" renewal-period="60" />
        <set-header name="X-Custom-Header" exists-action="override">
            <value>custom-value</value>
        </set-header>
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
        <set-header name="X-Response-Header" exists-action="override">
            <value>response-value</value>
        </set-header>
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

const policyWithHLFragments = `<policies>
    <inbound>
        <base />
        <!-- Generate correlation ID for request tracking -->
        <set-variable name="correlationId" value="@(context.RequestId.ToString())" />

        <!-- HiddenLayer: oauth token management -->
        <include-fragment fragment-id="hl-oauth-token-management" />

        <!-- HiddenLayer: interactions input -->
        <include-fragment fragment-id="hl-interactions-input" />
    </inbound>

    <backend>
        <base />
    </backend>

    <outbound>
        <base />

        <!-- HiddenLayer: interactions output -->
        <include-fragment fragment-id="hl-interactions-output" />

    </outbound>

    <on-error>
        <base />
    </on-error>
</policies>`

const policyWithHLAndCustom = `<policies>
    <inbound>
        <base />
        <rate-limit calls="100" renewal-period="60" />
        <!-- Generate correlation ID for request tracking -->
        <set-variable name="correlationId" value="@(context.RequestId.ToString())" />

        <!-- HiddenLayer: oauth token management -->
        <include-fragment fragment-id="hl-oauth-token-management" />

        <!-- HiddenLayer: interactions input -->
        <include-fragment fragment-id="hl-interactions-input" />
        <set-header name="X-Custom-Header" exists-action="override">
            <value>custom-value</value>
        </set-header>
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
        <set-header name="X-Response-Header" exists-action="override">
            <value>response-value</value>
        </set-header>
        <!-- HiddenLayer: interactions output -->
        <include-fragment fragment-id="hl-interactions-output" />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

func mustLoadV1(t *testing.T) *Package {
	t.Helper()
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("failed to load v1-interactions: %v", err)
	}
	return pkg
}

func mustLoadV2RequestEvals(t *testing.T) *Package {
	t.Helper()
	pkg, err := LoadPackage("v2-request-evals")
	if err != nil {
		t.Fatalf("failed to load v2-request-evals: %v", err)
	}
	return pkg
}

func mustLoadV2ResponseEvals(t *testing.T) *Package {
	t.Helper()
	pkg, err := LoadPackage("v2-response-evals")
	if err != nil {
		t.Fatalf("failed to load v2-response-evals: %v", err)
	}
	return pkg
}

func TestHasHiddenLayerFragments(t *testing.T) {
	pkg := mustLoadV1(t)

	tests := []struct {
		name     string
		policy   string
		expected bool
	}{
		{"empty policy", emptyPolicy, false},
		{"base policy only", basePolicyOnly, false},
		{"custom elements only", policyWithCustomElements, false},
		{"has HL fragments", policyWithHLFragments, true},
		{"has HL and custom", policyWithHLAndCustom, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasHiddenLayerFragments(tt.policy, pkg)
			if result != tt.expected {
				t.Errorf("HasHiddenLayerFragments() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasHiddenLayerFragments_RequiresFullPackage(t *testing.T) {
	pkg := mustLoadV2RequestEvals(t)

	policyWithSharedOnly := `<policies>
    <inbound>
        <base />
        <include-fragment fragment-id="hl-oauth-token-management" />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	if HasHiddenLayerFragments(policyWithSharedOnly, pkg) {
		t.Fatal("expected shared oauth fragment alone to be insufficient for package detection")
	}
}

func TestGetPresentFragments(t *testing.T) {
	pkg := mustLoadV1(t)

	tests := []struct {
		name     string
		policy   string
		expected []string
	}{
		{"empty policy", emptyPolicy, nil},
		{"base policy only", basePolicyOnly, nil},
		{"has HL fragments", policyWithHLFragments, []string{
			"hl-oauth-token-management",
			"hl-interactions-input",
			"hl-interactions-output",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPresentFragments(tt.policy, pkg)
			if len(result) != len(tt.expected) {
				t.Errorf("GetPresentFragments() returned %d fragments, want %d", len(result), len(tt.expected))
				return
			}
			for i, frag := range result {
				if frag != tt.expected[i] {
					t.Errorf("GetPresentFragments()[%d] = %s, want %s", i, frag, tt.expected[i])
				}
			}
		})
	}
}

func TestInjectHiddenLayerFragments_EmptyPolicy(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := InjectHiddenLayerFragments("", pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	// Should return the dynamically built policy
	expected := BuildHiddenLayerPolicy(pkg)
	if result != expected {
		t.Errorf("InjectHiddenLayerFragments() should return BuildHiddenLayerPolicy for empty input")
	}
}

func TestInjectHiddenLayerFragments_BaseOnly(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := InjectHiddenLayerFragments(basePolicyOnly, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	// Should contain all HL fragments
	for _, fragID := range pkg.AllFragmentIDs() {
		if !strings.Contains(result, fragID) {
			t.Errorf("InjectHiddenLayerFragments() result missing fragment %s", fragID)
		}
	}

	// Should contain correlationId variable
	if !strings.Contains(result, "correlationId") {
		t.Errorf("InjectHiddenLayerFragments() result missing correlationId variable")
	}
}

func TestInjectHiddenLayerFragments_PreservesCustomElements(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := InjectHiddenLayerFragments(policyWithCustomElements, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	// Should contain all HL fragments
	for _, fragID := range pkg.AllFragmentIDs() {
		if !strings.Contains(result, fragID) {
			t.Errorf("Result missing fragment %s", fragID)
		}
	}

	// Should preserve custom elements
	customElements := []string{
		"rate-limit",
		"X-Custom-Header",
		"X-Response-Header",
		"custom-value",
		"response-value",
	}
	for _, elem := range customElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Result missing custom element %s", elem)
		}
	}
}

func TestInjectHiddenLayerFragments_Idempotent(t *testing.T) {
	pkg := mustLoadV1(t)

	// First injection
	result1, err := InjectHiddenLayerFragments(basePolicyOnly, pkg)
	if err != nil {
		t.Fatalf("First injection error = %v", err)
	}

	// Second injection should be idempotent
	result2, err := InjectHiddenLayerFragments(result1, pkg)
	if err != nil {
		t.Fatalf("Second injection error = %v", err)
	}

	// Count fragment occurrences - should only appear once each
	for _, fragID := range pkg.AllFragmentIDs() {
		count := strings.Count(result2, fragID)
		if count != 1 {
			t.Errorf("Fragment %s appears %d times, want 1 (idempotency violated)", fragID, count)
		}
	}
}

func TestInjectHiddenLayerFragments_MultiplePackages_NoDuplicateCorrelationId(t *testing.T) {
	pkgA := mustLoadV2ResponseEvals(t)
	pkgB := mustLoadV2RequestEvals(t)

	withA, err := InjectHiddenLayerFragments(basePolicyOnly, pkgA)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments(pkgA) error = %v", err)
	}
	withAB, err := InjectHiddenLayerFragments(withA, pkgB)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments(pkgB) error = %v", err)
	}

	if c := strings.Count(withAB, `name="correlationId"`); c != 1 {
		t.Fatalf("expected correlationId to be injected once, got %d", c)
	}
	if c := strings.Count(withAB, `fragment-id="hl-oauth-token-management"`); c != 1 {
		t.Fatalf("expected hl-oauth-token-management include once, got %d", c)
	}

	oauthPos := strings.Index(withAB, `fragment-id="hl-oauth-token-management"`)
	reqPos := strings.Index(withAB, `fragment-id="hl-v2-request-evaluations"`)
	if oauthPos == -1 || reqPos == -1 {
		t.Fatalf("expected oauth and request eval fragments to be present (oauth=%d, request=%d)", oauthPos, reqPos)
	}
	if oauthPos > reqPos {
		t.Fatalf("expected oauth fragment to appear before request evals fragment (oauth=%d, request=%d)", oauthPos, reqPos)
	}
}

func TestRemoveHiddenLayerFragments_EmptyPolicy(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := RemoveHiddenLayerFragments("", pkg)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragments() error = %v", err)
	}

	// Should return BasePolicy for empty input
	if result != BasePolicy {
		t.Errorf("RemoveHiddenLayerFragments() should return BasePolicy for empty input")
	}
}

func TestRemoveHiddenLayerFragments_HLOnly(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := RemoveHiddenLayerFragments(policyWithHLFragments, pkg)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragments() error = %v", err)
	}

	// Should not contain any HL fragments
	for _, fragID := range pkg.AllFragmentIDs() {
		if strings.Contains(result, fragID) {
			t.Errorf("Result still contains fragment %s", fragID)
		}
	}

	// Should not contain HiddenLayer comments
	if strings.Contains(strings.ToLower(result), "hiddenlayer") {
		t.Errorf("Result still contains HiddenLayer comments")
	}

	// Should still be valid XML with required sections
	if err := ValidatePolicy(result); err != nil {
		t.Errorf("Result is not valid policy: %v", err)
	}
}

func TestRemoveHiddenLayerFragments_PreservesCustomElements(t *testing.T) {
	pkg := mustLoadV1(t)
	result, err := RemoveHiddenLayerFragments(policyWithHLAndCustom, pkg)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragments() error = %v", err)
	}

	// Should not contain any HL fragments
	for _, fragID := range pkg.AllFragmentIDs() {
		if strings.Contains(result, fragID) {
			t.Errorf("Result still contains fragment %s", fragID)
		}
	}

	// Should preserve custom elements
	customElements := []string{
		"rate-limit",
		"X-Custom-Header",
		"X-Response-Header",
		"custom-value",
		"response-value",
	}
	for _, elem := range customElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Result missing custom element %s", elem)
		}
	}

	// Should still be valid XML
	if err := ValidatePolicy(result); err != nil {
		t.Errorf("Result is not valid policy: %v", err)
	}
}

func TestRemoveHiddenLayerFragments_NoHLFragments(t *testing.T) {
	pkg := mustLoadV1(t)
	original := policyWithCustomElements
	result, err := RemoveHiddenLayerFragments(original, pkg)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragments() error = %v", err)
	}

	// Should preserve all custom elements
	customElements := []string{
		"rate-limit",
		"X-Custom-Header",
		"X-Response-Header",
	}
	for _, elem := range customElements {
		if !strings.Contains(result, elem) {
			t.Errorf("Result missing custom element %s", elem)
		}
	}
}

func TestDetectHiddenLayerFragmentIDs(t *testing.T) {
	ids := DetectHiddenLayerFragmentIDs(policyWithHLFragments)
	if len(ids) != 3 {
		t.Errorf("DetectHiddenLayerFragmentIDs() returned %d IDs, want 3: %v", len(ids), ids)
	}

	ids = DetectHiddenLayerFragmentIDs(basePolicyOnly)
	if len(ids) != 0 {
		t.Errorf("DetectHiddenLayerFragmentIDs() returned %d IDs for base policy, want 0", len(ids))
	}
}

func TestRemoveHiddenLayerFragmentsByIDs(t *testing.T) {
	ids := []string{"hl-oauth-token-management", "hl-interactions-input", "hl-interactions-output"}
	result, err := RemoveHiddenLayerFragmentsByIDs(policyWithHLFragments, ids)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragmentsByIDs() error = %v", err)
	}
	for _, id := range ids {
		if strings.Contains(result, id) {
			t.Errorf("Result still contains fragment %s", id)
		}
	}
	if err := ValidatePolicy(result); err != nil {
		t.Errorf("Result is not valid policy: %v", err)
	}
}

func TestRemoveHiddenLayerFragmentsByIDs_PartialRemovalPreservesCorrelationWhenHLRemains(t *testing.T) {
	const policyWithMultipleHL = `<policies>
    <inbound>
        <base />
        <!-- Generate correlation ID for request tracking -->
        <set-variable name="correlationId" value="@(context.RequestId.ToString())" />
        <include-fragment fragment-id="hl-oauth-token-management" />
        <include-fragment fragment-id="hl-v2-request-evaluations" />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
        <include-fragment fragment-id="hl-v2-response-evaluations" />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	result, err := RemoveHiddenLayerFragmentsByIDs(policyWithMultipleHL, []string{"hl-v2-request-evaluations"})
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragmentsByIDs() error = %v", err)
	}
	if !strings.Contains(result, `name="correlationId"`) {
		t.Fatal("expected correlationId to be preserved when other HL fragments remain")
	}
	if !strings.Contains(result, `fragment-id="hl-oauth-token-management"`) {
		t.Fatal("expected shared oauth fragment include to remain")
	}
	if !strings.Contains(result, `fragment-id="hl-v2-response-evaluations"`) {
		t.Fatal("expected other HL fragment include to remain")
	}
}

func TestInjectHiddenLayerFragments_ReordersOAuthBeforeOtherHLFragments(t *testing.T) {
	const misordered = `<policies>
    <inbound>
        <base />
        <include-fragment fragment-id="hl-v2-request-evaluations" />
        <include-fragment fragment-id="hl-oauth-token-management" />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	pkg := mustLoadV2RequestEvals(t)
	fixed, err := InjectHiddenLayerFragments(misordered, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	oauthPos := strings.Index(fixed, `fragment-id="hl-oauth-token-management"`)
	reqPos := strings.Index(fixed, `fragment-id="hl-v2-request-evaluations"`)
	if oauthPos == -1 || reqPos == -1 {
		t.Fatalf("expected oauth and request eval fragments to be present (oauth=%d, request=%d)", oauthPos, reqPos)
	}
	if oauthPos > reqPos {
		t.Fatalf("expected oauth fragment to appear before request evals fragment (oauth=%d, request=%d)", oauthPos, reqPos)
	}
}

func TestInjectHiddenLayerFragments_RepairsMissingCorrelationId(t *testing.T) {
	const missingCorrelation = `<policies>
    <inbound>
        <base />
        <include-fragment fragment-id="hl-oauth-token-management" />
        <include-fragment fragment-id="hl-v2-request-evaluations" />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	pkg := mustLoadV2RequestEvals(t)
	fixed, err := InjectHiddenLayerFragments(missingCorrelation, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}
	if !strings.Contains(fixed, `name="correlationId"`) {
		t.Fatal("expected InjectHiddenLayerFragments to add correlationId when HL fragments exist")
	}
}

func TestInjectHiddenLayerFragments_InsertsCorrelationBeforeHLFragments_WhenBaseIsAfterFragments(t *testing.T) {
	const baseAfterFragments = `<policies>
    <inbound>
        <include-fragment fragment-id="hl-oauth-token-management" />
        <include-fragment fragment-id="hl-v2-request-evaluations" />
        <base />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	pkg := mustLoadV2RequestEvals(t)
	fixed, err := InjectHiddenLayerFragments(baseAfterFragments, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	corrPos := strings.Index(fixed, `name="correlationId"`)
	oauthPos := strings.Index(fixed, `fragment-id="hl-oauth-token-management"`)
	if corrPos == -1 || oauthPos == -1 {
		t.Fatalf("expected correlationId and oauth fragment to be present (corr=%d, oauth=%d)", corrPos, oauthPos)
	}
	if corrPos > oauthPos {
		t.Fatalf("expected correlationId to appear before oauth fragment (corr=%d, oauth=%d)", corrPos, oauthPos)
	}
}

func TestInjectHiddenLayerFragments_ReordersCorrelationBeforeHLFragments(t *testing.T) {
	const misorderedCorrelation = `<policies>
    <inbound>
        <base />
        <include-fragment fragment-id="hl-oauth-token-management" />
        <include-fragment fragment-id="hl-v2-request-evaluations" />
        <!-- Generate correlation ID for request tracking -->
        <set-variable name="correlationId" value="@(context.RequestId.ToString())" />
    </inbound>
    <backend>
        <base />
    </backend>
    <outbound>
        <base />
    </outbound>
    <on-error>
        <base />
    </on-error>
</policies>`

	pkg := mustLoadV2RequestEvals(t)
	fixed, err := InjectHiddenLayerFragments(misorderedCorrelation, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	corrPos := strings.Index(fixed, `name="correlationId"`)
	oauthPos := strings.Index(fixed, `fragment-id="hl-oauth-token-management"`)
	if corrPos == -1 || oauthPos == -1 {
		t.Fatalf("expected correlationId and oauth fragment to be present (corr=%d, oauth=%d)", corrPos, oauthPos)
	}
	if corrPos > oauthPos {
		t.Fatalf("expected correlationId to appear before oauth fragment (corr=%d, oauth=%d)", corrPos, oauthPos)
	}
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name      string
		policy    string
		expectErr bool
	}{
		{"valid policy", basePolicyOnly, false},
		{"valid HL policy", policyWithHLFragments, false},
		{"empty policy", "", true},
		{"missing root", "<inbound><base/></inbound>", true},
		{"missing section", `<policies><inbound><base/></inbound><backend><base/></backend><outbound><base/></outbound></policies>`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.policy)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidatePolicy() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestInjectAndRemove_RoundTrip(t *testing.T) {
	pkg := mustLoadV1(t)

	original := policyWithCustomElements

	// Inject HL fragments
	withHL, err := InjectHiddenLayerFragments(original, pkg)
	if err != nil {
		t.Fatalf("InjectHiddenLayerFragments() error = %v", err)
	}

	// Verify HL fragments are present
	if !HasHiddenLayerFragments(withHL, pkg) {
		t.Error("After injection, policy should have HL fragments")
	}

	// Remove HL fragments
	cleaned, err := RemoveHiddenLayerFragments(withHL, pkg)
	if err != nil {
		t.Fatalf("RemoveHiddenLayerFragments() error = %v", err)
	}

	// Verify HL fragments are gone
	if HasHiddenLayerFragments(cleaned, pkg) {
		t.Error("After removal, policy should not have HL fragments")
	}

	// Verify custom elements are preserved
	customElements := []string{
		"rate-limit",
		"X-Custom-Header",
		"X-Response-Header",
	}
	for _, elem := range customElements {
		if !strings.Contains(cleaned, elem) {
			t.Errorf("After round-trip, missing custom element %s", elem)
		}
	}
}

func TestFormatPolicyForDisplay(t *testing.T) {
	longPolicy := strings.Repeat("line\n", 100)

	// Should truncate long policies
	result := FormatPolicyForDisplay(longPolicy, 10)
	if !strings.Contains(result, "lines omitted") {
		t.Error("Long policy should be truncated with omission message")
	}

	// Should not truncate short policies
	shortPolicy := "short\npolicy"
	result = FormatPolicyForDisplay(shortPolicy, 10)
	if result != shortPolicy {
		t.Error("Short policy should not be modified")
	}
}
