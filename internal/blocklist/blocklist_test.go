package blocklist_test

import (
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/blocklist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePlanValidator struct{ known []string }

func (f fakePlanValidator) IsPlanName(name string) bool {
	for _, k := range f.known {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}

var testPlans = fakePlanValidator{known: []string{"aws", "gcp", "azure", "trial", "free"}}

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "blocklist-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func parseInline(op string, rules ...string) (blocklist.OperationBlocklist, error) {
	yaml := op + ":\n"
	for _, r := range rules {
		yaml += "  - '" + r + "'\n"
	}
	f, err := os.CreateTemp("", "bl-*.yaml")
	if err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err = f.WriteString(yaml); err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	if err = f.Close(); err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	bl, err := blocklist.ReadFromFile(f.Name())
	if err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	return bl.WithPlanValidator(testPlans)
}

// ctx is a test helper that builds an OperationContext with only the plan set.
func ctx(plan string) blocklist.OperationContext {
	return blocklist.OperationContext{PlanName: plan}
}

// --- parser ---

func TestParseRule_MessageOnly(t *testing.T) {
	// A rule with no filters at all is a no-op.
	bl, err := parseInline("provision", `"always blocked"`)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(ctx("any")))
}

func TestParseRule_WithPlan(t *testing.T) {
	bl, err := parseInline("provision", `"blocked {plan}","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(ctx("aws")), "blocked aws")
	assert.NoError(t, bl.CheckProvision(ctx("gcp")))
}

func TestParseRule_GlobalAccountPlaceholder(t *testing.T) {
	bl, err := parseInline("provision", `"blocked {plan} for {globalAccount}","plan=trial","GA=ga-123"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "ga-123"}), "blocked trial for ga-123")
}

