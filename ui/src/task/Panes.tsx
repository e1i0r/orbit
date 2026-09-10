// The panes that read the record and nothing else.
//
// Six of the cockpit's twelve are a reading of the same list of events: the
// gates it stopped at, what each phase cost, what the sandbox refused, what
// a person said, what the engine was thinking, and the story it told at the
// end. They are together in one file because they are one idea — filter the
// record, and say what is left in the shape that reading wants.

import { useState } from "react";
import { money } from "../parts/money";
import type { Entry, Step, Task } from "../api";
import { Empty } from "../parts/Empty";
import { Pill } from "../parts/Pill";
import { meaning } from "./kinds";

const of = (entries: Entry[] | null, ...kinds: string[]) =>
  (entries ?? []).filter((e) => kinds.includes(e.kind));

// Gates is every check the run was put to, and what it answered.
//
// A check is a command and an exit code, so the row leads with which command
// and what it said. The output goes in a block that keeps its own line
// breaks: it is `go test`'s output, and run together into a paragraph it is
// forty package names in a row that nobody can read.
//
// A passing check's output is folded and a failing one's is not. That is the
// whole of the difference between them: nobody reads the output of a check
// that passed, and the output of one that failed is the reason the run
// stopped.
export function Gates({ task }: { task?: Task }) {
  const gates = of(
    task?.entries ?? null,
    "gate.passed",
    "gate.failed",
    "loop.checked",
    "phase.waiting",
  );

  if (gates.length === 0) {
    return (
      <Empty
        said="No gate has been asked yet"
        next="A flow's gates run after a phase, and a loop's checks run every turn."
      />
    );
  }

  const failed = gates.filter((e) => e.kind === "gate.failed").length;

  return (
    <div className="flex max-w-[900px] flex-col gap-2.5">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-said">{gates.length}</span>{" "}
        {gates.length === 1 ? "check" : "checks"} ran
        {failed > 0 ? (
          <>
            {" · "}
            <span className="text-bad">{failed} refused the work</span>
          </>
        ) : (
          <span className="text-ok"> · all of them passed</span>
        )}
      </p>

      <ol className="flex flex-col gap-1.5">
        {gates.map((e, i) => (
          <Gate key={i} gate={e} />
        ))}
      </ol>
    </div>
  );
}

// Gate is one check: what it was, what it answered, and what it printed.
function Gate({ gate }: { gate: Entry }) {
  const bad = gate.kind === "gate.failed";
  const [open, setOpen] = useState(bad);
  const text = (gate.text ?? "").trimEnd();

  return (
    <li className={`rounded-md border ${bad ? "border-bad/30" : "border-edge"} bg-panel`}>
      <div className="flex items-baseline gap-2 px-3 py-1.5">
        <Pill tone={bad ? "bad" : gate.kind === "phase.waiting" ? "wait" : "ok"}>
          {meaning(gate.kind).said}
        </Pill>

        <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-said">
          {gate.gate || gate.phase || "—"}
        </span>

        {gate.phase && gate.gate && (
          <span className="shrink-0 font-mono text-[10px] text-faint">{gate.phase}</span>
        )}

        {gate.exit && (
          <span className={`shrink-0 font-mono text-[10px] ${bad ? "text-bad" : "text-faint"}`}>
            exit {gate.exit}
          </span>
        )}

        {text && (
          <button
            onClick={() => setOpen(!open)}
            aria-expanded={open}
            className="shrink-0 text-[10px] text-faint transition-colors hover:text-said"
          >
            {open ? "hide what it printed" : said(lines(text))}
          </button>
        )}
      </div>

      {open && text && (
        <pre className="max-h-72 overflow-auto border-t border-edge bg-well px-3 py-2 font-mono text-[11px] leading-[1.5] text-aside">
          {text}
        </pre>
      )}
    </li>
  );
}

// lines is how much a check printed, so the button says what opening it
// costs the reader.
function lines(text: string): number {
  return text.split("\n").length;
}

