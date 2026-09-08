// One file of a diff: what happened to it, and the change itself.
//
// The header is the part a reader scans — a badge saying whether the file
// was added, deleted or renamed, the count, and a bar of five blocks that
// says the shape of the change before any number is read. The rows below it
// are git's hunks, with the gaps between them openable: git writes three
// lines of context and the question "what is above this" is answered by the
// file, not by a wider diff.

import { useCallback, useMemo, useState } from "react";
import { api } from "../../api";
import type { File, Hunk, Line, Status } from "./parse";
import { tongue } from "./paint";
import { Split, Unified, type Look } from "./rows";

// How many lines one press of an expander brings in. GitHub's number, and
// for the same reason: enough to see the function you are in, few enough
// that a wrong guess is cheap.
const step = 20;

// And how many one press of "all" may bring in. A generated file with a
// change in the middle of it has thousands of lines above that change, and
// rendering them all at once is how a tab stops answering. Pressing again
// carries on.
const most = 500;

const badges: Record<Status, { said: string; tone: string }> = {
  added: { said: "added", tone: "bg-ok/15 text-ok" },
  deleted: { said: "deleted", tone: "bg-bad/15 text-bad" },
  renamed: { said: "renamed", tone: "bg-accent/15 text-accent" },
  modified: { said: "changed", tone: "bg-edge/60 text-aside" },
};

export function FileDiff({
  file,
  task,
  split,
  wrap,
  shut,
  toggle,
  viewed,
  see,
}: {
  file: File;
  task: string;
  split: boolean;
  wrap: boolean;
  shut: boolean;
  toggle: () => void;
  viewed: boolean;
  see: (yes: boolean) => void;
}) {
  const language = tongue(file.name);
  // Guarded rather than indexed: a status this build does not know is a
  // header git added since, and it must not take the page down with it.
  const badge = badges[file.status] ?? badges.modified;
  const [at, setAt] = useState<string>();
  const [whole, setWhole] = useState<string[]>();
  const [gone, setGone] = useState(false);
  // open is, per gap, the run of after-side lines the reader has asked for.
  const [open, setOpen] = useState<Record<number, [number, number]>>({});

  // Stable, so the memo on the row tables can do its job: a look rebuilt on
  // every render is a new object every render, and memo compares by identity.
  const pick = useCallback(
    (key: string) => {
      setAt(key);
      navigator.clipboard?.writeText(`${file.name}:${key.split(":")[1]}`).catch(() => {
        // A browser that will not give the clipboard is not a failure worth
        // interrupting a reader over; the line is still marked.
      });
    },
    [file.name],
  );

  const look: Look = useMemo(
    () => ({ language, wrap, at, pick }),
    [language, wrap, at, pick],
  );

  // The file is read once, on the first press of any expander, and never for
  // a file nobody opens out: a diff of forty files would otherwise be forty
  // reads of forty files nobody asked to see more of.
  const read = async () => {
    if (whole || gone) return whole;

    const got = await api.file(task, file.name);
    if (got.missing || got.text === undefined) {
      setGone(true);

      return undefined;
    }

    const lines = got.text.split("\n");
    setWhole(lines);

    return lines;
  };

  const show = async (gap: number, from: number, to: number) => {
    const lines = await read();
    if (!lines) return;

    setOpen((was) => {
      const had = was[gap];

      return { ...was, [gap]: had ? [Math.min(had[0], from), Math.max(had[1], to)] : [from, to] };
    });
  };

  return (
    <section id={anchorOf(file.name)} className="scroll-mt-2 overflow-hidden rounded-md border border-edge">
      <header className="sticky top-0 z-10 flex items-center gap-2 border-b border-edge bg-panel px-3 py-1.5">
        <button
          onClick={toggle}
          aria-expanded={!shut}
          className="text-faint transition-colors hover:text-said"
          title={shut ? "Show this file" : "Hide this file"}
        >
          <span className={`inline-block transition-transform ${shut ? "" : "rotate-90"}`}>›</span>
        </button>

        <span className={`shrink-0 rounded px-1.5 py-px text-[10px] ${badge.tone}`}>
          {badge.said}
        </span>

        <h3 className="min-w-0 flex-1 truncate font-mono text-[11px] text-said" title={file.name}>
          {file.was && <span className="text-faint">{file.was} → </span>}
          {file.name}
          {file.mode && <span className="text-faint"> · {file.mode}</span>}
        </h3>

        <Blocks added={file.added} removed={file.removed} />

        <span className="shrink-0 font-mono text-[11px]">
          <span className="text-ok">+{file.added}</span>{" "}
          <span className="text-bad">−{file.removed}</span>
        </span>

        <button
          onClick={() => void navigator.clipboard?.writeText(file.name)}
          className="shrink-0 text-[10px] text-faint transition-colors hover:text-said"
          title="Copy the path"
        >
          copy
        </button>

        <label className="flex shrink-0 cursor-pointer items-center gap-1 text-[10px] text-faint select-none">
          <input
            type="checkbox"
            checked={viewed}
            onChange={(e) => see(e.target.checked)}
            className="size-2.5 accent-accent"
          />
          Viewed
        </label>
      </header>

      {!shut &&
        (file.binary ? (
          <p className="bg-well px-3 py-2 text-[11px] text-faint">Binary file, not shown.</p>
        ) : (
          <div className="overflow-x-auto bg-well font-mono text-[11.5px] leading-[1.55]">
            {file.hunks.map((hunk, i) => (
              // The browser is told it may skip a hunk it is not showing.
              // A long file is thousands of rows, and laying out the ones
              // nobody is looking at is most of the cost of having them.
              <div key={i} className="[content-visibility:auto] [contain-intrinsic-size:auto_600px]">
                <Gap
                  file={file}
                  gap={i}
                  whole={whole}
                  gone={gone}
                  open={open[i]}
                  show={show}
                  look={look}
                  split={split}
                />
                <div className="flex gap-2 bg-accent/8 px-3 py-0.5 text-accent">
                  <span>{hunk.header}</span>
                  {hunk.inside && <span className="truncate text-faint">{hunk.inside}</span>}
                </div>
                {split ? (
                  <Split lines={hunk.lines} look={look} />
                ) : (
                  <Unified lines={hunk.lines} look={look} />
                )}
              </div>
            ))}

            <Gap
              file={file}
              gap={file.hunks.length}
              whole={whole}
              gone={gone}
              open={open[file.hunks.length]}
              show={show}
              look={look}
              split={split}
            />
          </div>
        ))}
    </section>
  );
}

