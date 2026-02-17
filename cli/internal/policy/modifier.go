package policy

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// PolicySection represents a section of the APIM policy
type PolicySection struct {
	Name     string
	Content  string
	Elements []string
}

// ParsedPolicy represents a parsed APIM policy
type ParsedPolicy struct {
	Inbound  string
	Backend  string
	Outbound string
	OnError  string
	Raw      string
}

// HasHiddenLayerFragments checks if the policy contains any of the package's fragments.
func HasHiddenLayerFragments(policyXML string, pkg *Package) bool {
	for _, fragID := range pkg.AllFragmentIDs() {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, policyXML); matched {
			return true
		}
	}
	return false
}

// GetPresentFragments returns which of the package's fragments are present in the policy.
func GetPresentFragments(policyXML string, pkg *Package) []string {
	var present []string
	for _, fragID := range pkg.AllFragmentIDs() {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, policyXML); matched {
			present = append(present, fragID)
		}
	}
	return present
}

// InjectHiddenLayerFragments injects HiddenLayer fragment includes into an existing policy
// while preserving all other policy elements.
func InjectHiddenLayerFragments(existingPolicy string, pkg *Package) (string, error) {
	if existingPolicy == "" {
		return BuildHiddenLayerPolicy(pkg), nil
	}

	parsed, err := parsePolicy(existingPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to parse existing policy: %w", err)
	}

	if hasAllHLFragments(existingPolicy, pkg) {
		return existingPolicy, nil
	}

	newInbound := injectIntoSection(parsed.Inbound, pkg.InboundIDs(), true)
	newOutbound := injectIntoSection(parsed.Outbound, pkg.OutboundIDs(), false)

	result := reconstructPolicy(newInbound, parsed.Backend, newOutbound, parsed.OnError)
	return result, nil
}

// RemoveHiddenLayerFragments removes HiddenLayer fragment includes from a policy
// using the package's fragment IDs.
func RemoveHiddenLayerFragments(existingPolicy string, pkg *Package) (string, error) {
	return RemoveHiddenLayerFragmentsByIDs(existingPolicy, pkg.AllFragmentIDs())
}

// DetectHiddenLayerFragmentIDs scans a policy for any hl-* include-fragment references
// and returns the detected fragment IDs.
func DetectHiddenLayerFragmentIDs(policyXML string) []string {
	re := regexp.MustCompile(`<include-fragment\s+fragment-id="(hl-[^"]+)"\s*/>`)
	matches := re.FindAllStringSubmatch(policyXML, -1)
	var ids []string
	seen := make(map[string]bool)
	for _, m := range matches {
		id := m[1]
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}

// RemoveHiddenLayerFragmentsByIDs removes the specified fragment IDs from a policy
// while preserving all other policy elements.
func RemoveHiddenLayerFragmentsByIDs(existingPolicy string, fragmentIDs []string) (string, error) {
	if existingPolicy == "" {
		return BasePolicy, nil
	}

	result := existingPolicy

	for _, fragID := range fragmentIDs {
		// Match comment followed by include-fragment
		pattern := fmt.Sprintf(`\s*<!--[^>]*-->\s*\n?\s*<include-fragment\s+fragment-id="%s"\s*/>\s*`, regexp.QuoteMeta(fragID))
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "\n")

		// Also match standalone include-fragment without comment
		pattern2 := fmt.Sprintf(`\s*<include-fragment\s+fragment-id="%s"\s*/>\s*`, regexp.QuoteMeta(fragID))
		re2 := regexp.MustCompile(pattern2)
		result = re2.ReplaceAllString(result, "\n")
	}

	// Remove the correlationId variable set by HiddenLayer (with its comment)
	correlationPattern := `\s*<!--[^>]*correlation[^>]*-->\s*\n?\s*<set-variable\s+name="correlationId"[^/]*/>\s*`
	reCorr := regexp.MustCompile("(?i)" + correlationPattern)
	result = reCorr.ReplaceAllString(result, "\n")

	// Also remove standalone correlationId set-variable (without comment)
	correlationPattern2 := `\s*<set-variable\s+name="correlationId"\s+value="@\(context\.RequestId\.ToString\(\)\)"\s*/>\s*`
	reCorr2 := regexp.MustCompile(correlationPattern2)
	result = reCorr2.ReplaceAllString(result, "\n")

	// Remove any remaining HiddenLayer-related comments
	hlCommentPattern := `\s*<!--[^>]*HiddenLayer[^>]*-->\s*`
	reHL := regexp.MustCompile("(?i)" + hlCommentPattern)
	result = reHL.ReplaceAllString(result, "\n")

	// Clean up extra blank lines
	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")
	result = cleanupWhitespace(result)

	return result, nil
}

