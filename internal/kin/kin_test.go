package kin

// A child that left the group, which is what a real engine does.

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestStopKillsAChildThatLeftTheGroup.
//
// The shape is opencode's, read off a real run: the engine leads a group of
// its own, and the shell it runs a tool call in leads a third. Killing the
// engine's group reached the engine and nothing else, so the command went
// on writing into the worktree of a task that had already been cancelled.
func TestStopKillsAChildThatLeftTheGroup(t *testing.T) {
	if os.Getenv("KIN_HELPER") != "" {
		helper(t)

		return
	}

	// The test binary again, as its own helper: what this needs is a
	// process that starts something and stays up, and the binary already
	// running is one.
	child := exec.Command(os.Args[0], "-test.run=TestStopKillsAChildThatLeftTheGroup", "-test.timeout=60s")

	child.Env = append(os.Environ(), "KIN_HELPER=1")
	child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	said, err := child.StdoutPipe()
	if err != nil {
		t.Fatalf("read what the helper says: %v", err)
	}

	if err := child.Start(); err != nil {
		t.Fatalf("start the helper: %v", err)
	}

	reaped := false

	t.Cleanup(func() {
		_ = Stop(child.Process.Pid) //nolint:errcheck // whatever is left, take it

		if !reaped {
			_, _ = child.Process.Wait() //nolint:errcheck // and reap it
		}
	})

	grandchild := saidPid(t, said)

	// The shape this test exists for: the grandchild is not in the group
	// that is about to be signalled.
	pgid, err := syscall.Getpgid(grandchild)
	if err != nil {
		t.Fatalf("read the grandchild's group: %v", err)
	}

	if pgid == child.Process.Pid {
		t.Fatalf("the grandchild is in the child's group, so this test proves nothing")
	}

	if err := Stop(child.Process.Pid); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Reaped before it is asked about: a process nobody has waited for is
	// a zombie, and a zombie answers kill(pid, 0) as though it were alive
	// — which is the same thing Start's own comment says about the pid it
	// hands back.
	if _, err := child.Process.Wait(); err != nil {
		t.Fatalf("wait for the helper: %v", err)
	}

	reaped = true

	if alive(grandchild) {
		t.Error("a child that left the group outlived the cancel")
	}

	if alive(child.Process.Pid) {
		t.Error("the process itself outlived the cancel")
	}
}

// helper is the other half of this test, running as its own process: it
// starts something in a group of its own, says which pid that is, and then
// waits to be killed.
func helper(t *testing.T) {
	t.Helper()

	sleeper := exec.Command("sleep", "45")
	sleeper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := sleeper.Start(); err != nil {
		t.Fatalf("the helper could not start a sleeper: %v", err)
	}

	if _, err := os.Stdout.WriteString("grandchild " + strconv.Itoa(sleeper.Process.Pid) + "\n"); err != nil {
		t.Fatalf("the helper could not say which pid: %v", err)
	}

	time.Sleep(45 * time.Second)
}

// saidPid is the pid the helper printed, or a failed test.
func saidPid(t *testing.T, said interface{ Read([]byte) (int, error) }) int {
	t.Helper()

	line, err := bufio.NewReader(said).ReadString('\n')
	if err != nil {
		t.Fatalf("the helper said nothing: %v", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "grandchild ")))
	if err != nil {
		t.Fatalf("the helper said %q: %v", line, err)
	}

	return pid
}

// alive is whether a pid still answers, after long enough for a SIGKILL to
// have landed.
func alive(pid int) bool {
	for range 20 {
		if err := syscall.Kill(pid, 0); err != nil {
			return false
		}

		time.Sleep(100 * time.Millisecond)
	}

	return true
}
