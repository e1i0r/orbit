package kin

// What a process started, and how to stop all of it.
//
// Killing a process group is most of the answer and not the whole of it. A
// child that puts itself in a group of its own leaves the one being
// signalled, and that is not a rare trick: opencode runs every shell
// command its bash tool is asked for in a new group. A run cancelled while
// one of those was going killed the engine, killed the engine's group, and
// left `/bin/zsh -c "sleep 120 && echo done >> notes.txt"` behind with
// ppid 1 — which then wrote into the worktree of a task that had already
// been cancelled, minutes after the record said it was over.
//
// So what is stopped is the family and not the group: every process
// descended from the one being killed, whatever group each of them chose.
// The tree is read before anything is signalled, because a child re-parents
// to init the moment its parent dies and a tree read afterwards has already
// lost them.

import (
	"os/exec"
	"strconv"
	"strings"
)

// Stop kills a process and everything it started.
//
// Descendants first and deepest first, so that nothing is orphaned into a
// tree this has already walked past; the process itself last, with its
// group, which is what reaches the children that stayed in it.
//
// Nothing is reported. This runs while a run is being cancelled, on top of
// whatever answer that run already has, and a process that was already gone
// is the ordinary case rather than a failure: the caller asked for it to be
// stopped and it is stopped.
func Stop(pid int) error {
	if pid <= 1 {
		return nil
	}

	for _, child := range descendants(pid) {
		kill(child)
	}

	killGroup(pid)
	kill(pid)

	return nil
}

// descendants is every process under pid, deepest first.
//
// Breadth first down the table and then reversed, which is the same order
// as walking up from the leaves: a parent is always signalled after the
// children it could still spawn more of.
func descendants(pid int) []int {
	children := table()

	var (
		out  []int
		next = []int{pid}
	)

	// A table read once cannot cycle, but a pid that names itself as its
	// own parent would loop here for ever, and a process table is read
	// from a program this one does not control.
	seen := map[int]bool{pid: true}

	for len(next) > 0 {
		at := next[0]
		next = next[1:]

		for _, child := range children[at] {
			if seen[child] {
				continue
			}

			seen[child] = true

			out = append(out, child)
			next = append(next, child)
		}
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}

// table is the process table as a map of parent to children.
//
// Read with ps rather than through a system call, because the call is a
// different one on every platform — sysctl on darwin, /proc on linux — and
// what is being asked for is two numbers per line. A table that cannot be
// read is an empty one: the group is still signalled, which is what this
// did before there was a tree to walk.
func table() map[int][]int {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}

	children := map[int][]int{}

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		parent, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		children[parent] = append(children[parent], pid)
	}

	return children
}
