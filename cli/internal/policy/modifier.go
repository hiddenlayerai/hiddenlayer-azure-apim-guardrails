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

func InjectHiddenLayerFragments(existingPolicy string, pkg *Package) (string, error) {
	if existingPolicy == "" {
		return BuildHiddenLayerPolicy(pkg), nil
	}

	parsed, err := parsePolicy(existingPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to parse existing policy: %w", err)
	}

	newInbound := injectIntoSection(parsed.Inbound, pkg.InboundIDs(), true)
	newInbound = normalizeOAuthTokenManagement(newInbound)
	newInbound = normalizeCorrelationID(newInbound)
	newOutbound := injectIntoSection(parsed.Outbound, pkg.OutboundIDs(), false)

	result := reconstructPolicy(newInbound, parsed.Backend, newOutbound, parsed.OnError)
	return result, nil
}

func RemoveHiddenLayerFragments(existingPolicy string, pkg *Package) (string, error) {
	return RemoveHiddenLayerFragmentsByIDs(existingPolicy, pkg.AllFragmentIDs())
}

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

func RemoveHiddenLayerFragmentsByIDs(existingPolicy string, fragmentIDs []string) (string, error) {
	if existingPolicy == "" {
		return BasePolicy, nil
	}

	result := existingPolicy

	for _, fragID := range fragmentIDs {
		pattern := fmt.Sprintf(`\s*<!--[^>]*-->\s*\n?\s*<include-fragment\s+fragment-id="%s"\s*/>\s*`, regexp.QuoteMeta(fragID))
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "\n")

		pattern2 := fmt.Sprintf(`\s*<include-fragment\s+fragment-id="%s"\s*/>\s*`, regexp.QuoteMeta(fragID))
		re2 := regexp.MustCompile(pattern2)
		result = re2.ReplaceAllString(result, "\n")
	}

	// Only remove correlationId and global HiddenLayer comments when no HL fragments remain.
	// This enables "safe partial removal" when multiple packages are applied to the same API policy.
	if len(DetectHiddenLayerFragmentIDs(result)) == 0 {
		correlationPattern := `\s*<!--[^>]*correlation[^>]*-->\s*\n?\s*<set-variable\s+name="correlationId"[^/]*/>\s*`
		reCorr := regexp.MustCompile("(?i)" + correlationPattern)
		result = reCorr.ReplaceAllString(result, "\n")

		correlationPattern2 := `\s*<set-variable\s+name="correlationId"\s+value="@\(context\.RequestId\.ToString\(\)\)"\s*/>\s*`
		reCorr2 := regexp.MustCompile(correlationPattern2)
		result = reCorr2.ReplaceAllString(result, "\n")

		hlCommentPattern := `\s*<!--[^>]*HiddenLayer[^>]*-->\s*`
		reHL := regexp.MustCompile("(?i)" + hlCommentPattern)
		result = reHL.ReplaceAllString(result, "\n")
	}

	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")
	result = cleanupWhitespace(result)

	return result, nil
}

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

func hasAllHLFragments(policyXML string, pkg *Package) bool {
	for _, fragID := range pkg.AllFragmentIDs() {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, policyXML); !matched {
			return false
		}
	}
	return true
}

