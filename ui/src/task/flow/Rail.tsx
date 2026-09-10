// A flow, drawn as the rail its phases hang off.
//
// It replaced a strip of chips above a list of the same names below, which
// said everything twice and put a phase's standing a thousand pixels from
// its name. One structure now: a row per phase, with what it runs on and
// what it has to satisfy on the row rather than behind a click, and the loop
// as an indented block off the same rail instead of a dashed box with its
// caption under the wrong thing.
//
// It is the shape the timeline already uses, because a flow is the same kind
// of thing: an ordered list of moments, some of them nested.

import { useState } from "react";
import { money } from "../../parts/money";
import { AlertTriangle, Check, Circle, Play, RotateCw, X, type LucideIcon } from "lucide-react";
import type { Gate, Phase, Standing } from "../../api";

interface Mark {
  said: string;
  icon: LucideIcon;
  node: string;
  text: string;
}

const marks: Record<Standing, Mark> = {
  done: { said: "done", icon: Check, node: "bg-ok/15 text-ok", text: "text-ok" },
  running: { said: "running", icon: Play, node: "bg-live/15 text-live", text: "text-live" },
  looping: { said: "going round", icon: RotateCw, node: "bg-live/15 text-live", text: "text-live" },
  waiting: { said: "waiting", icon: AlertTriangle, node: "bg-wait/15 text-wait", text: "text-wait" },
  failed: { said: "failed", icon: X, node: "bg-bad/15 text-bad", text: "text-bad" },
  cancelled: { said: "cancelled", icon: X, node: "bg-wait/15 text-wait", text: "text-wait" },
  pending: { said: "pending", icon: Circle, node: "bg-edge text-faint", text: "text-faint" },
};

// waiting is the mark a phase gets where there is no run behind it — the
// catalogue of flows, where a phase has not happened to anything and saying
// "pending" would be a claim about a run that has not begun.
const unrun: Mark = { said: "", icon: Circle, node: "bg-edge text-faint", text: "text-faint" };

function markOf(standing?: Standing): Mark {
  return standing ? (marks[standing] ?? marks.pending) : unrun;
}

export function Rail({ phases }: { phases: Phase[] }) {
  return (
    <ol className="flex flex-col">
      {phases.map((phase, i) => (
        <Node key={phase.name} phase={phase} last={i === phases.length - 1} />
      ))}
    </ol>
  );
}

