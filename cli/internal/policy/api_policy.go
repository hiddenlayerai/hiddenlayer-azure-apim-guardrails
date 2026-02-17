package policy

import (
	"fmt"
	"strings"
)

// BuildHiddenLayerPolicy generates the default policy XML that includes all
// fragments from the given package.
func BuildHiddenLayerPolicy(pkg *Package) string {
	var b strings.Builder
	b.WriteString("<policies>\n")
	b.WriteString("    <inbound>\n")
	b.WriteString("        <base />\n")
	b.WriteString("        <!-- Generate correlation ID for request tracking -->\n")
	b.WriteString("        <set-variable name=\"correlationId\" value=\"@(context.RequestId.ToString())\" />\n")
	for _, id := range pkg.InboundIDs() {
		b.WriteString(fmt.Sprintf("\n        <!-- %s -->\n", getFragmentComment(id)))
		b.WriteString(fmt.Sprintf("        <include-fragment fragment-id=\"%s\" />\n", id))
	}
	b.WriteString("    </inbound>\n\n")
	b.WriteString("    <backend>\n")
	b.WriteString("        <base />\n")
	b.WriteString("    </backend>\n\n")
	b.WriteString("    <outbound>\n")
	b.WriteString("        <base />\n")
	for _, id := range pkg.OutboundIDs() {
		b.WriteString(fmt.Sprintf("\n        <!-- %s -->\n", getFragmentComment(id)))
		b.WriteString(fmt.Sprintf("        <include-fragment fragment-id=\"%s\" />\n", id))
	}
	b.WriteString("    </outbound>\n\n")
	b.WriteString("    <on-error>\n")
	b.WriteString("        <base />\n")
	b.WriteString("    </on-error>\n")
	b.WriteString("</policies>")
	return b.String()
}

// BasePolicy returns a minimal policy without HiddenLayer
const BasePolicy = `<policies>
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