func TestParseRule_PlanList(t *testing.T) {
	bl, err := parseInline("provision", `"blocked {plan}","plan=aws,gcp"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(ctx("aws")), "blocked aws")
	assert.EqualError(t, bl.CheckProvision(ctx("gcp")), "blocked gcp")
	assert.NoError(t, bl.CheckProvision(ctx("azure")))
}

// --- operation-type checks ---

func TestCheckUpdate(t *testing.T) {
	bl, err := parseInline("update", `"update blocked for {plan}","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckUpdate(ctx("aws")), "update blocked for aws")
	assert.NoError(t, bl.CheckUpdate(ctx("gcp")))
}

func TestCheckPlanUpgrade(t *testing.T) {
	bl, err := parseInline("planUpgrade", `"plan upgrade blocked for {plan}","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckPlanUpgrade(ctx("aws")), "plan upgrade blocked for aws")
	assert.NoError(t, bl.CheckPlanUpgrade(ctx("gcp")))
}

func TestCheckDeprovision(t *testing.T) {
	bl, err := parseInline("deprovision", `"deprovision blocked for {plan}","plan=gcp"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckDeprovision(ctx("gcp")), "deprovision blocked for gcp")
	assert.NoError(t, bl.CheckDeprovision(ctx("aws")))
}

// --- multi-rule and empty ---

func TestCheckRules_MultipleRules_FirstMatchWins(t *testing.T) {
	bl, err := parseInline("provision", `"first","plan=aws"`, `"second","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(ctx("aws")), "first")
}

func TestCheckRules_EmptyBlocklist(t *testing.T) {
	var bl blocklist.OperationBlocklist
	assert.NoError(t, bl.CheckProvision(ctx("aws")))
	assert.NoError(t, bl.CheckUpdate(ctx("aws")))
	assert.NoError(t, bl.CheckPlanUpgrade(ctx("aws")))
	assert.NoError(t, bl.CheckDeprovision(ctx("aws")))
}

// --- PlanValidator: unknown plan names are rejected at validation time ---

func TestMatchesPlan_UnknownPlanInRuleIsError(t *testing.T) {
	path := writeYAML(t, "provision:\n  - '\"blocked\",\"plan=notaplan\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	_, err = bl.WithPlanValidator(testPlans)
	assert.ErrorContains(t, err, "notaplan")
}

func TestMatchesPlan_UnknownPlanInListIsError(t *testing.T) {
	path := writeYAML(t, "provision:\n  - '\"blocked\",\"plan=aws,notaplan\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	_, err = bl.WithPlanValidator(testPlans)
	assert.ErrorContains(t, err, "notaplan")
}

// --- YAML: single string vs list ---

func TestReadFromFile_FullExample(t *testing.T) {
	yaml := `
provision:
  - '"provisioning is blocked for {plan} plan","plan=aws"'
  - '"provisioning is blocked for {plan} plan","plan=gcp"'
update: '"update is blocked for {plan}","plan=trial"'
planUpgrade: '"plan upgrade is blocked for {plan}","plan=aws"'
deprovision: '"deprovisioning is blocked for {plan}","plan=gcp"'
`
	path := writeYAML(t, yaml)
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	bl, err = bl.WithPlanValidator(testPlans)
	require.NoError(t, err)

	assert.EqualError(t, bl.CheckProvision(ctx("aws")), "provisioning is blocked for aws plan")
	assert.EqualError(t, bl.CheckProvision(ctx("gcp")), "provisioning is blocked for gcp plan")
	assert.NoError(t, bl.CheckProvision(ctx("azure")))

	assert.EqualError(t, bl.CheckUpdate(ctx("trial")), "update is blocked for trial")
	assert.NoError(t, bl.CheckUpdate(ctx("aws")))

	assert.EqualError(t, bl.CheckPlanUpgrade(ctx("aws")), "plan upgrade is blocked for aws")
	assert.NoError(t, bl.CheckPlanUpgrade(ctx("gcp")))

	assert.EqualError(t, bl.CheckDeprovision(ctx("gcp")), "deprovisioning is blocked for gcp")
	assert.NoError(t, bl.CheckDeprovision(ctx("aws")))
}

// --- hardening: empty/blank string rules are no-ops ---

func TestReadFromFile_EmptyFile(t *testing.T) {
	// Empty file → no-op blocklist, no error.
	path := writeYAML(t, "")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(ctx("trial")))
	assert.NoError(t, bl.CheckUpdate(ctx("trial")))
	assert.NoError(t, bl.CheckPlanUpgrade(ctx("trial")))
	assert.NoError(t, bl.CheckDeprovision(ctx("trial")))
}

func TestReadFromFile_EmptyKeysAreNoOp(t *testing.T) {
	// Keys present but with no rules → no-op.
	path := writeYAML(t, "provision:\ndeprovision:\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(ctx("trial")))
	assert.NoError(t, bl.CheckDeprovision(ctx("trial")))
}

func TestParseRule_EmptyStringSingleIsNoOp(t *testing.T) {
	// provision: '' → no rules loaded, no error.
	path := writeYAML(t, "provision: ''\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(ctx("trial")))
}

func TestParseRule_EmptyStringInListIsNoOp(t *testing.T) {
	// Empty string entry in a list is skipped; valid rule still works.
	path := writeYAML(t, "provision:\n  - ''\n  - '\"blocked\",\"plan=trial\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	bl, err = bl.WithPlanValidator(testPlans)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(ctx("trial")), "blocked")
	assert.NoError(t, bl.CheckProvision(ctx("aws")))
}

// --- hardening: empty message is a parse error ---

func TestParseRule_EmptyMessageSingleIsError(t *testing.T) {
	// provision: '""' → empty message, must fail loading.
	path := writeYAML(t, "provision: '\"\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_EmptyMessageWithPlanIsError(t *testing.T) {
	// provision: '"","plan=trial"' → empty message, must fail loading.
	path := writeYAML(t, "provision: '\"\",\"plan=trial\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

// --- hardening: message only (no filters) is a no-op ---

func TestParseRule_MessageOnlyIsNoOp(t *testing.T) {
	// No filters at all → no-op, does not block any plan.
	path := writeYAML(t, "provision: '\"blocked\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	bl, err = bl.WithPlanValidator(testPlans)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(ctx("trial")))
	assert.NoError(t, bl.CheckProvision(ctx("aws")))
	assert.NoError(t, bl.CheckProvision(ctx("gcp")))
}

// --- hardening: WithPlanValidator rejects unknown plan names ---

func TestWithPlanValidator_UnknownPlanNameIsError(t *testing.T) {
	// "trail" is a typo for "trial" — must be rejected at validation time.
	path := writeYAML(t, "provision: '\"blocked\",\"plan=trail\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	_, err = bl.WithPlanValidator(testPlans)
	assert.ErrorContains(t, err, "trail")
}

func TestWithPlanValidator_KnownPlanNameIsAccepted(t *testing.T) {
	path := writeYAML(t, "provision: '\"blocked\",\"plan=trial\"'\n")
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	_, err = bl.WithPlanValidator(testPlans)
	assert.NoError(t, err)
}

// --- GA!= exclusion ---

func TestGAExclusion_WithPlan_AllCombinations(t *testing.T) {
	// plan=trial + GA!=X: only trial plan for non-X GAs is blocked.
	bl, err := parseInline("provision", `"blocked","plan=trial","GA!=exempted-ga"`)
	require.NoError(t, err)

	// trial + other GA → blocked
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "other-ga"}), "blocked")
	// trial + exempted GA → not blocked
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "exempted-ga"}))
	// aws + other GA → not blocked (plan doesn't match)
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "aws", GlobalAccountID: "other-ga"}))
}

func TestGAExclusion_OnlyGANoPlan_IsError(t *testing.T) {
	// GA!= without plan= must be rejected — plan filter is required.
	path := writeYAML(t, "provision: '\"blocked\",\"GA!=exempted-ga\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestGAExclusion_CaseInsensitive(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=trial","GA!=ExemptedGA"`)
	require.NoError(t, err)

	// case-insensitive match — exempted regardless of casing
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "exemptedga"}))
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "EXEMPTEDGA"}))
}

