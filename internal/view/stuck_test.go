package view

import "testing"

func ending(key string, band Band, min int) Task {
	return Task{Band: band, Since: at(min), Reason: Reason{Key: key}}
}

func stuckAt(min int) Task { return ending(ReasonStuck, NeedsYou, min) }

func TestStuckStreakCountsBackFromTheNewestEnding(t *testing.T) {
	for _, tc := range []struct {
		name  string
		tasks []Task
		want  int
	}{
		{
			name:  "nothing has ended",
			tasks: []Task{{Band: Running, Since: at(1)}},
			want:  0,
		},
		{
			name:  "three in a row",
			tasks: []Task{stuckAt(1), stuckAt(2), stuckAt(3)},
			want:  3,
		},
		{
			// The order the tasks arrive in is the board's, which is the
			// order they are drawn in — so the run is counted on when they
			// stopped.
			name:  "a run that finished after them ends the count",
			tasks: []Task{stuckAt(1), stuckAt(2), ending("", Done, 3)},
			want:  0,
		},
		{
			name:  "a run that finished before them does not",
			tasks: []Task{ending("", Done, 1), stuckAt(2), stuckAt(3)},
			want:  2,
		},
		{
			name:  "a run still going is not an ending",
			tasks: []Task{stuckAt(1), {Band: Running, Since: at(2)}, stuckAt(3)},
			want:  2,
		},
		{
			// Held and waiting are stops inside a run that has not ended:
			// something is still holding a worktree.
			name:  "a run stopped at a gate is not an ending",
			tasks: []Task{stuckAt(1), ending(ReasonGate, NeedsYou, 2), stuckAt(3)},
			want:  2,
		},
		{
			name:  "a cancelled run is an ending that is not stuck",
			tasks: []Task{stuckAt(1), stuckAt(2), ending(ReasonCancelled, Done, 3)},
			want:  0,
		},
		{
			name:  "a failed run is an ending that is not stuck",
			tasks: []Task{stuckAt(1), stuckAt(2), ending(ReasonFailed, NeedsYou, 3)},
			want:  0,
		},
	} {
		got := StuckStreak(tc.tasks)
		if len(got) != tc.want {
			t.Errorf("%s: StuckStreak is %d long, want %d", tc.name, len(got), tc.want)
			continue
		}

		for i := 1; i < len(got); i++ {
			if got[i].Since.After(got[i-1].Since) {
				t.Errorf("%s: StuckStreak is not newest first: %v then %v", tc.name, got[i-1].Since, got[i].Since)
			}
		}
	}
}
