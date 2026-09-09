// The flow the task was started under, and how far the run got through it.
//
// One rail, and everything a reader needs on the row: the phase, what it ran
// on, what it had to satisfy, what it cost, how long it took and where it
// got to. What is behind a click is only the reason for the row — what the
// phase may touch, why it stopped, and what it wrote.
//
// The cockpit draws this as a tree of box characters because a tree is what
// a terminal can draw. Here a loop can be an indented block off the same
// rail rather than a node that says "going round" and hides its phases.

import type { Flow } from "../api";
import { Empty } from "../parts/Empty";
import { Rail } from "./flow/Rail";

export function FlowView({ flow }: { flow?: Flow }) {
  if (!flow) return <p className="text-xs text-aside">Reading the flow…</p>;

  if (flow.failed) {
    return <Empty said={`The flow "${flow.name}" could not be read`} next={flow.failed} />;
  }

  const phases = flow.phases ?? [];

  if (phases.length === 0) {
    return <Empty said="This flow has no phases" />;
  }

  return (
    <div className="flex max-w-[900px] flex-col gap-3">
      <div>
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <h2 className="font-mono text-xs text-said">{flow.name}</h2>
          <span className="text-[10px] text-faint">
            {phases.length} {phases.length === 1 ? "phase" : "phases"} · up to {flow.attempts}{" "}
            {flow.attempts === 1 ? "attempt" : "attempts"} each
            {flow.diffBudget ? ` · ${flow.diffBudget}-line budget` : ""}
          </span>
        </div>
        {flow.description && <p className="mt-1 text-[11px] text-aside">{flow.description}</p>}
      </div>

      <div className="rounded-md border border-edge bg-panel px-3 py-3">
        <Rail phases={phases} />
      </div>
    </div>
  );
}
