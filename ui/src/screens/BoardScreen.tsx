// The board: every task, in the band that says what it is waiting for.
//
// The bands are the structure and not a filter, because the question Orbit
// answers is "what needs me" — a flat list of forty rows answers a different
// one. Each band carries its own colour down its left edge, so the answer is
// readable before a word is.

import type { Band, Board, TaskSummary } from "../api";
import { Empty } from "../parts/Empty";
import { Pill } from "../parts/Pill";

const bands: { id: Band; name: string; said: string; rail: string }[] = [
  { id: "needs_you", name: "Needs you", said: "stopped, and waiting on a person", rail: "border-wait" },
  { id: "running", name: "Running", said: "an engine is working on it now", rail: "border-live" },
  { id: "todo", name: "To do", said: "written down, not started", rail: "border-faint" },
  { id: "done", name: "Done", said: "finished, merged or given up on", rail: "border-ok" },
];

export function BoardScreen({ board, open }: { board?: Board; open: (id: string) => void }) {
  if (!board) {
    return <p className="text-xs text-aside">Reading the board…</p>;
  }

  const tasks = board.tasks ?? [];

  if (tasks.length === 0) {
    return (
      <Empty
        said="No tasks yet"
        next={`Nothing has been written down against the repositories under ${board.root}. Write one with orbit new, and it lands here.`}
      />
    );
  }

  return (
    <div className="flex flex-col gap-5">
      {bands.map((band) => {
        const inIt = tasks.filter((t) => t.band === band.id);
        if (inIt.length === 0) return null;

        return (
          <section key={band.id} className={`border-l-2 pl-4 ${band.rail}`}>
            <header className="mb-2 flex items-baseline gap-2.5">
              <h2 className="text-xs font-semibold">{band.name}</h2>
              <span className="font-mono text-[10px] text-faint">{inIt.length}</span>
              <span className="text-[11px] text-faint">{band.said}</span>
            </header>

            <div className="overflow-hidden rounded-md border border-edge">
              <table className="w-full border-collapse text-xs">
                <thead>
                  <tr className="bg-panel text-[10px] tracking-[0.09em] text-faint uppercase">
                    <Th className="w-28">Task</Th>
                    <Th>What it is</Th>
                    <Th className="w-40">Repository</Th>
                    <Th className="w-52">Flow</Th>
                    <Th className="w-28">Engine</Th>
                  </tr>
                </thead>
                <tbody>
                  {inIt.map((t) => (
                    <Row key={t.id} task={t} open={open} />
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        );
      })}
    </div>
  );
}

function Th({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return <th className={`px-3 py-1.5 text-left font-semibold ${className}`}>{children}</th>;
}

function Row({ task, open }: { task: TaskSummary; open: (id: string) => void }) {
  return (
    <tr
      onClick={() => open(task.id)}
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && open(task.id)}
      className="cursor-pointer border-t border-edge/60 bg-page transition-colors hover:bg-panel"
    >
      <td className="px-3 py-1.5 font-mono text-[11px] text-accent">{task.id}</td>
      <td className="max-w-0 truncate px-3 py-1.5" title={task.title}>
        {task.title || <span className="text-faint">no description</span>}
      </td>
      <td className="px-3 py-1.5 text-aside">{task.repo || "—"}</td>
      <td className="px-3 py-1.5">
        <span className="block truncate font-mono text-[11px] text-aside">
          {task.flow || "—"}
          {task.phase && <span className="text-faint"> · {task.phase}</span>}
        </span>
      </td>
      <td className="px-3 py-1.5">
        {task.engine ? <Pill tone="quiet">{task.engine}</Pill> : <span className="text-faint">—</span>}
      </td>
    </tr>
  );
}
