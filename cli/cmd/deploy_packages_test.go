package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/hiddenlayerai/hiddenlayer-azure-apim-guardrails/cli/internal/azure"
)

type fakeDeployClient struct {
	namedValues map[string]*azure.NamedValue
	fragments   map[string]string
	updatedNV   []string
	updatedFrag []string
}

func (f *fakeDeployClient) GetNamedValueInfo(name string) (*azure.NamedValue, error) {
	if f.namedValues == nil {
		return nil, nil
	}
	return f.namedValues[name], nil
}

func (f *fakeDeployClient) CreateOrUpdateNamedValue(name, displayName, value string, secret bool) error {
	if f.namedValues == nil {
		f.namedValues = map[string]*azure.NamedValue{}
	}
	f.namedValues[name] = &azure.NamedValue{Name: name, Value: value, Secret: secret}
	f.updatedNV = append(f.updatedNV, name)
	return nil
}

func (f *fakeDeployClient) GetPolicyFragmentContent(name string) (string, bool, error) {
	if f.fragments == nil {
		return "", false, nil
	}
	xml, ok := f.fragments[name]
	return xml, ok, nil
}

func (f *fakeDeployClient) CreateOrUpdatePolicyFragment(name, xmlContent, description string) error {
	if f.fragments == nil {
		f.fragments = map[string]string{}
	}
	f.fragments[name] = xmlContent
	f.updatedFrag = append(f.updatedFrag, name)
	return nil
}

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

func TestDeployNamedValueCreatesMissing(t *testing.T) {
	client := &fakeDeployClient{}
	nv := namedValueDef{name: "hl-project-id", displayName: "hl-project-id", value: "project", secret: false}

	action, err := deployNamedValue(client, nv, false)
	if err != nil {
		t.Fatalf("deployNamedValue error: %v", err)
	}
	if action != "created" {
		t.Fatalf("action = %q, want created", action)
	}
	if len(client.updatedNV) != 1 || client.updatedNV[0] != nv.name {
		t.Fatalf("updatedNV = %v, want [%s]", client.updatedNV, nv.name)
	}
}

func TestDeployNamedValueRefusesDifferentExistingValueWithoutOverwrite(t *testing.T) {
	client := &fakeDeployClient{
		namedValues: map[string]*azure.NamedValue{
			"hl-project-id": {Name: "hl-project-id", Value: "old", Secret: false},
		},
	}
	nv := namedValueDef{name: "hl-project-id", displayName: "hl-project-id", value: "new", secret: false}

	_, err := deployNamedValue(client, nv, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("error = %q, want --overwrite guidance", err)
	}
	if len(client.updatedNV) != 0 {
		t.Fatalf("updatedNV = %v, want no updates", client.updatedNV)
	}
}

func TestDeployNamedValueOverwritesDifferentExistingValueWithOverwrite(t *testing.T) {
	client := &fakeDeployClient{
		namedValues: map[string]*azure.NamedValue{
			"hl-project-id": {Name: "hl-project-id", Value: "old", Secret: false},
		},
	}
	nv := namedValueDef{name: "hl-project-id", displayName: "hl-project-id", value: "new", secret: false}

	action, err := deployNamedValue(client, nv, true)
	if err != nil {
		t.Fatalf("deployNamedValue error: %v", err)
	}
	if action != "overwritten" {
		t.Fatalf("action = %q, want overwritten", action)
	}
	if client.namedValues[nv.name].Value != "new" {
		t.Fatalf("value = %q, want new", client.namedValues[nv.name].Value)
	}
}

func TestDeployNamedValueSkipsExistingSecretWithoutOverwrite(t *testing.T) {
	client := &fakeDeployClient{
		namedValues: map[string]*azure.NamedValue{
			"hl-client-secret": {Name: "hl-client-secret", Value: "[secret]", Secret: true},
		},
	}
	nv := namedValueDef{name: "hl-client-secret", displayName: "hl-client-secret", value: "new-secret", secret: true}

	action, err := deployNamedValue(client, nv, false)
	if err != nil {
		t.Fatalf("deployNamedValue error: %v", err)
	}
	if action != "skipped" {
		t.Fatalf("action = %q, want skipped", action)
	}
	if len(client.updatedNV) != 0 {
		t.Fatalf("updatedNV = %v, want no updates", client.updatedNV)
	}
}