func injectIntoSection(sectionContent string, fragmentIDs []string, isInbound bool) string {
	if sectionContent == "" {
		sectionContent = "\n        <base />\n    "
	}

	missing := make([]string, 0, len(fragmentIDs))
	for _, fragID := range fragmentIDs {
		pattern := fmt.Sprintf(`<include-fragment\s+fragment-id="%s"\s*/>`, regexp.QuoteMeta(fragID))
		if matched, _ := regexp.MatchString(pattern, sectionContent); matched {
			continue
		}
		missing = append(missing, fragID)
	}
	needCorrelation := false
	if isInbound && !strings.Contains(sectionContent, `name="correlationId"`) {
		needCorrelation = len(missing) > 0 || strings.Contains(sectionContent, `fragment-id="hl-`)
	}
	if len(missing) == 0 && !needCorrelation {
		return sectionContent
	}

	var correlation strings.Builder
	var fragments strings.Builder
	if needCorrelation {
		correlation.WriteString("        <!-- Generate correlation ID for request tracking -->\n")
		correlation.WriteString("        <set-variable name=\"correlationId\" value=\"@(context.RequestId.ToString())\" />\n")
	}

	for i, fragID := range missing {
		comment := getFragmentComment(fragID)
		fragments.WriteString(fmt.Sprintf("\n        <!-- %s -->\n", comment))
		fragments.WriteString(fmt.Sprintf("        <include-fragment fragment-id=\"%s\" />\n", fragID))

		if i < len(missing)-1 {
			fragments.WriteString("")
		}
	}

	// Ensure correlationId is injected before any HiddenLayer fragment includes (if needed).
	if correlation.Len() > 0 {
		corrBlock := correlation.String()
		corrAfterBase := "\n" + corrBlock

		hlIdx := strings.Index(sectionContent, `fragment-id="hl-`)
		if hlIdx != -1 {
			lineStart := strings.LastIndex(sectionContent[:hlIdx], "\n")
			if lineStart == -1 {
				lineStart = 0
			} else {
				lineStart++
			}
			insertAt := lineStart

			prevLineEnd := lineStart - 1
			if prevLineEnd >= 0 {
				prevLineStart := strings.LastIndex(sectionContent[:prevLineEnd], "\n")
				if prevLineStart == -1 {
					prevLineStart = 0
				} else {
					prevLineStart++
				}
				prevLine := strings.TrimSpace(sectionContent[prevLineStart:prevLineEnd])
				if strings.HasPrefix(prevLine, "<!--") && strings.HasSuffix(prevLine, "-->") {
					insertAt = prevLineStart
				}
			}

			sectionContent = sectionContent[:insertAt] + corrBlock + sectionContent[insertAt:]
		} else {
			basePattern := regexp.MustCompile(`(<base\s*/>|<base>\s*</base>)`)
			if basePattern.MatchString(sectionContent) {
				sectionContent = basePattern.ReplaceAllString(sectionContent, "$1"+corrAfterBase)
			} else {
				sectionContent = corrAfterBase + sectionContent
			}
		}
	}

	// When layering packages, make sure new inbound fragments don't end up before the
	// existing oauth fragment (which defines access_token).
	if isInbound && strings.Contains(sectionContent, `fragment-id="hl-oauth-token-management"`) {
		oauthPattern := regexp.MustCompile(`(?m)^[ \t]*<include-fragment\s+fragment-id="hl-oauth-token-management"\s*/>\s*$`)
		if oauthPattern.MatchString(sectionContent) {
			return oauthPattern.ReplaceAllString(sectionContent, "$0"+fragments.String())
		}
	}

	basePattern := regexp.MustCompile(`(<base\s*/>|<base>\s*</base>)`)
	if basePattern.MatchString(sectionContent) {
		return basePattern.ReplaceAllString(sectionContent, "$1"+fragments.String())
	}

	return fragments.String() + sectionContent
}

func getFragmentComment(fragID string) string {
	name := strings.TrimPrefix(fragID, "hl-")
	name = strings.ReplaceAll(name, "-", " ")
	return "HiddenLayer: " + name
}

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

