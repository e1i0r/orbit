package engine

import (
	"slices"
	"strings"
	"testing"
)

// postures is every subset of the vocabulary, including the empty one.
//
// The security tests below assert things that must hold for all of them
// rather than for a couple of hand-picked examples, because the dangerous
// case is always the combination nobody thought to write down.
func postures() [][]string {
	names := []string{PermissionRead, PermissionRepo, PermissionNetwork}
	all := [][]string{}

	for mask := 0; mask < 1<<len(names); mask++ {
		var set []string

		for i, n := range names {
			if mask&(1<<i) != 0 {
				set = append(set, n)
			}
		}

		all = append(all, set)
	}

	return all
}

// argvFor builds claude's command line for a posture, failing the test if the
// posture is refused. Used by the tests that care about what comes out
// rather than about what is rejected.
func argvFor(t *testing.T, perms []string) []string {
	t.Helper()

	got, err := claudeArgs(Request{Prompt: "retry on 5xx", Permissions: perms})
	if err != nil {
		t.Fatalf("claudeArgs(%v): %v", perms, err)
	}

	return got
}

func TestPermittedAcceptsEveryNameInTheVocabulary(t *testing.T) {
	for _, set := range postures() {
		if err := Permitted(set); err != nil {
			t.Errorf("Permitted(%v) = %v, want nil", set, err)
		}
	}
}

// TestPermittedRefusesANameNobodyDefined is the whole reason the vocabulary
// is closed. A posture spelled "repository" that quietly fell through to the
// binary's own default would grant more than the flow file asked for, and
// nothing on the screen or in the record would say so.
func TestPermittedRefusesANameNobodyDefined(t *testing.T) {
	err := Permitted([]string{PermissionRead, "repository"})
	if err == nil {
		t.Fatal("a permission nobody defined was accepted")
	}

	if !strings.Contains(err.Error(), "repository") {
		t.Errorf("the error does not name the permission it refused: %v", err)
	}
}

func TestPermittedAcceptsThePostureThatAsksForNothing(t *testing.T) {
	if err := Permitted(nil); err != nil {
		t.Errorf("Permitted(nil) = %v, want nil — asking for nothing is a posture", err)
	}
}

// TestEveryPermissionMapsToAStatedArgv checks that each name in the
// vocabulary buys something visible on the command line. A name that mapped
// to nothing would be a permission the flow file grants and the engine
// silently ignores, which is the defect this task exists to close.
func TestEveryPermissionMapsToAStatedArgv(t *testing.T) {
	for _, tc := range []struct {
		perm string
		want string
	}{
		{PermissionRead, "Read"},
		{PermissionRepo, "Edit"},
		{PermissionNetwork, "WebFetch"},
	} {
		joined := strings.Join(argvFor(t, []string{tc.perm}), " ")
		if !strings.Contains(joined, tc.want) {
			t.Errorf("%q produced %q, which never mentions %q", tc.perm, joined, tc.want)
		}
	}
}

// TestAnEmptyPostureIsTheMostRestrictiveArgvRatherThanNone is the
// distinction the whole change rests on. Passing no permission flags leaves
// the binary's own default in charge, stated nowhere, on a process whose
// working directory sits inside the state root. Asking for nothing has to
// come out as an argv that says so.
//
// Both halves have to be stated, which is why the tool assertions below are
// about what the flag says and not only about what it omits. Asserting that
// the argv never mentions Edit was satisfied for free by an argv carrying no
// --allowedTools at all — the test passed while the tool question was being
// handed to whatever the machine's settings files happened to say.
func TestAnEmptyPostureIsTheMostRestrictiveArgvRatherThanNone(t *testing.T) {
	got := argvFor(t, nil)

	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "--permission-mode") {
		t.Errorf("the empty posture produced %q, which states no posture at all", joined)
	}

	if !slices.Contains(got, "plan") {
		t.Errorf("the empty posture produced %q, want the strictest mode claude names", joined)
	}

	at := slices.Index(got, "--allowedTools")
	if at < 0 || at == len(got)-1 {
		t.Fatalf("the empty posture produced %q, which leaves the tool half to the machine's settings", joined)
	}

	if got[at+1] != noTools {
		t.Errorf("the empty posture allowed %q, want %q — the list is stated and grants nothing", got[at+1], noTools)
	}

	for _, forbidden := range []string{"Edit", "Write", "Bash", "WebFetch", "WebSearch"} {
		if strings.Contains(joined, forbidden) {
			t.Errorf("the empty posture allowed %s: %q", forbidden, joined)
		}
	}
}

// TestTheEmptySentinelNamesNoToolThisPackageGrants is what makes noTools
// safe to write on a command line. It is a name chosen because nothing
// matches it; a tool that arrived later under that name would turn the
// posture that asks for nothing into a posture that asks for something, and
// this is the assertion that would notice.
func TestTheEmptySentinelNamesNoToolThisPackageGrants(t *testing.T) {
	if slices.Contains(toolOrder, noTools) {
		t.Fatalf("%q is a tool this package grants, so the posture that asks for nothing is asking for it", noTools)
	}
}