func TestDeployPolicyFragmentCreatesMissing(t *testing.T) {
	client := &fakeDeployClient{}
	def := fragDef{id: "hl-fragment", xml: "<fragment />", desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, false)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "created" {
		t.Fatalf("action = %q, want created", action)
	}
	if len(client.updatedFrag) != 1 || client.updatedFrag[0] != def.id {
		t.Fatalf("updatedFrag = %v, want [%s]", client.updatedFrag, def.id)
	}
}

func TestDeployPolicyFragmentRefusesDifferentXMLWithoutOverwrite(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{"hl-fragment": "<fragment>customer edit</fragment>"},
	}
	def := fragDef{id: "hl-fragment", xml: "<fragment />", desc: "test fragment"}

	_, err := deployPolicyFragment(client, def, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--overwrite") {
		t.Fatalf("error = %q, want --overwrite guidance", err)
	}
	if len(client.updatedFrag) != 0 {
		t.Fatalf("updatedFrag = %v, want no updates", client.updatedFrag)
	}
}

func TestDeployPolicyFragmentOverwritesDifferentXMLWithOverwrite(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{"hl-fragment": "<fragment>customer edit</fragment>"},
	}
	def := fragDef{id: "hl-fragment", xml: "<fragment />", desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, true)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "overwritten" {
		t.Fatalf("action = %q, want overwritten", action)
	}
	if client.fragments[def.id] != def.xml {
		t.Fatalf("fragment XML = %q, want %q", client.fragments[def.id], def.xml)
	}
}

func TestDeployPolicyFragmentTreatsNormalizedXMLAsUnchanged(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{
			"hl-fragment": "\r\n<fragment>\r\n    <set-variable name=\"test\" value=\"true\" />   \r\n</fragment>\r\n",
		},
	}
	def := fragDef{id: "hl-fragment", xml: "<fragment>\n    <set-variable name=\"test\" value=\"true\" />\n</fragment>", desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, false)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("action = %q, want unchanged", action)
	}
	if len(client.updatedFrag) != 0 {
		t.Fatalf("updatedFrag = %v, want no updates", client.updatedFrag)
	}
}

func TestDeployPolicyFragmentTreatsEscapedXMLAsUnchanged(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{
			"hl-fragment": "&lt;fragment&gt;\n    &lt;set-variable name=&#34;test&#34; value=&#34;true&#34; /&gt;\n&lt;/fragment&gt;",
		},
	}
	def := fragDef{id: "hl-fragment", xml: "<fragment>\n    <set-variable name=\"test\" value=\"true\" />\n</fragment>", desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, false)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("action = %q, want unchanged", action)
	}
	if len(client.updatedFrag) != 0 {
		t.Fatalf("updatedFrag = %v, want no updates", client.updatedFrag)
	}
}

func TestDeployPolicyFragmentTreatsPolicyEntityEscapingAsUnchanged(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{
			"hl-fragment": `<fragment>
    <when condition='@(a &amp;&amp; b)'>
        <return-response />
    </when>
</fragment>`,
		},
	}
	def := fragDef{id: "hl-fragment", xml: `<fragment>
    <when condition='@(a && b)'>
        <return-response />
    </when>
</fragment>`, desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, false)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("action = %q, want unchanged", action)
	}
	if len(client.updatedFrag) != 0 {
		t.Fatalf("updatedFrag = %v, want no updates", client.updatedFrag)
	}
}

