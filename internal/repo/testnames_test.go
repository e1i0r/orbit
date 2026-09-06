package repo

// A test's name, read as the sentence somebody meant by it.

import (
	"strings"
	"testing"
)

// TestTheThreeShapesATestIsDeclaredIn: Go, Python, and the string every
// JavaScript runner takes.
func TestTheThreeShapesATestIsDeclaredIn(t *testing.T) {
	body := strings.Join([]string{
		"func TestRejectsNegativeAmounts(t *testing.T) {",
		"def test_rounds_half_up_to_two_decimals():",
		"it('refuses a currency it does not know', () => {",
		`test("keeps the ledger balanced", async () => {`,
	}, "\n")

	got := testNames(body)
	want := []string{
		"rejects negative amounts",
		"rounds half up to two decimals",
		"refuses a currency it does not know",
		"keeps the ledger balanced",
	}

	if len(got) != len(want) {
		t.Fatalf("read %d names: %+v", len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d is %q, want %q", i, got[i], want[i])
		}
	}
}

// TestASuiteOfTwoHundredIsNotASpecification: past a handful it is a file
// listing, and the point is to say what is at risk.
func TestASuiteOfTwoHundredIsNotASpecification(t *testing.T) {
	var body strings.Builder
	for i := range 200 {
		body.WriteString("func Test")
		body.WriteString(string(rune('A' + i%26)))
		body.WriteString("Something(t *testing.T) {\n")
	}

	if got := testNames(body.String()); len(got) != mostContracts {
		t.Errorf("read %d names out of two hundred", len(got))
	}
}

// TestWhatIsATestFile, by the conventions that cover nearly every
// repository.
func TestWhatIsATestFile(t *testing.T) {
	for _, path := range []string{
		"test_pricing.py", "pricing_test.go", "pricing.test.ts",
		"pricing.spec.js", "app/tests/pricing.py",
	} {
		if !looksLikeTests(path) {
			t.Errorf("%q is not read as a test file", path)
		}
	}

	for _, path := range []string{"pricing.py", "latest.go", "contest.js"} {
		if looksLikeTests(path) {
			t.Errorf("%q is read as a test file", path)
		}
	}
}