function said(n: number): string {
  return `${n} ${n === 1 ? "line" : "lines"}`;
}

// Cost is what the task spent, phase by phase.
//
// The share matters as much as the amount: a reader asking "where did three
// dollars go" is asking which phase took most of it, and a column of four
// figures makes them do that arithmetic themselves.
export function Cost({ task }: { task?: Task }) {
  const priced = of(
    task?.entries ?? null,
    "phase.finished",
    "phase.failed",
    "phase.retried",
  ).filter((e) => e.cost);

  if (priced.length === 0) {
    return (
      <Empty
        said="Nothing has been spent yet"
        next="A phase reports what it cost when it ends, and the engine has to say so."
      />
    );
  }

  const spent = task?.spent ?? 0;
  const most = Math.max(...priced.map((e) => e.cost ?? 0));

  return (
    <div className="flex max-w-[900px] flex-col gap-3">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-sm text-said tabular-nums">{money(spent)}</span> over{" "}
        {priced.length} {priced.length === 1 ? "phase" : "phases"}
      </p>

      <ol className="flex flex-col gap-2">
        {priced.map((e, i) => (
          <li key={i} className="grid grid-cols-[1fr_auto] items-baseline gap-x-3 gap-y-1">
            <span className="truncate font-mono text-[11px] text-said">{e.phase}</span>

            <span className="flex shrink-0 items-baseline gap-2.5">
              <span className="font-mono text-[10px] text-faint tabular-nums">
                {spent > 0 ? `${Math.round(((e.cost ?? 0) / spent) * 100)}%` : ""}
              </span>
              <span className="w-16 text-right font-mono text-[11px] text-aside tabular-nums">
                {money(e.cost ?? 0)}
              </span>
            </span>

            <span className="col-span-2 h-[3px] overflow-hidden rounded-full bg-well">
              <span
                className="block h-full rounded-full bg-accent/70"
                style={{ width: `${Math.max(2, ((e.cost ?? 0) / most) * 100)}%` }}
              />
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

// Refused is what the sandbox said no to, which is the one list that says
// the posture is too narrow rather than the work being wrong.
//
// The tool leads the row and what it was asked to do follows it, kept on its
// own lines: a refusal is a command somebody wrote, and run together into a
// paragraph it stops being one.
export function Refused({ task }: { task?: Task }) {
  const refusals = of(task?.entries ?? null, "phase.refused");

  if (refusals.length === 0) {
    return (
      <Empty
        said="Nothing was refused"
        next="A tool the sandbox turned down lands here, with what it was asked to do."
      />
    );
  }

  return (
    <div className="flex max-w-[900px] flex-col gap-2.5">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-bad">{refusals.length}</span>{" "}
        {refusals.length === 1 ? "call was" : "calls were"} turned down. Each is the sandbox saying
        no to the posture the phase was given, not to the work.
      </p>

      <ol className="flex flex-col gap-1.5">
        {refusals.map((e, i) => (
          <li key={i} className="rounded-md border border-bad/30 bg-bad/5">
            <div className="flex items-baseline gap-2 px-3 py-1.5">
              <span className="font-mono text-[11px] text-bad">{e.tool || "a tool"}</span>
              {e.phase && (
                <span className="ml-auto shrink-0 font-mono text-[10px] text-faint">{e.phase}</span>
              )}
              <time className="shrink-0 text-[10px] text-faint tabular-nums">{when(e.at)}</time>
            </div>

            {e.text && (
              <pre className="max-h-44 overflow-auto border-t border-bad/20 px-3 py-1.5 font-mono text-[11px] leading-[1.5] text-aside">
                {e.text}
              </pre>
            )}
          </li>
        ))}
      </ol>
    </div>
  );
}

// Notes is what a person said, and when — the brief before the run, and
// every word since.
//
// Oldest first, because this is a conversation and a conversation is read
// forwards. The timeline is the tab that opens newest-first; here the first
// thing said is the brief the whole task came from.
export function Notes({ task }: { task?: Task }) {
  const said = of(
    task?.entries ?? null,
    "task.noted",
    "task.dialogue",
    "supervisor.message",
  );

  if (said.length === 0) {
    return (
      <Empty
        said="Nobody has said anything yet"
        next="orbit note leaves a word for the next phase, and it lands here."
      />
    );
  }

  return (
    <div className="flex max-w-[900px] flex-col gap-2.5">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-said">{said.length}</span>{" "}
        {said.length === 1 ? "thing was" : "things were"} said to this task. Each is read by the
        phase that starts after it.
      </p>

      <ol className="flex flex-col gap-2.5">
        {said.map((e, i) => (
          <li key={i} className="border-l-2 border-edge pl-3">
            <p className="flex flex-wrap items-baseline gap-x-2 text-[10px] text-faint">
              <span className="text-aside">{meaning(e.kind).said}</span>
              <span className="tabular-nums">{when(e.at)}</span>
              {e.phase && <span>· {e.phase}</span>}
            </p>
            <p className="mt-0.5 text-xs whitespace-pre-wrap text-said">{e.text}</p>
          </li>
        ))}
      </ol>
    </div>
  );
}

// Thinking is what the engine showed of its own work: what it weighed, and
// what it turned down.
//
// It is the one pane whose whole content is the model talking to itself, so
// it keeps its own line breaks and its own paragraphs. Cut into a clamp it
// would be the first sentence of a reasoning nobody can follow.
export function Thinking({ task }: { task?: Task }) {
  const thoughts = of(task?.entries ?? null, "phase.thought");

  if (thoughts.length === 0) {
    return (
      <Empty
        said="No thinking was recorded"
        next="An engine asked to show its work writes it here as it goes. Not every engine does, and a phase that was not asked writes none."
      />
    );
  }

  return (
    <div className="flex max-w-[900px] flex-col gap-2.5">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-said">{thoughts.length}</span>{" "}
        {thoughts.length === 1 ? "block" : "blocks"} of the engine's own reasoning. Nothing here was
        verified — it is what the model said it was weighing.
      </p>

      <ol className="flex flex-col gap-2">
        {thoughts.map((e, i) => (
          <li key={i} className="rounded-md border border-edge bg-panel px-3 py-2">
            <p className="flex flex-wrap items-baseline gap-x-2 text-[10px] text-faint">
              {e.phase && <span className="font-mono text-aside">{e.phase}</span>}
              <span className="tabular-nums">{when(e.at)}</span>
              {e.truncated && <span className="text-wait">the record kept less than it printed</span>}
            </p>
            <p className="mt-1 text-xs whitespace-pre-wrap text-aside">{e.text}</p>
          </li>
        ))}
      </ol>
    </div>
  );
}

// Report is the story a task told about itself: how the change came about,
// in the parts the record keeps it in.
//
// The five parts are a chain — the route in, what it is for, what went
// wrong, why, and what was done — so they are set as one table rather than
// five boxes with a heading each. What the change asks and promises follows
// it as four lists in a single column: two columns left a ragged hole,
// because "decided against" is four sentences and "needs" is one.
export function Report({ task }: { task?: Task }) {
  const told = (task?.entries ?? []).filter((e) => e.story).pop();
  const delta = (task?.entries ?? []).filter((e) => e.delta).pop();

  if (!told && !delta) {
    return (
      <Empty
        said="No story yet"
        next="The last phase of a flow is asked how the change came about, and what it writes lands here."
      />
    );
  }

  const parts: [string, string | undefined][] = [
    ["What it is", told?.story?.entry],
    ["What it is for", told?.story?.purpose],
    ["What went wrong", told?.story?.symptom],
    ["Why", told?.story?.cause],
    ["What was done", told?.story?.fix],
  ];

  return (
    <div className="flex max-w-[900px] flex-col gap-5">
      {told && (
        <dl className="flex flex-col gap-2.5">
          {parts
            .filter(([, what]) => what)
            .map(([name, what]) => (
              <div key={name} className="grid gap-1 sm:grid-cols-[140px_1fr] sm:gap-4">
                <dt className="text-[11px] text-faint">{name}</dt>
                <dd className="text-xs whitespace-pre-wrap text-said">{what}</dd>
              </div>
            ))}
        </dl>
      )}

      {delta?.delta && (
        <div className="flex flex-col gap-3 border-t border-edge pt-4">
          <p className="text-[11px] text-faint">
            What the engine says its own change asks and promises. Nobody verified it — no command
            can — and the last part is the only place a rejected approach is written down.
          </p>

          <Listed name="Callers must now" said={delta.delta.needs} />
          <Listed name="It now holds" said={delta.delta.guarantees} />
          <Listed name="It took for granted" said={delta.delta.assumes} />
          <Listed name="Considered and not taken" said={delta.delta.instead} />
        </div>
      )}
    </div>
  );
}

// Artifacts is the path the agent walked: every file it changed, and how
// often it looked at one on the way.
//
// In the order it first reached them, and the pane says so: the order is the
// reading. A list sorted by how much changed would answer a different
// question, and hide the one this answers — where it started and what it
// went to next.
export function Artifacts({ task }: { task?: Task }) {
  const walk: Step[] = task?.walk ?? [];

  if (walk.length === 0) {
    return (
      <Empty
        said="No files touched yet"
        next="Every file a phase writes to is recorded as it happens, in the order it was first reached."
      />
    );
  }

  const changed = walk.reduce((n, one) => n + one.touches, 0);
  const read = walk.reduce((n, one) => n + one.read, 0);
  const most = Math.max(...walk.map((one) => one.touches + one.read), 1);

  return (
    <div className="flex max-w-[900px] flex-col gap-3">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-said">{walk.length}</span>{" "}
        {walk.length === 1 ? "file" : "files"}, in the order the agent first reached them ·{" "}
        <span className="font-mono text-ok">{changed}</span>{" "}
        {changed === 1 ? "write" : "writes"} ·{" "}
        <span className="font-mono text-aside">{read}</span> {read === 1 ? "read" : "reads"}
      </p>

      <ol className="flex flex-col gap-1">
        {walk.map((step, i) => (
          <li key={step.path} className="flex items-center gap-2.5">
            <span className="w-5 shrink-0 text-right font-mono text-[10px] text-faint tabular-nums">
              {i + 1}
            </span>

            <span
              className="min-w-0 flex-1 truncate font-mono text-[11px] text-said"
              title={step.path}
            >
              {step.path}
            </span>

            <span className="flex h-[3px] w-24 shrink-0 overflow-hidden rounded-full bg-well">
              <span
                className="block h-full bg-ok"
                style={{ width: `${(step.touches / most) * 100}%` }}
              />
              <span
                className="block h-full bg-edge"
                style={{ width: `${(step.read / most) * 100}%` }}
              />
            </span>

            <span className="w-8 shrink-0 text-right font-mono text-[10px] text-ok tabular-nums">
              {step.touches || "—"}
            </span>
            <span className="w-8 shrink-0 text-right font-mono text-[10px] text-faint tabular-nums">
              {step.read || "—"}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

function Listed({ name, said }: { name: string; said?: string[] }) {
  if (!said || said.length === 0) return null;

  return (
    <div>
      <h3 className="text-[11px] text-faint">
        {name}
      </h3>
      <ul className="mt-1 flex flex-col gap-0.5">
        {said.map((one, i) => (
          <li key={i} className="text-xs text-aside">
            {one}
          </li>
        ))}
      </ul>
    </div>
  );
}

function when(at: string): string {
  const t = new Date(at);

  return Number.isNaN(t.getTime())
    ? "—"
    : t.toLocaleString([], {
        day: "numeric",
        month: "short",
        hour: "2-digit",
        minute: "2-digit",
      });
}
