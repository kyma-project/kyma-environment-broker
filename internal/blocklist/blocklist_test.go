package blocklist_test

import (
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/blocklist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePlanValidator accepts a fixed set of known plan names (case-insensitive).
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

// writeYAML writes a blocklist YAML file and returns the path.
func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "blocklist-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// parseInline builds a flat-format blocklist YAML for a single operation type
// and wires the test PlanValidator.
func parseInline(op string, rules ...string) (blocklist.OperationBlocklist, error) {
	yaml := op + ":\n"
	for _, r := range rules {
		yaml += "  - '" + r + "'\n"
	}
	f, err := os.CreateTemp("", "bl-*.yaml")
	if err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	defer os.Remove(f.Name())
	if _, err = f.WriteString(yaml); err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	f.Close()
	bl, err := blocklist.ReadFromFile(f.Name())
	if err != nil {
		return blocklist.OperationBlocklist{}, err
	}
	return bl.WithPlanValidator(testPlans), nil
}

// --- parser ---

func TestParseRule_MessageOnly(t *testing.T) {
	bl, err := parseInline("provision", `"always blocked"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("any", "any"), "always blocked")
}

func TestParseRule_WithPlan(t *testing.T) {
	bl, err := parseInline("provision", `"blocked {plan}","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("aws", "ga1"), "blocked aws")
	assert.NoError(t, bl.CheckProvision("gcp", "ga1"))
}

func TestParseRule_WithGAList(t *testing.T) {
	bl, err := parseInline("provision", `"blocked GA={GA}","GA=id1,id2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("aws", "id1"), "blocked GA=id1")
	assert.EqualError(t, bl.CheckProvision("gcp", "id2"), "blocked GA=id2")
	assert.NoError(t, bl.CheckProvision("aws", "id3"))
}

func TestParseRule_WithGANegation(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","GA=!id2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("aws", "id1"), "blocked")
	assert.NoError(t, bl.CheckProvision("aws", "id2"))
}

func TestParseRule_PlanAndGA(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=aws","GA=id1,id2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("aws", "id1"), "blocked")
	assert.NoError(t, bl.CheckProvision("gcp", "id1")) // plan mismatch
	assert.NoError(t, bl.CheckProvision("aws", "id3")) // GA mismatch
}

func TestParseRule_SANegation(t *testing.T) {
	bl, err := parseInline("update", `"update blocked for {SA}","SA=!id2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckUpdate("aws", "id1"), "update blocked for id1")
	assert.NoError(t, bl.CheckUpdate("aws", "id2"))
}

func TestParseRule_HRAndPRParsedButNotChecked(t *testing.T) {
	bl, err := parseInline("provision", `"blocked","plan=aws","HR=eu","PR=cf-eu10"`)
	require.NoError(t, err)
	// HR and PR don't affect matching — only plan matters here
	assert.EqualError(t, bl.CheckProvision("aws", "ga1"), "blocked")
	assert.NoError(t, bl.CheckProvision("gcp", "ga1"))
}

// --- operation-type checks ---

func TestCheckPlanUpgrade(t *testing.T) {
	bl, err := parseInline("planUpgrade", `"plan upgrade blocked for {plan}","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckPlanUpgrade("aws"), "plan upgrade blocked for aws")
	assert.NoError(t, bl.CheckPlanUpgrade("gcp"))
}

func TestCheckDeprovision(t *testing.T) {
	bl, err := parseInline("deprovision", `"deprovision blocked plan={plan} GA={GA}","plan=gcp","GA=id1,id2"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckDeprovision("gcp", "id1"), "deprovision blocked plan=gcp GA=id1")
	assert.NoError(t, bl.CheckDeprovision("aws", "id1"))
	assert.NoError(t, bl.CheckDeprovision("gcp", "id3"))
}

func TestCheckRules_MultipleRules_FirstMatchWins(t *testing.T) {
	bl, err := parseInline("provision", `"first","plan=aws"`, `"second","plan=aws"`)
	require.NoError(t, err)
	assert.EqualError(t, bl.CheckProvision("aws", "ga"), "first")
}

func TestCheckRules_EmptyBlocklist(t *testing.T) {
	var bl blocklist.OperationBlocklist
	assert.NoError(t, bl.CheckProvision("aws", "ga"))
	assert.NoError(t, bl.CheckUpdate("aws", "sa"))
	assert.NoError(t, bl.CheckPlanUpgrade("aws"))
	assert.NoError(t, bl.CheckDeprovision("aws", "ga"))
}

// --- YAML: single string vs list ---

func TestReadFromFile_FullExample(t *testing.T) {
	yaml := `
provision:
  - '"provisioning is blocked for {plan} plan and global accounts {GA}","plan=aws","GA=id1,id2"'
  - '"provisioning is blocked for {plan} plan and global accounts {GA}","plan=gcp","GA=id1,id2"'
update: '"update is blocked for subaccount not being {SA}","SA=!id2"'
planUpgrade: '"plan upgrade is blocked for plan {plan}","plan=aws"'
deprovision: '"deprovisioning is blocked for this {plan} and global accounts {GA}","plan=gcp","GA=id1,id2"'
`
	path := writeYAML(t, yaml)
	bl, err := blocklist.ReadFromFile(path)
	require.NoError(t, err)
	bl = bl.WithPlanValidator(testPlans)

	// provision list
	assert.EqualError(t, bl.CheckProvision("aws", "id1"), "provisioning is blocked for aws plan and global accounts id1")
	assert.EqualError(t, bl.CheckProvision("gcp", "id2"), "provisioning is blocked for gcp plan and global accounts id2")
	assert.NoError(t, bl.CheckProvision("azure", "id1"))

	// update single string — blocks all except SA=id2
	assert.EqualError(t, bl.CheckUpdate("any", "id1"), "update is blocked for subaccount not being id1")
	assert.NoError(t, bl.CheckUpdate("any", "id2"))

	// planUpgrade single string
	assert.EqualError(t, bl.CheckPlanUpgrade("aws"), "plan upgrade is blocked for plan aws")
	assert.NoError(t, bl.CheckPlanUpgrade("gcp"))

	// deprovision single string
	assert.EqualError(t, bl.CheckDeprovision("gcp", "id1"), "deprovisioning is blocked for this gcp and global accounts id1")
	assert.NoError(t, bl.CheckDeprovision("aws", "id1"))
}

func TestMatchesPlan_UnknownPlanInRuleDoesNotMatch(t *testing.T) {
	// "notaplan" is not in testPlans, so the rule should never fire
	bl, err := parseInline("provision", `"blocked","plan=notaplan"`)
	require.NoError(t, err)
	assert.NoError(t, bl.CheckProvision("notaplan", "ga"))
}

// --- error cases ---

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
