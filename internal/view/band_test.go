package view

import (
	"reflect"
	"testing"
)

// generated is the set the partition is tested over: every ordering the fold
// tables describe, plus a hand-built task for every band somebody can write
// down, crossed with the fields a later reader may change on a Task after
// the fold produced it. The last two bands are values no constant names,
// which is how a Band from outside this package's vocabulary — a fixture
// with a slip in it, a number read off a file — reaches BandOf.
func generated() []Task {
	var tasks []Task
	for _, c := range allCases() {
		tasks = append(tasks, Fold(c.events))
	}

	for _, band := range append(Bands(), Band(17), Band(-1)) {
		for _, live := range []Liveness{LiveFree, LiveHeld, LiveUnknown} {
			for _, read := range []bool{false, true} {
				for _, damaged := range []int{0, 3} {
					tasks = append(tasks, Task{ID: "ACME-1", Band: band, Live: live, Read: read, Damaged: damaged})
				}
			}
		}
	}

	return tasks
}

// TestBandsPartitionEveryTask is the invariant this package exists for. The
// header counts a band and the list draws it, and in v1 those were two rules
// that agreed by inspection until they did not — `Pending 4` above a
// different four, three separate times. Here the header and the list are the
// same call, so three things have to hold: the call is a function (asking
// again gives the same answer), it is total (its answer is always one of the
// four Bands draws), and no task answers two.
func TestBandsPartitionEveryTask(t *testing.T) {
	tasks := generated()

	// Being a function is checked head-on rather than being left to the
	// passes below to notice. They ask BandOf a fixed number of times per
	// task — once counting and once per band drawn — and a predicate that
	// alternated its answer would have that alternation cancel out over an
	// even number of calls and go unseen. Whether a defect is caught should
	// not come down to the parity of a loop bound.
	for i, task := range tasks {
		first := BandOf(task)
		for range 8 {
			if got := BandOf(task); got != first {
				t.Fatalf("task %d was banded %s and then %s: BandOf is not a function of its argument", i, first, got)
			}
		}
	}

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

// TestABandWrittenDownIsTheBandAnswered is the seam every later task builds
// fixtures through. A golden row is typed out as `Task{Band: Running}` and
// the header counts it with BandOf, so if the predicate answered from
// anything a fixture author cannot set, the fixture and the count would
// disagree — and the disagreement would be frozen into a golden. The
// unexported state is crossed in deliberately: it must not be able to
// override what is written down.
func TestABandWrittenDownIsTheBandAnswered(t *testing.T) {
	for _, band := range Bands() {
		if got := BandOf(Task{Band: band}); got != band {
			t.Errorf("a task written down as %s is banded %s", band, got)
		}

		for s := state(0); s < stateCount; s++ {
			if got := BandOf(Task{Band: band, state: s}); got != band {
				t.Errorf("state %d moved a task written down as %s to %s", s, band, got)
			}
		}
	}
}

// TestABandNobodyNamesIsAnsweredAsNeedsYou pins the normalisation, so that
// it is a decision rather than whatever the switch happened to do. A Band
// outside the four would otherwise be drawn by no list while still being
// counted, which is the partition failing quietly — and NeedsYou is the band
// to fail into, because it is the one somebody looks at.
func TestABandNobodyNamesIsAnsweredAsNeedsYou(t *testing.T) {
	for _, band := range []Band{Band(-1), Band(4), Band(17)} {
		if got := BandOf(Task{Band: band}); got != NeedsYou {
			t.Errorf("Band(%d) is banded %s, want %s", int(band), got, NeedsYou)
		}
	}
}

// TestFoldStoresTheAnswerThePredicateGives closes the other half: Task.Band
// is written once by Fold and read by everything, so a fold that decided the
// band by its own reasoning would be a second rule again. The first check is
// where that shows; the second is that Fold never stores a band outside the
// four, since a stored Band(9) would be normalised on the way out and the
// row and the count would drift apart by one.
func TestFoldStoresTheAnswerThePredicateGives(t *testing.T) {
	for _, c := range allCases() {
		got := Fold(c.events)
		if want := bandOfState(got.state); got.Band != want {
			t.Errorf("%s: Fold stored band %s and the state it folded into is banded %s", c.name, got.Band, want)
		}

		if got.Band != BandOf(got) {
			t.Errorf("%s: Fold stored band %s and BandOf answers %s, so it stored one of no band", c.name, got.Band, BandOf(got))
		}
	}
}

// TestEveryStateHasABand walks the states themselves rather than the tasks,
// so a state added to the fold without a band to put it in fails here
// instead of arriving on screen in whatever band the default happens to be.
func TestEveryStateHasABand(t *testing.T) {
	want := map[state]Band{
		stateNew:         ToDo,
		stateRunning:     Running,
		stateHeld:        Running,
		stateWaiting:     NeedsYou,
		statePhaseFailed: NeedsYou,
		stateFailed:      NeedsYou,
		stateTimedOut:    NeedsYou,
		stateAbandoned:   NeedsYou,
		stateCancelled:   Done,
		stateFinished:    Done,
		stateStuck:       NeedsYou,
		stateOverBudget:  NeedsYou,
	}
	if len(want) != int(stateCount) {
		t.Fatalf("this test names %d states and the fold has %d — place the new one in a band", len(want), stateCount)
	}

	for s, band := range want {
		if got := bandOfState(s); got != band {
			t.Errorf("state %d is banded %s, want %s", s, got, band)
		}
	}
}

// TestAnUnfoldedTaskIsToDo is why ToDo is the zero value. A Task nobody has
// folded — the one a caller declares before it has anything to put in it —
// is a task written down and nothing more, and that is a true sentence
// rather than a convenient one.
func TestAnUnfoldedTaskIsToDo(t *testing.T) {
	if got := BandOf(Task{}); got != ToDo {
		t.Errorf("an empty task is banded %s, want %s", got, ToDo)
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

		alive.Live = LiveHeld
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
