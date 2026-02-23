package policy

import (
	"strings"
	"testing"
)

func loadV1Package(t *testing.T) *Package {
	t.Helper()
	pkg, err := LoadPackage("v1-interactions")
	if err != nil {
		t.Fatalf("failed to load v1-interactions package: %v", err)
	}
	return pkg
}

func getFragmentXML(t *testing.T, pkg *Package, id string) string {
	t.Helper()
	xml, ok := pkg.GetFragmentXML(id)
	if !ok {
		t.Fatalf("fragment %q not found in package", id)
	}
	return xml
}

func TestFragmentsContainV1InteractionsEndpoint(t *testing.T) {
	pkg := loadV1Package(t)

	for _, id := range []string{"hl-interactions-input", "hl-interactions-output"} {
		t.Run(id, func(t *testing.T) {
			xml := getFragmentXML(t, pkg, id)
			if !strings.Contains(xml, "/detection/v1/interactions") {
				t.Errorf("fragment %q does not contain expected endpoint /detection/v1/interactions", id)
			}
		})
	}
}

func TestFragmentsInjectMetadataInBody(t *testing.T) {
	pkg := loadV1Package(t)

	metadataPatterns := []string{
		`["metadata"]`,
		`["model"]`,
		`["requester_id"]`,
	}

	for _, id := range []string{"hl-interactions-input", "hl-interactions-output"} {
		t.Run(id, func(t *testing.T) {
			xml := getFragmentXML(t, pkg, id)
			for _, pattern := range metadataPatterns {
				if !strings.Contains(xml, pattern) {
					t.Errorf("%s does not contain metadata pattern %q", id, pattern)
				}
			}
		})
	}
}

func TestFragmentCountV1(t *testing.T) {
	pkg := loadV1Package(t)
	expected := 3
	if len(pkg.Fragments) != expected {
		t.Errorf("expected %d fragments, got %d", expected, len(pkg.Fragments))
	}
}

func TestInputFragmentReadsMetadataHeaders(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	expectedHeaders := []string{"HL-Model", "HL-Provider", "HL-Requester-Id"}
	for _, header := range expectedHeaders {
		if !strings.Contains(input, header) {
			t.Errorf("InputFragment does not read metadata header %q", header)
		}
	}
}

func TestInputFragmentDeletesMetadataHeaders(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	deletePatterns := []string{
		`<set-header name="HL-Model" exists-action="delete"`,
		`<set-header name="HL-Provider" exists-action="delete"`,
		`<set-header name="HL-Requester-Id" exists-action="delete"`,
	}
	for _, pattern := range deletePatterns {
		if !strings.Contains(input, pattern) {
			t.Errorf("InputFragment does not delete header matching pattern %q", pattern)
		}
	}
}

func TestInputFragmentHandlesBlockAction(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	if !strings.Contains(input, `"Block"`) {
		t.Error("InputFragment does not handle Block action (v1 casing)")
	}
	if !strings.Contains(input, "<return-response>") {
		t.Error("InputFragment does not use return-response for Block action")
	}
	if !strings.Contains(input, "hl-blocked") {
		t.Error("InputFragment does not construct hl-blocked response")
	}
}

func TestOutputFragmentSurfacesRuntimeActionHeader(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(output, "HL-Runtime-Action") {
		t.Error("OutputFragment does not surface HL-Runtime-Action header")
	}
}

func TestFragmentsDoNotContainLegacyXHLHeaders(t *testing.T) {
	pkg := loadV1Package(t)
	for _, frag := range pkg.Fragments {
		if strings.Contains(frag.XML, "X-HL-") {
			t.Errorf("fragment %q still contains legacy X-HL-* header references", frag.ID)
		}
	}
}

func TestRuntimeActionContractUsesEvaluationAction(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(input, `["evaluation"]?["action"]`) {
		t.Error("InputFragment does not parse evaluation.action from response body")
	}
	if !strings.Contains(output, `["evaluation"]?["action"]`) {
		t.Error("OutputFragment does not parse evaluation.action from response body")
	}
}

func TestFragmentsExtractThreatLevel(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(input, `["evaluation"]?["threat_level"]`) {
		t.Error("InputFragment does not parse evaluation.threat_level from response body")
	}
	if !strings.Contains(output, `["evaluation"]?["threat_level"]`) {
		t.Error("OutputFragment does not parse evaluation.threat_level from response body")
	}
}