func TestGAExclusion_EmptyValue_IsError(t *testing.T) {
	// GA!= with no value must be rejected at parse time.
	path := writeYAML(t, "provision: '\"blocked\",\"GA!=\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestGAExclusion_EmptyGAInContext_DoesNotMatchExclusion(t *testing.T) {
	// Defensive guard: broker validates globalaccount_id before reaching the blocklist,
	// but if an empty GA somehow arrives, it should not be treated as an exempt account.
	bl, err := parseInline("provision", `"blocked","plan=trial","GA!=exempted-ga"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: ""}), "blocked")
}

// --- GA= inclusion ---

func TestGAInclusion_WithPlan_AllCombinations(t *testing.T) {
	// plan=trial + GA=X: only trial plan for GA X is blocked.
	bl, err := parseInline("provision", `"blocked","plan=trial","GA=targeted-ga"`)
	require.NoError(t, err)

	// trial + targeted GA → blocked
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "targeted-ga"}), "blocked")
	// trial + other GA → not blocked
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "other-ga"}))
	// aws + targeted GA → not blocked (plan doesn't match)
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "aws", GlobalAccountID: "targeted-ga"}))
}

func TestGAInclusion_OnlyGANoPlan_IsError(t *testing.T) {
	// GA= without plan= must be rejected — plan filter is required.
	path := writeYAML(t, "provision: '\"blocked\",\"GA=targeted-ga\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestGAInclusion_CaseInsensitive(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=trial","GA=TargetedGA"`)
	require.NoError(t, err)

	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "targetedga"}), "blocked")
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "TARGETEDGA"}), "blocked")
}

func TestGAInclusion_EmptyValue_IsError(t *testing.T) {
	path := writeYAML(t, "provision: '\"blocked\",\"GA=\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestGAInclusion_EmptyGAInContext_NoMatch(t *testing.T) {
	// When the operation has no GA (empty string), GA= filter does not match.
	bl, err := parseInline("provision", `"blocked","plan=trial","GA=targeted-ga"`)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: ""}))
}

// --- GA!= list ---

func TestGAExclusion_List_ExemptsAll(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=trial","GA!=ga-1,ga-2"`)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "ga-1"}))
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "ga-2"}))
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "other-ga"}), "blocked")
}

func TestGAExclusion_List_EmptySegment_IsError(t *testing.T) {
	path := writeYAML(t, "provision: '\"blocked\",\"plan=trial\",\"GA!=ga-1,,ga-2\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

// --- GA= list ---

func TestGAInclusion_List_BlocksAll(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=trial","GA=ga-1,ga-2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "ga-1"}), "blocked")
	assert.EqualError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "ga-2"}), "blocked")
	assert.NoError(t, bl.CheckProvision(blocklist.OperationContext{PlanName: "trial", GlobalAccountID: "other-ga"}))
}

func TestGAInclusion_List_EmptySegment_IsError(t *testing.T) {
	path := writeYAML(t, "provision: '\"blocked\",\"plan=trial\",\"GA=ga-1,,ga-2\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestReadFromFile_NotFound(t *testing.T) {
	_, err := blocklist.ReadFromFile("/nonexistent/path.yaml")
	assert.Error(t, err)
}

func TestParseRule_MissingOpeningQuote(t *testing.T) {
	path := writeYAML(t, "provision:\n  - 'no-quote,plan=aws'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_MissingClosingQuote(t *testing.T) {
	path := writeYAML(t, "provision:\n  - '\"unterminated'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_TokenWithoutEquals(t *testing.T) {
	path := writeYAML(t, "provision:\n  - '\"msg\",\"noequals\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_UnknownKey(t *testing.T) {
	// Unknown keys with = operator are rejected.
	path := writeYAML(t, "provision:\n  - '\"msg\",\"SUBACCOUNT=id1\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_UnknownNegationKey(t *testing.T) {
	path := writeYAML(t, "provision:\n  - '\"msg\",\"SUBACCOUNT!=id1\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestReadFromFile_UnknownTopLevelKey(t *testing.T) {
	// "planUpgarde" is a typo — must fail fast rather than silently ignore.
	path := writeYAML(t, "planUpgarde:\n  - '\"msg\",\"plan=aws\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_TrailingComma(t *testing.T) {
	// A trailing comma would silently drop the incomplete token, turning a
	// scoped rule into a global "match all plans" rule — must be a parse error.
	path := writeYAML(t, "provision:\n  - '\"msg\",'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_TrailingCommaAfterMessageIsError(t *testing.T) {
	// provision: '"msg",' → trailing comma, must fail loading.
	path := writeYAML(t, "provision: '\"msg\",'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_EmptyToken(t *testing.T) {
	// An empty quoted token "" is meaningless and must be rejected.
	path := writeYAML(t, "provision:\n  - '\"msg\",\"\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_EmptyPlanFilter(t *testing.T) {
	// plan= with no value must be rejected — would silently match nothing but looks like a filter.
	path := writeYAML(t, "provision: '\"msg\",\"plan=\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}

func TestParseRule_EmptyPlanSegment(t *testing.T) {
	// plan=aws,,gcp has an empty segment — must be rejected.
	path := writeYAML(t, "provision: '\"msg\",\"plan=aws,,gcp\"'\n")
	_, err := blocklist.ReadFromFile(path)
	assert.Error(t, err)
}
