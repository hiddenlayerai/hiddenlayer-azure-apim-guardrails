package policy

type PolicyDefinitionIDRequirement struct {
	EnvVar     string
	NamedValue string
}

func PolicyDefinitionIDRequirementForPackage(packageName string) (PolicyDefinitionIDRequirement, bool) {
	switch packageName {
	case "v2-beta-request-evals":
		return PolicyDefinitionIDRequirement{
			EnvVar:     "HL_REQ_EVALS_POLICY_ID",
			NamedValue: "hl-req-evals-policy-id",
		}, true
	case "v2-beta-response-evals":
		return PolicyDefinitionIDRequirement{
			EnvVar:     "HL_RESP_EVALS_POLICY_ID",
			NamedValue: "hl-resp-evals-policy-id",
		}, true
	default:
		return PolicyDefinitionIDRequirement{}, false
	}
}
