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
  tone: "go" | "quiet" | "bad";
  /** What the reader is asked to type, for the verbs that carry words. */
  writes?: { placeholder: string; required: boolean };
  /** An extra yes-or-no the verb takes. */
  also?: { name: string; said: string };
}

const verbs: Record<Verb, Shape> = {
  start: {
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
  approve: {
    name: "Approve",
    asks: (t) =>
      `Accept ${(t.pending ?? []).join(", ")} for ${t.id}? The next run goes past the dependency gate.`,
    tone: "go",
  },
};

const tones = {
  go: "border-accent/40 bg-accent/10 text-accent hover:bg-accent/20",
  quiet: "border-edge bg-well text-aside hover:text-said hover:bg-hover",
  bad: "border-bad/40 bg-bad/10 text-bad hover:bg-bad/20",
};

// offered is which verbs mean something for a task as it stands.
//
// Note, Direct and Requeue are offered whatever the task is doing, because
// all three are recorded and wait: a note written on a task nobody is
// running is read by the phase that starts next, which is the point of it.
export function offered(task: Task): Verb[] {
  const out: Verb[] = task.held
    ? ["continue", "skip", "pause", "resume", "cancel"]
    : ["start"];

  if ((task.pending ?? []).length > 0) out.push("approve");

  return [...out, "direct", "note", "requeue"];
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
    setAlso(false);
  };

  const press = async (verb: Verb) => {
    const says: Says = { text: wrote.trim() || undefined, restart: also || undefined };

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
      <div className="flex flex-wrap items-center justify-end gap-1.5">
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
        <div className="flex w-[46ch] flex-col gap-1.5 rounded-md border border-edge bg-panel px-3 py-2">
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