func normalizeOAuthTokenManagement(inbound string) string {
	const oauthID = "hl-oauth-token-management"
	includeRe := regexp.MustCompile(`<include-fragment\s+fragment-id="(hl-[^"]+)"\s*/>`)
	all := includeRe.FindAllStringSubmatchIndex(inbound, -1)
	if len(all) == 0 {
		return inbound
	}

	oauthIdx := -1
	firstOtherIdx := -1
	for _, m := range all {
		idStart, idEnd := m[2], m[3]
		id := inbound[idStart:idEnd]
		if id == oauthID {
			oauthIdx = m[0]
			continue
		}
		if firstOtherIdx == -1 || m[0] < firstOtherIdx {
			firstOtherIdx = m[0]
		}
	}
	if oauthIdx == -1 || firstOtherIdx == -1 || oauthIdx < firstOtherIdx {
		return inbound
	}

	lineStart := strings.LastIndex(inbound[:oauthIdx], "\n")
	if lineStart == -1 {
		lineStart = 0
	} else {
		lineStart++
	}
	blockStart := lineStart

	prevLineEnd := lineStart - 1
	if prevLineEnd >= 0 {
		prevLineStart := strings.LastIndex(inbound[:prevLineEnd], "\n")
		if prevLineStart == -1 {
			prevLineStart = 0
		} else {
			prevLineStart++
		}
		prevLine := strings.TrimSpace(inbound[prevLineStart:prevLineEnd])
		if strings.HasPrefix(prevLine, "<!--") && strings.HasSuffix(prevLine, "-->") {
			blockStart = prevLineStart
		}
	}

	blockEnd := strings.Index(inbound[oauthIdx:], "\n")
	if blockEnd != -1 {
		blockEnd = oauthIdx + blockEnd + 1
	} else {
		blockEnd = len(inbound)
	}

	block := inbound[blockStart:blockEnd]
	without := inbound[:blockStart] + inbound[blockEnd:]

	// Insert oauth block right before the first other HiddenLayer fragment include (including its comment line if present).
	insertAt := firstOtherIdx
	otherLineStart := strings.LastIndex(inbound[:firstOtherIdx], "\n")
	if otherLineStart == -1 {
		otherLineStart = 0
	} else {
		otherLineStart++
	}
	otherBlockStart := otherLineStart
	otherPrevLineEnd := otherLineStart - 1
	if otherPrevLineEnd >= 0 {
		otherPrevLineStart := strings.LastIndex(inbound[:otherPrevLineEnd], "\n")
		if otherPrevLineStart == -1 {
			otherPrevLineStart = 0
		} else {
			otherPrevLineStart++
		}
		otherPrevLine := strings.TrimSpace(inbound[otherPrevLineStart:otherPrevLineEnd])
		if strings.HasPrefix(otherPrevLine, "<!--") && strings.HasSuffix(otherPrevLine, "-->") {
			otherBlockStart = otherPrevLineStart
		}
	}
	insertAt = otherBlockStart

	if insertAt > blockStart {
		insertAt -= (blockEnd - blockStart)
	}

	if insertAt < 0 || insertAt > len(without) {
		return inbound
	}

	return without[:insertAt] + block + without[insertAt:]
}

func normalizeCorrelationID(inbound string) string {
	// Ensure the correlationId variable is defined before any HiddenLayer fragments
	// that reference it (e.g., when setting X-Correlation-Id).
	hlIdx := strings.Index(inbound, `fragment-id="hl-`)
	if hlIdx == -1 {
		return inbound
	}

	setIdx := strings.Index(inbound, `name="correlationId"`)
	if setIdx == -1 {
		return inbound
	}

	corrLineStart := strings.LastIndex(inbound[:setIdx], "\n")
	if corrLineStart == -1 {
		corrLineStart = 0
	} else {
		corrLineStart++
	}
	blockStart := corrLineStart

	prevLineEnd := corrLineStart - 1
	if prevLineEnd >= 0 {
		prevLineStart := strings.LastIndex(inbound[:prevLineEnd], "\n")
		if prevLineStart == -1 {
			prevLineStart = 0
		} else {
			prevLineStart++
		}
		prevLine := strings.TrimSpace(inbound[prevLineStart:prevLineEnd])
		if strings.HasPrefix(prevLine, "<!--") && strings.HasSuffix(prevLine, "-->") {
			blockStart = prevLineStart
		}
	}

	corrLineEnd := strings.Index(inbound[corrLineStart:], "\n")
	if corrLineEnd != -1 {
		corrLineEnd = corrLineStart + corrLineEnd + 1
	} else {
		corrLineEnd = len(inbound)
	}
	blockEnd := corrLineEnd

	// If correlationId is already before the first HL fragment include, keep it.
	if blockStart < hlIdx {
		return inbound
	}

	block := inbound[blockStart:blockEnd]
	without := inbound[:blockStart] + inbound[blockEnd:]

	hlIdx2 := strings.Index(without, `fragment-id="hl-`)
	if hlIdx2 == -1 {
		return inbound
	}

	hlLineStart := strings.LastIndex(without[:hlIdx2], "\n")
	if hlLineStart == -1 {
		hlLineStart = 0
	} else {
		hlLineStart++
	}
	insertAt := hlLineStart

	hlPrevLineEnd := hlLineStart - 1
	if hlPrevLineEnd >= 0 {
		hlPrevLineStart := strings.LastIndex(without[:hlPrevLineEnd], "\n")
		if hlPrevLineStart == -1 {
			hlPrevLineStart = 0
		} else {
			hlPrevLineStart++
		}
		hlPrevLine := strings.TrimSpace(without[hlPrevLineStart:hlPrevLineEnd])
		if strings.HasPrefix(hlPrevLine, "<!--") && strings.HasSuffix(hlPrevLine, "-->") {
			insertAt = hlPrevLineStart
		}
	}

	if insertAt < 0 || insertAt > len(without) {
		return inbound
	}

	return without[:insertAt] + block + without[insertAt:]
}

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