// Gap is the unchanged run between two hunks: the lines the reader has asked
// to see of it, and the controls that ask for more.
function Gap({
  file,
  gap,
  whole,
  gone,
  open,
  show,
  look,
  split,
}: {
  file: File;
  gap: number;
  whole?: string[];
  gone: boolean;
  open?: [number, number];
  show: (gap: number, from: number, to: number) => void;
  look: Look;
  split: boolean;
}) {
  const range = gapOf(file.hunks, gap, whole?.length);
  if (!range || gone) return null;

  const [from, to] = range;
  const offset = offsetOf(file.hunks, gap);

  // Nothing opened yet is one run of hidden lines, not two, so it gets one
  // row: an arrow for each end of it, and the count to take the whole gap.
  if (!open) {
    return gone ? null : (
      <Opener
        said={`${to - from + 1} more`}
        down={() => show(gap, from, Math.min(to, from + step - 1))}
        up={() => show(gap, Math.max(from, to - step + 1), to)}
        all={() => show(gap, Math.max(from, to - most + 1), to)}
      />
    );
  }

  const [shownFrom, shownTo] = open;
  const lines: Line[] = [];

  for (let n = shownFrom; n <= shownTo; n++) {
    lines.push({ kind: "context", text: whole?.[n - 1] ?? "", after: n, before: n - offset });
  }

  return (
    <>
      {shownFrom > from && (
        <Opener
          said={`${shownFrom - from} more`}
          up={() => show(gap, Math.max(from, shownFrom - step), shownFrom - 1)}
          all={() => show(gap, Math.max(from, shownFrom - most), shownFrom - 1)}
        />
      )}

      {split ? <Split lines={lines} look={look} /> : <Unified lines={lines} look={look} />}

      {shownTo < to && (
        <Opener
          said={`${to - shownTo} more`}
          down={() => show(gap, shownTo + 1, Math.min(to, shownTo + step))}
          all={() => show(gap, shownTo + 1, Math.min(to, shownTo + most))}
        />
      )}
    </>
  );
}

// Opener is one row of controls for the lines nobody has asked for yet.
function Opener({
  said,
  up,
  down,
  all,
}: {
  said: string;
  up?: () => void;
  down?: () => void;
  all: () => void;
}) {
  return (
    <div className="flex items-center gap-2 border-y border-edge/40 bg-panel/40 px-3 py-0.5">
      {up && (
        <button onClick={up} className="text-accent transition-colors hover:text-said" title="Show the lines above">
          ↑
        </button>
      )}
      {down && (
        <button onClick={down} className="text-accent transition-colors hover:text-said" title="Show the lines below">
          ↓
        </button>
      )}
      <button
        onClick={all}
        className="text-[10px] text-faint transition-colors hover:text-said"
        title="Show all of them"
      >
        {said}
      </button>
    </div>
  );
}

// Blocks is the shape of the change in five squares, read before any number
// is: mostly green is a file that grew, mostly red one that was cut back.
function Blocks({ added, removed }: { added: number; removed: number }) {
  const total = added + removed;
  const green = total === 0 ? 0 : Math.max(added > 0 ? 1 : 0, Math.round((added / total) * 5));
  const red = total === 0 ? 0 : Math.min(5 - green, Math.max(removed > 0 ? 1 : 0, 5 - green));

  return (
    <span className="flex shrink-0 gap-px" aria-hidden>
      {Array.from({ length: 5 }, (_, i) => (
        <span
          key={i}
          className={`size-1.5 ${i < green ? "bg-ok" : i < green + red ? "bg-bad" : "bg-edge"}`}
        />
      ))}
    </span>
  );
}

// gapOf is the run of after-side lines between one hunk and the one before
// it, and nothing when the two touch.
function gapOf(hunks: Hunk[], gap: number, lines?: number): [number, number] | undefined {
  const next = hunks[gap];
  const last = hunks[gap - 1];

  const from = last ? last.afterAt + last.afterFor : 1;
  const to = next ? next.afterAt - 1 : (lines ?? 0) - 1;

  return to >= from ? [from, to] : undefined;
}

// offsetOf is how far the two sides have drifted apart by this point in the
// file, so an unchanged line opened out can be given both its numbers.
function offsetOf(hunks: Hunk[], gap: number): number {
  const at = hunks[gap] ?? hunks[gap - 1];
  if (!at) return 0;

  return at.afterAt - at.beforeAt;
}

// anchorOf is a file's own id on the page, for the tree to jump to.
export function anchorOf(name: string): string {
  return `file-${name.replace(/[^\w-]/g, "-")}`;
}