// Node is one phase on the rail: what it is, what it runs on, where it got
// to — and, opened, everything else the flow says about it.
function Node({ phase, last, inner }: { phase: Phase; last: boolean; inner?: boolean }) {
  const [open, setOpen] = useState(false);
  const at = markOf(phase.standing);
  const Icon = at.icon;
  const loop = phase.loop;
  const dials = [phase.engine, phase.model, phase.effort, phase.thinking].filter(Boolean);
  const gates = phase.gates ?? [];
  const may = phase.permissions ?? [];
  // What is behind the click is only what a run leaves behind. Everything a
  // flow says about a phase is on the row: opening one of those gave a box
  // the width of the page around the single word "repo".
  const more = Boolean(phase.cause || phase.said);

  return (
    <li className="relative flex gap-2.5">
      {/* The rail carries on under a node whether it is open or shut, which
          is what keeps a folded flow a flow. */}
      {!last && <span className="absolute top-6 bottom-0 left-[9px] w-px bg-edge" aria-hidden />}

      <span
        className={`z-10 mt-1 grid size-[19px] shrink-0 place-items-center rounded-full ${at.node}`}
      >
        <Icon size={11} strokeWidth={2.5} aria-hidden />
      </span>

      <div className={`min-w-0 flex-1 ${last ? "pb-0" : "pb-3"}`}>
        <button
          onClick={() => more && setOpen(!open)}
          aria-expanded={more ? open : undefined}
          disabled={!more}
          className="flex w-full items-baseline gap-2 text-left disabled:cursor-default"
        >
          <span className={`font-mono ${inner ? "text-[11px]" : "text-xs"} text-said`}>
            {phase.name}
          </span>

          {dials.length > 0 && (
            <span className="truncate text-[10px] text-faint">{dials.join(" · ")}</span>
          )}

          {may.length > 0 && (
            <span className="shrink-0 text-[10px] text-faint">may touch {may.join(", ")}</span>
          )}

          {phase.waits && <span className="shrink-0 text-[10px] text-wait">stops for a person</span>}

          <span className="ml-auto flex shrink-0 items-baseline gap-2.5">
            {phase.cost ? (
              <span className="font-mono text-[10px] text-aside tabular-nums">
                {money(phase.cost)}
              </span>
            ) : null}
            {took(phase) && (
              <span className="font-mono text-[10px] text-faint tabular-nums">{took(phase)}</span>
            )}
            {at.said && <span className={`text-[10px] ${at.text}`}>{at.said}</span>}
            {more && (
              <span className={`text-faint transition-transform ${open ? "rotate-90" : ""}`}>›</span>
            )}
          </span>
        </button>

        {/* A loop says what makes it stop on the line under its name, and
            holds its phases inside a rail of their own. */}
        {loop && (
          <div className="mt-1.5 border-l border-dashed border-edge pl-3">
            <p className="mb-1.5 text-[10px] text-faint">
              goes round{" "}
              {phase.standing ? `${loop.turns} of ${loop.max} times` : `up to ${loop.max} times`},
              until{" "}
              <span className="text-aside">
                {(loop.until ?? []).map((g) => g.name).join(" and ") || "something says so"}
              </span>
            </p>

            <ol className="flex flex-col">
              {(loop.phases ?? []).map((one, i) => (
                <Node
                  key={one.name}
                  phase={one}
                  last={i === (loop.phases ?? []).length - 1}
                  inner
                />
              ))}
            </ol>

            <Gates of={loop.until ?? []} />
          </div>
        )}

        {!loop && <Gates of={gates} />}

        {open && <Opened phase={phase} />}
      </div>
    </li>
  );
}

// Gates is what a phase has to satisfy, on the rail rather than behind a
// click: a command with an exit code is the most concrete thing a flow says
// about a phase, and hiding it left every row saying only a name.
function Gates({ of }: { of: Gate[] }) {
  if (of.length === 0) return null;

  return (
    <ul className="mt-1 flex flex-col gap-0.5">
      {of.map((gate) => (
        <li key={gate.name} className="flex min-w-0 gap-2 font-mono text-[10px]">
          <span className="shrink-0 text-faint">{gate.name}</span>
          <span className="truncate text-aside" title={gate.command}>
            {gate.command}
          </span>
        </li>
      ))}
    </ul>
  );
}

// Opened is what a run left behind: why the phase stopped, and what it
// wrote. Only those two, and only when there are any — everything a flow
// says about a phase is on the row above, and a panel the width of the page
// around one word is worse than no panel.
function Opened({ phase }: { phase: Phase }) {
  return (
    <div className="mt-1.5 flex flex-col gap-1.5 rounded border border-edge bg-well px-2.5 py-2">
      {phase.cause && (
        <p className="text-[11px] text-bad">
          <span className="text-faint">stopped because </span>
          {phase.cause}
        </p>
      )}

      {phase.said && (
        // Boxed and scrolled rather than let run: a phase's answer is often
        // a page of markdown, and a row that grows to hold it buries the
        // phases under it.
        <div className="max-h-44 overflow-y-auto">
          <p className="text-[11px] whitespace-pre-wrap text-aside">{phase.said}</p>
        </div>
      )}
    </div>
  );
}

// took is how long a phase ran, in the largest unit that is still exact
// enough to act on.
function took(phase: Phase): string {
  if (!phase.started || !phase.ended) return "";

  const ms = new Date(phase.ended).getTime() - new Date(phase.started).getTime();
  if (Number.isNaN(ms) || ms < 0) return "";

  const secs = Math.round(ms / 1000);
  if (secs < 60) return `${secs}s`;

  const mins = Math.floor(secs / 60);
  if (mins < 60) return `${mins}m ${secs % 60}s`;

  return `${Math.floor(mins / 60)}h ${mins % 60}m`;
}