func TestDeployPolicyFragmentTreatsAPIMRawXMLFormattingAsUnchanged(t *testing.T) {
	client := &fakeDeployClient{
		fragments: map[string]string{
			"hl-fragment": "<fragment>\n\t<!-- test -->\n\t<set-variable name=\"hl_skip_models\" value=\"@(context.Request.Method.Equals(\"GET\", StringComparison.OrdinalIgnoreCase) && context.Request.Url.Path != null)\" />\n\t<when condition=\"@((bool)context.Variables[\"hl_skip_models\"])\">\n\t\t<set-variable name=\"hl_runtime_action\" value=\"@(\"skipped\")\" />\n\t\t<cache-store-value key=\"aad-ccg-token\" value=\"@((string)context.Variables[\"access_token\"])\" duration=\"{{hl-oauth-cache-seconds}}\" />\n\t\t<set-variable name=\"hl_is_streaming\" value=\"@{ var originalBody = (string)context.Variables.GetValueOrDefault(\"hl_original_body\", \"\"); if (string.IsNullOrEmpty(originalBody)) { return false; } return true; }\" />\n\t</when>\n</fragment>",
		},
	}
	def := fragDef{id: "hl-fragment", xml: `<fragment>
    <!-- test -->
    <set-variable name="hl_skip_models" value='@(context.Request.Method.Equals("GET", StringComparison.OrdinalIgnoreCase) &amp;&amp; context.Request.Url.Path != null)' />
    <when condition='@((bool)context.Variables["hl_skip_models"])'>
        <set-variable name="hl_runtime_action" value='@("skipped")' />
        <cache-store-value key="aad-ccg-token" value='@((string)context.Variables["access_token"])' duration="{{hl-oauth-cache-seconds}}" />
        <set-variable name="hl_is_streaming" value='@{
            var originalBody = (string)context.Variables.GetValueOrDefault("hl_original_body", "");
            if (string.IsNullOrEmpty(originalBody)) {
                return false;
            }
            return true;
        }' />
    </when>
</fragment>`, desc: "test fragment"}

	action, err := deployPolicyFragment(client, def, false)
	if err != nil {
		t.Fatalf("deployPolicyFragment error: %v", err)
	}
	if action != "unchanged" {
		t.Fatalf("action = %q, want unchanged", action)
	}
	if len(client.updatedFrag) != 0 {
		t.Fatalf("updatedFrag = %v, want no updates", client.updatedFrag)
	}
}

func TestCollectOverwritePreviewListsChangedNamedValuesAndFragments(t *testing.T) {
	client := &fakeDeployClient{
		namedValues: map[string]*azure.NamedValue{
			"hl-client-id":     {Name: "hl-client-id", Value: "old-client", Secret: false},
			"hl-client-secret": {Name: "hl-client-secret", Value: "[secret]", Secret: true},
			"hl-host":          {Name: "hl-host", Value: "hiddenlayer.ai", Secret: false},
		},
		fragments: map[string]string{
			"hl-fragment": "<fragment>customer edit</fragment>",
			"same":        "<fragment />",
		},
	}

	preview, err := collectOverwritePreview(client,
		[]namedValueDef{
			{name: "hl-client-id", value: "new-client"},
			{name: "hl-client-secret", value: "new-secret", secret: true},
			{name: "hl-host", value: "hiddenlayer.ai"},
			{name: "missing", value: "created"},
		},
		[]fragDef{
			{id: "hl-fragment", xml: "<fragment />"},
			{id: "same", xml: "<fragment />"},
			{id: "missing-fragment", xml: "<fragment />"},
		},
	)
	if err != nil {
		t.Fatalf("collectOverwritePreview error: %v", err)
	}
	if len(preview.namedValues) != 2 {
		t.Fatalf("namedValues = %#v, want 2 entries", preview.namedValues)
	}
	if preview.namedValues[0].name != "hl-client-id" || preview.namedValues[0].existingValue != "old-client" || preview.namedValues[0].newValue != "new-client" {
		t.Fatalf("first named value preview = %#v", preview.namedValues[0])
	}
	if preview.namedValues[1].name != "hl-client-secret" || !preview.namedValues[1].secret {
		t.Fatalf("second named value preview = %#v, want secret name only", preview.namedValues[1])
	}
	if len(preview.fragments) != 1 || preview.fragments[0] != "hl-fragment" {
		t.Fatalf("fragments = %#v, want [hl-fragment]", preview.fragments)
	}
}

func TestConfirmOverwriteRequiresYesAndHidesSecretValues(t *testing.T) {
	preview := overwritePreview{
		namedValues: []namedValueOverwrite{
			{name: "hl-client-id", existingValue: "old-client", newValue: "new-client"},
			{name: "hl-client-secret", secret: true},
		},
		fragments: []string{"hl-fragment"},
	}
	var output bytes.Buffer

	if err := confirmOverwrite(strings.NewReader("no\n"), &output, preview); err == nil {
		t.Fatal("expected cancellation")
	}
	out := output.String()
	for _, want := range []string{
		"hl-client-id: \"old-client\" -> \"new-client\"",
		"hl-client-secret (secret)",
		"hl-fragment",
		"Type 'yes' to continue",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("confirm output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "new-secret") || strings.Contains(out, "[secret] ->") {
		t.Fatalf("confirm output leaked secret value:\n%s", out)
	}

	output.Reset()
	if err := confirmOverwrite(strings.NewReader("yes\n"), &output, preview); err != nil {
		t.Fatalf("confirmOverwrite error = %v, want nil", err)
	}
}
