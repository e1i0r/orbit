// The task at a glance: what it is, where it got to, and what it cost.
//
// It is the tab that opens, so it answers the question somebody opens a task
// with — is this done, what is it waiting for, and did it do what I asked.
//
// One block of facts and no cards. It used to be three: a grid, a box around
// the single word "orbit", and a third repeating the task's own title, which
// is already the sentence under the heading. A box around one value is worse
// than the value on a line.

import type { Task } from "../api";
import { Pill } from "../parts/Pill";
import { meaning } from "./kinds";

const bands: Record<string, { said: string; tone: "wait" | "live" | "quiet" | "ok" }> = {
  needs_you: { said: "Needs you", tone: "wait" },
  running: { said: "Running", tone: "live" },
  todo: { said: "To do", tone: "quiet" },
  done: { said: "Done", tone: "ok" },
};

export function Overview({ task }: { task?: Task }) {
  if (!task) return <p className="text-xs text-aside">Reading the record…</p>;

  const band = bands[task.band] ?? { said: task.band, tone: "quiet" as const };
  const entries = task.entries ?? [];
  const last = entries[entries.length - 1];
  const phases = entries.filter((e) => e.kind === "phase.finished").length;
  const repos = (task.repos ?? [task.repo]).filter(Boolean);
  const story = entries.filter((e) => e.story).pop()?.story;
  const started = entries.find((e) => e.kind === "task.started")?.at;

  return (
    <div className="flex max-w-[900px] flex-col gap-4">
      <dl className="grid gap-x-6 gap-y-3.5 sm:grid-cols-4">
        <Fact name="State">
          <Pill tone={band.tone}>{band.said}</Pill>
        </Fact>

        <Fact name="Flow">
          <span className="font-mono text-xs text-said">{task.flow || "—"}</span>
          {task.phase && <span className="font-mono text-xs text-faint"> · {task.phase}</span>}
        </Fact>

        <Fact name="Engine">
          <span className="text-xs text-said">{task.engine || "—"}</span>
          {task.model && <span className="text-xs text-faint"> · {task.model}</span>}
        </Fact>

        <Fact name="Repositories">
          <span className="font-mono text-xs text-said">
            {repos.length > 0 ? repos.join(", ") : "none joined yet"}
          </span>
        </Fact>

        <Fact name="Phases finished">
          <span className="font-mono text-xs text-said tabular-nums">{phases}</span>
        </Fact>

        <Fact name="Spent">
          <span className="font-mono text-xs text-said tabular-nums">
            {task.spent ? `$${task.spent.toFixed(4)}` : "—"}
          </span>
        </Fact>

        <Fact name="First run">
          <span className="text-xs text-said">{started ? when(started) : "never started"}</span>
        </Fact>

        <Fact name="Last thing that happened">
          <span className="text-xs text-said">{last ? meaning(last.kind).said : "—"}</span>
          {last && <span className="text-xs text-faint"> · {when(last.at)}</span>}
        </Fact>
      </dl>

      {(task.pending ?? []).length > 0 && (
        <p className="rounded-md border border-wait/30 bg-wait/5 px-3 py-2 text-xs text-wait">
          Waiting on you to accept{" "}
          <span className="font-mono">{(task.pending ?? []).join(", ")}</span>. Approve it and the
          next run goes past the dependency gate.
        </p>
      )}

      {story?.fix && (
        <div>
          <h2 className="text-[10px] tracking-[0.09em] text-faint uppercase">What was done</h2>
          <p className="mt-1 text-xs text-said">{story.fix}</p>
          {story.cause && (
            <p className="mt-1 text-xs text-aside">
              <span className="text-faint">because </span>
              {story.cause}
            </p>
          )}
        </div>
      )}
    </div>
  );
}

function Fact({ name, children }: { name: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="text-[10px] tracking-[0.09em] text-faint uppercase">{name}</dt>
      <dd className="mt-1 truncate">{children}</dd>
    </div>
  );
}

function when(at: string): string {
  const t = new Date(at);

  return Number.isNaN(t.getTime())
    ? "—"
    : t.toLocaleString([], { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}
