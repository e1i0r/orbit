//go:build !windows

package lowly

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// stepVar makes this test binary the step: see TestMain.
const stepVar = "ORBIT_LOWLY_TEST_STEP"

// TestMain lets the test binary stand in for orbit: started with stepVar
// set, it does what `orbit __lowered` does and becomes the program.
func TestMain(m *testing.M) {
	if os.Getenv(stepVar) != "" {
		if err := Exec(os.Args[1:]); err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}

// TestTheProgramRunsAtTheLowestPriority, and what it starts too. golangci
// at 1000% under a session opened from the cockpit froze the machine; run
// behind the step, it would have given way.
func TestTheProgramRunsAtTheLowestPriority(t *testing.T) {
	// The shell reports its own niceness and then its child's, which is
	// what make check starting golangci looks like.
	cmd := exec.Command(os.Args[0], "--", "sh", "-c", "ps -o nice= -p $$; sh -c 'ps -o nice= -p $$'")

	cmd.Env = append(os.Environ(), stepVar+"=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the step did not run the program: %v: %s", err, out)
	}

	read := strings.Fields(string(out))
	if len(read) != 2 {
		t.Fatalf("want the shell's niceness and its child's, got %q", out)
	}

	for i, got := range read {
		if got != "19" {
			t.Errorf("process %d ran at niceness %s, want 19", i, got)
		}
	}
}

// TestAStepWithNothingBehindItRefuses rather than running nothing.
func TestAStepWithNothingBehindItRefuses(t *testing.T) {
	if err := Exec([]string{"--"}); err == nil {
		t.Error("the step ran with no program behind it")
	}
}

// TestArgsPutTheStepFirst. The child is orbit itself, and the step is how
// it knows to become the program rather than run a command.
func TestArgsPutTheStepFirst(t *testing.T) {
	got := strings.Join(Args("claude", "--resume", "x"), " ")
	if got != Arg+" -- claude --resume x" {
		t.Errorf("the step is started as %q", got)
	}
}
