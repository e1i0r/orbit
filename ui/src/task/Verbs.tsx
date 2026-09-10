// What a reader can do to a task, rather than read about it.
//
// Only the verbs that mean something where the task stands: a Start on a
// running task is a button that exists to be refused, and a Pause on one
// nobody is running is the same. Which are offered is decided from `held`
// and not from the band — a run parked at a gate sits in "needs you" and is
// very much alive, so the band would offer Start on a task something was
// already running and hide the Continue that was the useful thing to press.
//
// Every one of them asks first. Some spend money and some throw work away,
// and none is undone by pressing it again — so the confirmation says what is
// about to happen in the words it will happen in, and nothing fires on one
// click. Three of them take the reader's own words, and those keep the box
// open until there is something in it: a note nobody wrote is a note the
// next phase reads as nothing.

import { useState } from "react";
import { api, type Says, type Task, type Verb } from "../api";

interface Shape {
  name: string;
  asks: (t: Task) => string;
  tone: "go" | "quiet" | "bad" | "out";
  /** What the reader is asked to type, for the verbs that carry words.
   *  field is the name the verb takes it under, and "text" for most. */
  writes?: { placeholder: string; required: boolean; field?: string };
  /** An extra yes-or-no the verb takes, by the name the verb declares it
   *  under — internal/verb/every.go. A box posted under any other key is a
   *  box the verb never sees, and it reads as the answer nobody gave.
   *  on is what it starts as. */
  also?: { name: string; said: string; on?: boolean };
}

// The keys are the names internal/verb declares, because they are what is
// posted: the page asks for a verb by name and the server looks it up in the
// one vocabulary. A key that is not a declared verb is a button that comes
// back with "that is not something Orbit can be asked for" — which is what
// `start` did after the verb was named `run`.
const verbs: Record<Verb, Shape> = {
  run: {
    name: "Start",
    asks: (t) =>
      `Run ${t.id} through the ${t.flow || "default"} flow? This starts an engine and spends money.`,
    tone: "go",
  },
  continue: {
    name: "Continue",
    asks: (t) =>
      `Let ${t.id} past the gate its flow stopped it at? The next phase runs, and that spends money.`,
    tone: "go",
  },
  skip: {
    name: "Skip",
    asks: (t) =>
      `Let ${t.id} past the phase it is in, without running it? Nothing is recorded for a phase that did not run.`,
    tone: "quiet",
  },
  pause: {
    name: "Pause",
    asks: (t) => `Ask the run of ${t.id} to stop at its next phase? It finishes the one it is in.`,
    tone: "quiet",
  },
  resume: {
    name: "Resume",
    asks: (t) => `Let ${t.id} carry on from the pause you asked for?`,
    tone: "quiet",
  },
  cancel: {
    name: "Cancel",
    asks: (t) =>
      `Stop the run of ${t.id} now? What the phase has written stays; what it was doing is lost.`,
    tone: "bad",
  },
  note: {
    name: "Note",
    asks: (t) => `Leave a word on ${t.id}. The phase that starts next reads it.`,
    tone: "quiet",
    writes: { placeholder: "Use cents, not floats.", required: true },
  },
  direct: {
    name: "Direct",
    asks: (t) =>
      `Correct ${t.id}. It goes on the record and the run in flight is stopped, so the next one starts having read it.`,
    tone: "go",
    writes: { placeholder: "The endpoint should reject negative amounts.", required: true },
    also: { name: "restart", said: "and start the next run now — this spends money" },
  },
  requeue: {
    name: "Requeue",
    asks: (t) =>
      `Take ${t.id} back to the queue? Whatever is holding it is stopped first. Say why, if you want it on the record.`,
    tone: "bad",
    writes: { placeholder: "The brief was wrong.", required: false },
  },
  pr: {
    name: "Open a pull request",
    asks: (t) =>
      `Push ${t.id}'s branch and open a pull request on GitHub? This leaves your machine — other people will see it.`,
    tone: "out",
  },
  merge: {
    name: "Merge",
    asks: (t) =>
      `Merge ${t.id}'s pull request and delete its branch? The change goes into the branch everyone else works from, and this cannot be undone from here.`,
    tone: "out",
  },
  "close-pr": {
    name: "Close the pull request",
    asks: (t) => `Close ${t.id}'s pull request without merging it? The work stays; the request goes.`,
    tone: "bad",
  },
  approve: {
    name: "Approve",
    asks: (t) =>
      `Accept ${(t.pending ?? []).join(", ")} for ${t.id}? The next run goes past the dependency gate.`,
    tone: "go",
  },
  permit: {
    name: "Permit",
    asks: (t) =>
      `Let ${t.id} do the irreversible thing it stopped in front of? It was marked critical so that a person would answer this.`,
    tone: "out",
    // Checked is yes, and it starts checked: the button says Permit, and a
    // box that had to be found and ticked before Permit permitted anything
    // is a button that does the opposite of what it says.
    also: { name: "yes", said: "yes — let it happen", on: true },
  },
  critical: {
    name: "Mark critical",
    asks: (t) =>
      `Mark ${t.id} as one that reaches something that matters? It stops and asks before anything that cannot be taken back.`,
    tone: "quiet",
    also: { name: "on", said: "critical — stop and ask before anything irreversible", on: true },
  },
  read: {
    name: "Mark read",
    asks: (t) =>
      `Mark ${t.id} as looked at? The board stops counting it against the unread cap that holds new runs back.`,
    tone: "quiet",
  },
  join: {
    name: "Join a repository",
    asks: (t) =>
      `Open a checkout of another repository for ${t.id}, so its work can reach into both. Name it as \`orbit repos\` lists it.`,
    tone: "quiet",
    writes: { placeholder: "payments", required: true, field: "name" },
  },
  delete: {
    name: "Delete",
    asks: (t) =>
      `Remove ${t.id} and everything written about it — its record, its notes, its worktree? Nothing here brings it back.`,
    tone: "bad",
  },
};