func TestFragmentsSurfaceInteractionHeaders(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(output, "HL-Interaction-Action") {
		t.Error("OutputFragment does not surface HL-Interaction-Action header")
	}
	if !strings.Contains(output, "HL-Interaction-Threat-Level") {
		t.Error("OutputFragment does not surface HL-Interaction-Threat-Level header")
	}
}

func TestInputFragmentBlockResponseIncludesInteractionHeaders(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	if !strings.Contains(input, "HL-Interaction-Action") {
		t.Error("InputFragment block return-response does not include HL-Interaction-Action header")
	}
	if !strings.Contains(input, "HL-Interaction-Threat-Level") {
		t.Error("InputFragment block return-response does not include HL-Interaction-Threat-Level header")
	}
}

func TestFragmentsInitializeThreatLevelVariable(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(input, `name="hl_threat_level"`) {
		t.Error("InputFragment does not initialize hl_threat_level variable")
	}
	if !strings.Contains(output, `name="hl_threat_level"`) {
		t.Error("OutputFragment does not initialize hl_threat_level variable")
	}
}

func TestFragmentsSkipModelDiscoveryEndpoint(t *testing.T) {
	pkg := loadV1Package(t)

	for _, id := range []string{"hl-oauth-token-management", "hl-interactions-input", "hl-interactions-output"} {
		t.Run(id, func(t *testing.T) {
			xml := getFragmentXML(t, pkg, id)
			if !strings.Contains(xml, `name="hl_skip_models"`) {
				t.Fatalf("%s does not include skip logic", id)
			}
			if !strings.Contains(xml, `/v1/models`) {
				t.Fatalf("%s skip logic does not reference /v1/models", id)
			}
		})
	}
}

func TestOutputFragmentSkipsStreamingResponses(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(output, `name="hl_is_streaming"`) {
		t.Fatal("OutputFragment does not define hl_is_streaming variable")
	}
	if !strings.Contains(output, `parsed["stream"]`) {
		t.Fatal("OutputFragment does not inspect request stream flag")
	}
	if !strings.Contains(output, `"skipped-stream"`) {
		t.Fatal("OutputFragment does not mark streaming bypass status")
	}
}

func TestInputFragmentConstructsV1Body(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	patterns := []string{`["input"]`, `["messages"]`, `new JObject`}
	for _, pattern := range patterns {
		if !strings.Contains(input, pattern) {
			t.Errorf("InputFragment does not contain v1 body pattern %q", pattern)
		}
	}
}

func TestOutputFragmentExtractsChoicesMessages(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	patterns := []string{`choices`, `["message"]`, `["output"]`}
	for _, pattern := range patterns {
		if !strings.Contains(output, pattern) {
			t.Errorf("OutputFragment does not contain pattern %q", pattern)
		}
	}
}

func TestInputFragmentHandlesRedactAction(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	if !strings.Contains(input, `"Redact"`) {
		t.Error("InputFragment does not handle Redact action")
	}
	if !strings.Contains(input, `modified_data`) {
		t.Error("InputFragment does not reference modified_data for redaction")
	}
	if !strings.Contains(input, `["input"]?["messages"]`) {
		t.Error("InputFragment does not reference modified_data.input.messages")
	}
}

func TestOutputFragmentHandlesRedactAction(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	if !strings.Contains(output, `"Redact"`) {
		t.Error("OutputFragment does not handle Redact action")
	}
	if !strings.Contains(output, `modified_data`) {
		t.Error("OutputFragment does not reference modified_data for redaction")
	}
	if !strings.Contains(output, `["output"]?["messages"]`) {
		t.Error("OutputFragment does not reference modified_data.output.messages")
	}
}

func TestInputFragmentStoresMetadataContextVars(t *testing.T) {
	pkg := loadV1Package(t)
	input := getFragmentXML(t, pkg, "hl-interactions-input")

	contextVars := []string{`"hl_model"`, `"hl_provider"`, `"hl_requester_id"`}
	for _, v := range contextVars {
		if !strings.Contains(input, v) {
			t.Errorf("InputFragment does not store context variable %s", v)
		}
	}
}

func TestOutputFragmentReusesMetadataContextVars(t *testing.T) {
	pkg := loadV1Package(t)
	output := getFragmentXML(t, pkg, "hl-interactions-output")

	contextVars := []string{`"hl_model"`, `"hl_provider"`, `"hl_requester_id"`}
	for _, v := range contextVars {
		if !strings.Contains(output, v) {
			t.Errorf("OutputFragment does not reuse context variable %s", v)
		}
	}
}
