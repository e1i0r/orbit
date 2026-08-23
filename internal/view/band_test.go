package view

import (
	"reflect"
	"testing"
)

// generated is the set the partition is tested over: every ordering the
// fold tables describe, plus a hand-built task for every state the fold can
// leave behind crossed with the fields a later reader may change. The last
// one carries a state no constant names, which is the only way a value from
// outside this package's own vocabulary can reach BandOf.
func generated() []Task {
	var tasks []Task
	for _, c := range allCases() {
		tasks = append(tasks, Fold(c.events))
	}
	for s := state(0); s < stateCount; s++ {
		for _, live := range []bool{false, true} {
			for _, read := range []bool{false, true} {
				for _, damaged := range []int{0, 3} {
					tasks = append(tasks, Task{ID: "ACME-1", state: s, Live: live, Read: read, Damaged: damaged})
				}
			}
		}
	}
	tasks = append(tasks, Task{ID: "ACME-1", state: stateCount + 7})
	return tasks
}

// TestBandsPartitionEveryTask is the invariant this package exists for. The
// header counts a band and the list draws it, and in v1 those were two rules
// that agreed by inspection until they did not — `Pending 4` above a
// different four, three separate times. Here the header and the list are the
// same call, so the two things left to prove are that the call is total (its
// answer is always one of the four Bands draws) and that it is a function
// (asking again gives the same answer).
//
// The counting pass below is the header. The four filtering passes are the
// list, one per band, each asking BandOf again from scratch. A band outside
// Bands leaves a task in no list and the totals disagree; an answer that
// changes between calls puts a task in two lists, or in none.
func TestBandsPartitionEveryTask(t *testing.T) {
	tasks := generated()

	header := map[Band]int{}
	for _, task := range tasks {
		header[BandOf(task)]++
	}

	answers := make([]int, len(tasks))
	total := 0
	for _, band := range Bands() {
		drawn := 0
		for i, task := range tasks {
			if BandOf(task) == band {
				answers[i]++
				drawn++
			}
		}
		if drawn != header[band] {
			t.Errorf("the header says %d tasks are %s and the list draws %d of them", header[band], band, drawn)
		}
		total += drawn
	}
	if total != len(tasks) {
		t.Errorf("the four bands hold %d tasks between them, and there are %d", total, len(tasks))
	}
	for i, n := range answers {
		if n != 1 {
			t.Errorf("task %d answers %d bands, want exactly one: %+v", i, n, tasks[i])
		}
	}
}

// TestFoldStoresTheAnswerThePredicateGives closes the other half of the same
// hole: Task.Band is written once by Fold and read by everything that draws
// a row, so a fold that decided the band by its own reasoning would be a
// second rule again, and this is where that would show.
func TestFoldStoresTheAnswerThePredicateGives(t *testing.T) {
	for _, c := range allCases() {
		got := Fold(c.events)
		if got.Band != BandOf(got) {
			t.Errorf("%s: Fold stored band %s and the predicate says %s", c.name, got.Band, BandOf(got))
		}
	}
}

// TestEveryStateHasABand walks the states themselves rather than the tasks,
// so a state added to the fold without a band to put it in fails here
// instead of arriving on screen in whatever band the default happens to be.
func TestEveryStateHasABand(t *testing.T) {
	want := map[state]Band{
		stateNew:       ToDo,
		stateRunning:   Running,
		stateHeld:      Running,
		stateWaiting:   NeedsYou,
		stateFailed:    NeedsYou,
		stateTimedOut:  NeedsYou,
		stateAbandoned: NeedsYou,
		stateCancelled: Done,
		stateFinished:  Done,
	}
	if len(want) != int(stateCount) {
		t.Fatalf("this test names %d states and the fold has %d — place the new one in a band", len(want), stateCount)
	}
	for s, band := range want {
		if got := BandOf(Task{state: s}); got != band {
			t.Errorf("state %d is banded %s, want %s", s, got, band)
		}
	}
}

// TestLiveDoesNotDecideTheBand pins the rule task 6's reconciliation rests
// on: a run whose process is gone is only abandoned once somebody has
// written task.abandoned into the record. If Live moved a task out of
// Running on its own, the window would know something no other reader of
// the record could know, and `orbit show` would disagree with the screen.
func TestLiveDoesNotDecideTheBand(t *testing.T) {
	for _, c := range allCases() {
		dead := Fold(c.events)
		alive := dead
		alive.Live = true
		if BandOf(alive) != BandOf(dead) {
			t.Errorf("%s: setting Live moved the task from %s to %s", c.name, BandOf(dead), BandOf(alive))
		}
	}
}

// TestBandsAreDrawnInOrder covers the order itself, and that a caller who
// keeps or reorders the slice cannot change what the next caller is given —
// a package variable handed out would be package state, and this package
// holds none.
func TestBandsAreDrawnInOrder(t *testing.T) {
	want := []Band{NeedsYou, Running, ToDo, Done}
	if got := Bands(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Bands = %v, want %v", got, want)
	}
	scribbled := Bands()
	scribbled[0] = Done
	if got := Bands(); !reflect.DeepEqual(got, want) {
		t.Errorf("Bands = %v after a caller wrote to a previous result, want %v", got, want)
	}
}