const tones = {
  go: "border-accent/40 bg-accent/10 text-accent hover:bg-accent/20",
  quiet: "border-edge bg-well text-aside hover:text-said hover:bg-hover",
  bad: "border-bad/40 bg-bad/10 text-bad hover:bg-bad/20",
  // The three that leave this machine get their own colour, because the
  // difference between them and the rest is not how careful to be — it is
  // that everything else can be undone by asking again, and these cannot.
  out: "border-ok/40 bg-ok/10 text-ok hover:bg-ok/20",
};

// offered is which verbs mean something for a task as it stands.
//
// Note, Direct and Requeue are offered whatever the task is doing, because
// all three are recorded and wait: a note written on a task nobody is
// running is read by the phase that starts next, which is the point of it.
export function offered(task: Task): Verb[] {
  const out: Verb[] = task.held
    ? ["continue", "skip", "pause", "resume", "cancel"]
    : ["run"];

  if ((task.pending ?? []).length > 0) out.push("approve");

  // Delivering is offered whatever the task is doing. Which of the three
  // makes sense — there is no pull request yet, there is one already — is
  // the command's own question, and it answers in its own words; a second
  // opinion here would be a rule in two places that would drift.
  // Delivering is offered whatever the task is doing. Which of the three
  // makes sense — there is no pull request yet, there is one already — is
  // the command's own question, and it answers in its own words; a second
  // opinion here would be a rule in two places that would drift.
  //
  // Critical, Join and Delete are offered whatever it is doing too, and for
  // the same reason the recorded three are: none of them is about the run.
  //
  // Permit, Read, Critical, Join and Delete are offered on the same terms
  // and for the same reason. Whether there is anything waiting to be
  // permitted, or anything to mark read, is the verb's own question and it
  // answers in its own words — a second rule here would be the one that
  // drifts.
  return [...out, "direct", "note", "requeue", "pr", "merge", "close-pr",
    "permit", "read", "critical", "join", "delete"];
}

