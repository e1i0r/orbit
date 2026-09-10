// What happened, in the order it happened.
//
// A rail with a node per event, the way frauddi draws a case's history: what
// happened on the line, who or what did it under that, when on the right. A
// terminal draws this as forty rows of dotted lowercase because forty rows is
// all it has; here there is room to say it.

import type { Entry } from "../api";
import { money } from "../parts/money";
import { plain } from "../parts/plain";
import { Empty } from "../parts/Empty";
import { meaning, type Tone } from "./kinds";

const nodes: Record<Tone, string> = {
  ok: "bg-ok/15 text-ok",
  bad: "bg-bad/15 text-bad",
  wait: "bg-wait/15 text-wait",
  live: "bg-live/15 text-live",
  accent: "bg-accent/15 text-accent",
  quiet: "bg-edge text-faint",
};

export function Timeline({ entries }: { entries: Entry[] | null }) {
  if (!entries) return <p className="text-xs text-aside">Reading the record…</p>;

  if (entries.length === 0) {
    return (
      <Empty
        said="Nothing recorded yet"
        next="Every phase, gate and decision writes a line here as it happens."
      />
    );
  }

  // Newest first: the question a person opens a running task with is what it
  // is doing now, not how it began.
  const newest = [...entries].reverse();

  return (
    // A measure and not a pixel count: the time of an entry sits at the
    // right edge of this box, and at a thousand pixels that put it a hand's
    // width away from the line it belongs to.
    <div className="measure">
      <p className="border-b border-dashed border-edge pb-3 text-[11px] text-faint">
        Everything the record holds about this task — phases, gates, tool calls and what a person
        said. Newest first.
      </p>

      <ol className="mt-4">
        {newest.map((e, i) => {
          const what = meaning(e.kind);
          const Icon = what.icon;
          const last = i === newest.length - 1;

          return (
            <li key={i} className="relative flex gap-3 pb-4">
              {!last && <span className="absolute top-7 bottom-0 left-[12px] w-px bg-edge" />}

              <span
                className={`z-10 grid size-[25px] shrink-0 place-items-center rounded-full ${nodes[what.tone]}`}
              >
                <Icon size={13} strokeWidth={2} aria-hidden />
              </span>

              <div className="min-w-0 flex-1">
                <div className="flex items-baseline justify-between gap-3">
                  <h3 className="text-xs font-medium">
                    {what.said}
                    {e.phase && <span className="text-aside"> · {e.phase}</span>}
                    {e.gate && <span className="text-aside"> · {e.gate}</span>}
                  </h3>
                  <time className="shrink-0 text-[10px] text-faint tabular-nums">{when(e.at)}</time>
                </div>

                {by(e) && <p className="text-[10px] text-faint italic">{by(e)}</p>}

                {e.text && (
                  // pre-wrap under the clamp, so two lines of an engine's
                  // answer are its first two lines rather than forty of them
                  // run together into a paragraph nobody can read. And its
                  // words rather than its markdown: "## Decisions" is the
                  // heading of what was said, not any of it.
                  <p
                    className="mt-1 line-clamp-2 text-xs whitespace-pre-wrap text-aside"
                    title={e.text}
                  >
                    {plain(e.text)}
                  </p>
                )}

                {(e.cost || e.exit) && (
                  <p className="mt-1.5 flex gap-1.5">
                    {e.cost ? <Chip>{money(e.cost)}</Chip> : null}
                    {e.exit ? <Chip>exit {e.exit}</Chip> : null}
                  </p>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}

// Chip is a value worth reading exactly: a cost, an exit code.
function Chip({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded bg-well px-1.5 py-px font-mono text-[10px] text-aside">{children}</span>
  );
}

// by is who or what did it, and nothing at all for Orbit itself.
//
// It used to answer "orbit" for everything an engine did not do, which put
// the same italic word under twenty rows in a row. What a reader is looking
// for on this line is the exception: which engine, on which model. Orbit
// wrote the rest, and a line saying so on every row says nothing.
function by(e: Entry): string {
  if (e.engine && e.model) return `${e.engine} · ${e.model}`;

  return e.engine ?? "";
}

function when(at: string): string {
  const t = new Date(at);
  if (Number.isNaN(t.getTime())) return "—";

  return t.toLocaleString([], {
    day: "numeric",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}
