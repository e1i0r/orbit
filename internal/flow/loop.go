package flow

// A loop: phases that go round until something verifiable says they can
// stop.

import "fmt"

// Loop is a group of phases repeated until every check passes, and never
// more than Max times.
//
// The three fields are three rules, and a loop missing any of them is
// refused at load rather than found out at two in the morning:
//
// Until is what says the work is done, and it is a command's exit code —
// never a model saying it is finished. A model asked whether its own work
// passes is being asked to mark its own paper, and the answer is yes.
//
// Max is what keeps an impossible check from spending the whole quota
// window on a wall. There is no default: a flow that declares a loop has to
// say how much rope it wants, because the right number is a property of the
// work and every wrong one is expensive.
//
// Phases is what goes round. They are ordinary phases and they are told what
// the check said last time, which is the difference between three tries and
// the same try three times.
type Loop struct {
	Phases []Phase `json:"phases"`
	Until  []Gate  `json:"until"`
	Max    int     `json:"max"`
}

// validate reports the first thing that would make a loop unrunnable.
//
// A loop inside a loop is refused. The cap of the outer one would stop
// bounding anything — three turns of a loop that itself runs three is nine,
// and nobody reading the flow file would see the nine — and no flow has yet
// needed one.
func (l *Loop) validate(flow, phase string) error {
	if l == nil {
		return nil
	}

	if len(l.Phases) == 0 {
		return fmt.Errorf("flow %q: the loop at phase %q repeats no phases", flow, phase)
	}

	if len(l.Until) == 0 {
		return fmt.Errorf("flow %q: the loop at phase %q has nothing that says when it is done", flow, phase)
	}

	if l.Max < 1 {
		return fmt.Errorf("flow %q: the loop at phase %q allows %d turns, and a loop with no cap is a loop with no end",
			flow, phase, l.Max)
	}

	for _, g := range l.Until {
		if g.Name == "" || g.Command == "" {
			return fmt.Errorf("flow %q: the loop at phase %q has a check with no name or no command", flow, phase)
		}
	}

	for _, p := range l.Phases {
		if p.Loop != nil {
			return fmt.Errorf("flow %q: the loop at phase %q holds another loop at %q, and then no number in the file says how many times %q runs",
				flow, phase, p.Name, p.Name)
		}

		if err := p.validate(flow, len(l.Phases)); err != nil {
			return err
		}
	}

	return nil
}
