package known

// The tray: what you said that nobody has answered yet.
//
// You tell the supervisor "never push a pull request without the tests
// passing". That sentence is already the answer — it is about your code, in
// your words, and you meant it — so the only thing left is to be asked
// whether to keep it. This is where the asking happens.
//
// Which band it sits in, and how the cursor reaches it, is in bands.go.

// keepAsSaid keeps the sentence under the cursor word for word and where the
// work was, which is the answer when there is nothing to correct.
func (s State) keepAsSaid(e Env) (State, Out) {
	one, waiting := s.onSaid()
	if !waiting {
		return s, Out{}
	}

	return s.keepWith(one, one.Text, one.Gate, one.Where, e)
}

// keepWith writes it down as a fact of yours, and takes it out of the tray.
//
// The words and the place are parameters rather than the sentence's own,
// because correcting is how most of these are accepted: what you meant is
// what you typed the second time, and where it belongs is almost always
// narrower than where you happened to say it.
func (s State) keepWith(one Said, phrase, check, where string, e Env) (State, Out) {
	if e.Keep == nil {
		return s, Out{}
	}

	if err := e.Keep(one.At, phrase, check, where); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), said(e.Words.T("knowledge.kept",
		"Orbit knows it now, and every phase will be told"))
}

// dropSaid says it was not a rule.
func (s State) dropSaid(e Env) (State, Out) {
	one, waiting := s.onSaid()
	if !waiting || e.Drop == nil {
		return s, Out{}
	}

	if err := e.Drop(one.At); err != nil {
		return s, said(err.Error())
	}

	return s.Sync(e), said(e.Words.T("knowledge.left_said",
		"left in the thread where you said it"))
}