// parsePolicy extracts the content of each section from the policy XML
func parsePolicy(policyXML string) (*ParsedPolicy, error) {
	parsed := &ParsedPolicy{Raw: policyXML}

	inboundRe := regexp.MustCompile(`(?s)<inbound>(.*?)</inbound>`)
	if match := inboundRe.FindStringSubmatch(policyXML); len(match) > 1 {
		parsed.Inbound = match[1]
	}

	backendRe := regexp.MustCompile(`(?s)<backend>(.*?)</backend>`)
	if match := backendRe.FindStringSubmatch(policyXML); len(match) > 1 {
		parsed.Backend = match[1]
	}

	outboundRe := regexp.MustCompile(`(?s)<outbound>(.*?)</outbound>`)
	if match := outboundRe.FindStringSubmatch(policyXML); len(match) > 1 {
		parsed.Outbound = match[1]
	}

	onErrorRe := regexp.MustCompile(`(?s)<on-error>(.*?)</on-error>`)
	if match := onErrorRe.FindStringSubmatch(policyXML); len(match) > 1 {
		parsed.OnError = match[1]
	}

	return parsed, nil
}

// hasAllHLFragments checks if the policy already has all fragments from the package.
func hasAllHLFragments(policyXML string, pkg *Package) bool {
	for _, fragID := range pkg.AllFragmentIDs() {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, policyXML); !matched {
			return false
		}
	}
	return true
}

// injectIntoSection injects fragment includes into a policy section.
// It places them after <base /> and adds the correlationId variable if isInbound.
func injectIntoSection(sectionContent string, fragmentIDs []string, isInbound bool) string {
	if sectionContent == "" {
		sectionContent = "\n        <base />\n    "
	}

	var fragments strings.Builder
	if isInbound {
		fragments.WriteString("\n        <!-- Generate correlation ID for request tracking -->\n")
		fragments.WriteString("        <set-variable name=\"correlationId\" value=\"@(context.RequestId.ToString())\" />\n")
	}

	for i, fragID := range fragmentIDs {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, sectionContent); matched {
			continue
		}

		comment := getFragmentComment(fragID)
		fragments.WriteString(fmt.Sprintf("\n        <!-- %s -->\n", comment))
		fragments.WriteString(fmt.Sprintf("        <include-fragment fragment-id=\"%s\" />\n", fragID))

		if i < len(fragmentIDs)-1 {
			fragments.WriteString("")
		}
	}

	if fragments.Len() == 0 {
		return sectionContent
	}

	basePattern := regexp.MustCompile(`(<base\s*/>|<base>\s*</base>)`)
	if basePattern.MatchString(sectionContent) {
		result := basePattern.ReplaceAllString(sectionContent, "$1"+fragments.String())
		return result
	}

	return fragments.String() + sectionContent
}

// getFragmentComment returns a descriptive comment for a fragment.
// It derives the comment from the fragment ID.
func getFragmentComment(fragID string) string {
	// Strip "hl-" prefix and convert hyphens to spaces
	name := strings.TrimPrefix(fragID, "hl-")
	name = strings.ReplaceAll(name, "-", " ")
	return "HiddenLayer: " + name
}

// reconstructPolicy rebuilds the full policy XML from sections
func reconstructPolicy(inbound, backend, outbound, onError string) string {
	var b strings.Builder
	b.WriteString("<policies>\n")
	b.WriteString("    <inbound>")
	b.WriteString(inbound)
	b.WriteString("</inbound>\n\n")
	b.WriteString("    <backend>")
	b.WriteString(backend)
	b.WriteString("</backend>\n\n")
	b.WriteString("    <outbound>")
	b.WriteString(outbound)
	b.WriteString("</outbound>\n\n")
	b.WriteString("    <on-error>")
	b.WriteString(onError)
	b.WriteString("</on-error>\n")
	b.WriteString("</policies>")
	return b.String()
}

// cleanupWhitespace removes excessive whitespace while preserving structure
func cleanupWhitespace(policyXML string) string {
	lines := strings.Split(policyXML, "\n")
	var cleaned []string
	prevEmpty := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isEmpty := trimmed == ""

		if isEmpty && prevEmpty {
			continue
		}

		cleaned = append(cleaned, line)
		prevEmpty = isEmpty
	}

	return strings.Join(cleaned, "\n")
}

// ValidatePolicy performs basic validation on policy XML
func ValidatePolicy(policyXML string) error {
	if policyXML == "" {
		return fmt.Errorf("policy is empty")
	}

	if !strings.Contains(policyXML, "<policies>") || !strings.Contains(policyXML, "</policies>") {
		return fmt.Errorf("policy must have <policies> root element")
	}

	sections := []string{"inbound", "backend", "outbound", "on-error"}
	for _, section := range sections {
		openTag := fmt.Sprintf("<%s>", section)
		closeTag := fmt.Sprintf("</%s>", section)
		if !strings.Contains(policyXML, openTag) || !strings.Contains(policyXML, closeTag) {
			return fmt.Errorf("policy must have <%s> section", section)
		}
	}

	decoder := xml.NewDecoder(strings.NewReader(policyXML))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("invalid XML: %w", err)
		}
	}

	return nil
}

// FormatPolicyForDisplay formats a policy for terminal display
func FormatPolicyForDisplay(policyXML string, maxLines int) string {
	lines := strings.Split(policyXML, "\n")
	if len(lines) <= maxLines {
		return policyXML
	}

	half := maxLines / 2
	result := strings.Join(lines[:half], "\n")
	result += fmt.Sprintf("\n... (%d lines omitted) ...\n", len(lines)-maxLines)
	result += strings.Join(lines[len(lines)-half:], "\n")
	return result
}