export function Verbs({ task, again }: { task: Task; again: () => void }) {
  const [asking, setAsking] = useState<Verb>();
  const [busy, setBusy] = useState<Verb>();
  const [said, setSaid] = useState<{ text: string; bad?: boolean }>();
  const [wrote, setWrote] = useState("");
  const [also, setAlso] = useState(false);

  const ask = (verb: Verb) => {
    setAsking(verb);
    setWrote("");
    setAlso(verbs[verb].also?.on ?? false);
  };

  const press = async (verb: Verb) => {
    const shape = verbs[verb];

    // Every verb takes its words under the name it declares them by, and
    // most of them call it "text". A body keyed by what this page felt like
    // calling it is a body the verb reads as empty — and an empty note is a
    // note the next phase reads as nothing.
    const says: Says = {};

    if (shape.writes && wrote.trim() !== "") {
      says[shape.writes.field ?? "text"] = wrote.trim();
    }

    if (shape.also) {
      says[shape.also.name] = also;
    }

    setAsking(undefined);
    setBusy(verb);

    try {
      const did = await api.do(task.id, verb, says);
      setSaid({ text: did.said });
      // The record is what says whether the run acted on it, so the page
      // reads it again rather than deciding for itself what changed.
      again();
    } catch (e) {
      setSaid({ text: (e as Error).message, bad: true });
    } finally {
      setBusy(undefined);
    }
  };

  const shape = asking ? verbs[asking] : undefined;
  const short = shape?.writes?.required === true && wrote.trim() === "";

  return (
    <div className="flex flex-col items-end gap-1.5">
      <div className="flex flex-wrap items-center gap-1.5 md:justify-end">
        {offered(task).map((verb) => (
          <button
            key={verb}
            onClick={() => ask(verb)}
            disabled={busy !== undefined}
            className={`rounded border px-2 py-0.5 text-[11px] transition-colors disabled:opacity-50 ${tones[verbs[verb].tone]}`}
          >
            {busy === verb ? "…" : verbs[verb].name}
          </button>
        ))}
      </div>

      {asking && shape && (
        <div className="flex w-full flex-col gap-1.5 rounded-md border border-edge bg-panel px-3 py-2 md:w-[46ch]">
          <p className="text-[11px] text-said">{shape.asks(task)}</p>

          {shape.writes && (
            <textarea
              value={wrote}
              onChange={(e) => setWrote(e.target.value)}
              placeholder={shape.writes.placeholder}
              rows={3}
              autoFocus
              className="w-full resize-y rounded border border-edge bg-well px-2 py-1 text-[11px] text-said placeholder:text-faint"
            />
          )}

          {shape.also && (
            <label className="flex cursor-pointer items-center gap-1.5 text-[11px] text-aside select-none">
              <input
                type="checkbox"
                checked={also}
                onChange={(e) => setAlso(e.target.checked)}
                className="size-2.5 accent-accent"
              />
              {shape.also.said}
            </label>
          )}

          <div className="flex justify-end gap-1.5">
            <button
              onClick={() => setAsking(undefined)}
              className="rounded border border-edge px-2 py-0.5 text-[11px] text-aside hover:text-said"
            >
              No
            </button>
            <button
              onClick={() => void press(asking)}
              disabled={short}
              className={`rounded border px-2 py-0.5 text-[11px] disabled:opacity-40 ${tones[shape.tone]}`}
              title={short ? "Write something first" : undefined}
            >
              Yes, {shape.name.toLowerCase()}
            </button>
          </div>
        </div>
      )}

      {said && (
        <p
          onClick={() => setSaid(undefined)}
          className={`max-w-[60ch] cursor-pointer text-right text-[11px] ${said.bad ? "text-bad" : "text-ok"}`}
          title="Click to clear"
        >
          {said.text}
        </p>
      )}
    </div>
  );
}