// TestEveryPostureStatesItsToolList sweeps the same eight subsets as the
// rules below. A posture that named no tools used to produce no flag, which
// read on `ps` as an absence and behaved as a delegation; every posture now
// says what it asked for, including the one that asked for nothing.
func TestEveryPostureStatesItsToolList(t *testing.T) {
	for _, set := range postures() {
		got := argvFor(t, set)

		at := slices.Index(got, "--allowedTools")
		if at < 0 || at == len(got)-1 {
			t.Errorf("posture %v produced %q, which states no tool list", set, strings.Join(got, " "))
			continue
		}

		if got[at+1] == "" {
			t.Errorf("posture %v produced an empty tool list, which a binary may read as no list at all", set)
		}
	}
}

// TestOnlyRepoCanWrite and its siblings pin the vocabulary's meanings
// against the argv, one sentence of the definition at a time.
func TestOnlyRepoCanWrite(t *testing.T) {
	for _, set := range postures() {
		joined := strings.Join(argvFor(t, set), " ")

		writes := strings.Contains(joined, "Edit") || strings.Contains(joined, "Write") || strings.Contains(joined, "Bash")
		if slices.Contains(set, PermissionRepo) != writes {
			t.Errorf("posture %v produced %q — repo is the only name that may write", set, joined)
		}
	}
}

func TestOnlyNetworkReachesTheWeb(t *testing.T) {
	for _, set := range postures() {
		joined := strings.Join(argvFor(t, set), " ")

		web := strings.Contains(joined, "WebFetch") || strings.Contains(joined, "WebSearch")
		if slices.Contains(set, PermissionNetwork) != web {
			t.Errorf("posture %v produced %q — network is the only name that reaches the web", set, joined)
		}
	}
}

// TestNoEngineIsEverHandedTheDangerousFlag is a security assertion, not a
// style one, and it is written as a test because the shortcut it forbids is
// the one a hurried change reaches for first.
//
// Mapping repo onto --dangerously-skip-permissions stops headless runs from
// prompting, which makes it look like the fix. It is not a posture: it turns
// the question off rather than answering it. The plan refuses it explicitly,
// and the same refusal covers --permission-mode bypassPermissions, which is
// the same grant under a politer name.
func TestNoEngineIsEverHandedTheDangerousFlag(t *testing.T) {
	for _, set := range postures() {
		joined := strings.Join(argvFor(t, set), " ")
		for _, forbidden := range []string{"--dangerously-skip-permissions", "bypassPermissions"} {
			if strings.Contains(joined, forbidden) {
				t.Errorf("posture %v produced %q, which carries %s", set, joined, forbidden)
			}
		}
	}
}

// TestNoEngineAutoApprovesEditsWithoutBash keeps one half of an expensive
// lesson from being relearned. A mode that auto-approved edits but not the
// commands those edits needed put a headless run into a loop of identical
// refusals for eight minutes, billed by the turn. The mechanism was the
// denials, not the mode, and nothing in this argv bounds turns or cost — the
// comment on the tool lists says so plainly, and the ceiling belongs to a
// later task. What this test can hold is the mode: whatever repo means, it
// may not mean that.
func TestNoEngineAutoApprovesEditsWithoutBash(t *testing.T) {
	for _, set := range postures() {
		joined := strings.Join(argvFor(t, set), " ")
		if strings.Contains(joined, "acceptEdits") {
			t.Errorf("posture %v produced %q, which auto-approves edits and not the commands they need", set, joined)
		}
	}
}

func TestClaudeArgsRefuseAPostureNobodyDefined(t *testing.T) {
	if _, err := claudeArgs(Request{Prompt: "x", Permissions: []string{"admin"}}); err == nil {
		t.Error("claude was handed a command line built from a permission nobody defined")
	}
}

// TestThePostureDoesNotDependOnTheOrderItWasWritten keeps the argv a
// function of the posture rather than of how a flow file happened to spell
// it, so two flows that grant the same thing produce the same command line.
func TestThePostureDoesNotDependOnTheOrderItWasWritten(t *testing.T) {
	one := strings.Join(argvFor(t, []string{PermissionNetwork, PermissionRead}), " ")

	two := strings.Join(argvFor(t, []string{PermissionRead, PermissionNetwork}), " ")
	if one != two {
		t.Errorf("the same posture produced two command lines:\n%s\n%s", one, two)
	}
}

// TestRepoImpliesRead states one thing the definitions leave implicit: a
// posture that may write inside the worktree can obviously read it, and a
// flow file that says only "repo" should not lose the ability to open a
// file.
func TestRepoImpliesRead(t *testing.T) {
	joined := strings.Join(argvFor(t, []string{PermissionRepo}), " ")
	if !strings.Contains(joined, "Read") {
		t.Errorf("repo produced %q, which cannot read the worktree it may write to", joined)
	}
}
