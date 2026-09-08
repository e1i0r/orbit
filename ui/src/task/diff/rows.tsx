// The rows of a diff: the two ways of laying a hunk out, and one line drawn.
//
// Both layouts read the same Line, so a change to how a line is coloured,
// wrapped or numbered lands in both. They were one file with the view around
// them; they are their own because a file's header and a file's rows are two
// subjects and the file was getting long.

import { memo } from "react";
import type { Line } from "./parse";
import { paint } from "./paint";

// Look is how the reader has asked to see the rows.
export interface Look {
  language?: string;
  /** wrap folds a long line into the column instead of scrolling it. */
  wrap: boolean;
  /** at is the line the reader picked, as "before:N" or "after:N". */
  at?: string;
  pick: (key: string) => void;
}

// Both layouts are memoised on the lines they were given and the look they
// were given. A file is a dozen hunks and a page is a dozen files, and
// without this every one of them was rebuilt whenever a reader picked a line
// in any one of them.
export const Unified = memo(function Unified({ lines, look }: { lines: Line[]; look: Look }) {
  return (
    <table className="w-full border-collapse">
      <tbody>
        {lines.map((line, i) => (
          <tr key={i} className={ground(line, look)}>
            <Gutter n={line.before} side="before" look={look} />
            <Gutter n={line.after} side="after" look={look} />
            <td className="w-3 pr-1 pl-1.5 text-center select-none">{mark(line)}</td>
            <td className={`w-full pr-3 ${flow(look)}`}>{ink(line, look.language)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
});

// Split puts what was beside what is. The rows are built by pairing the runs
// the way the parser paired them, so a line and its edit sit level.
export const Split = memo(function Split({ lines, look }: { lines: Line[]; look: Look }) {
  const rows: [Line | undefined, Line | undefined][] = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!line) continue;

    if (line.kind === "context") {
      rows.push([line, line]);
      continue;
    }

    if (line.kind === "removed") {
      let gone = i;
      while (lines[gone]?.kind === "removed") gone++;

      let came = gone;
      while (lines[came]?.kind === "added") came++;

      const most = Math.max(gone - i, came - gone);
      for (let n = 0; n < most; n++) rows.push([lines[i + n], lines[gone + n]]);

      i = came - 1;
      continue;
    }

    if (line.kind === "added") rows.push([undefined, line]);
  }

  return (
    <table className="w-full table-fixed border-collapse">
      <tbody>
        {rows.map(([was, now], i) => (
          <tr key={i}>
            <Side line={was?.kind === "added" ? undefined : was} side="before" look={look} />
            <Side line={now?.kind === "removed" ? undefined : now} side="after" look={look} edge />
          </tr>
        ))}
      </tbody>
    </table>
  );
});

function Side({
  line,
  side,
  look,
  edge,
}: {
  line?: Line;
  side: "before" | "after";
  look: Look;
  edge?: boolean;
}) {
  const n = line ? (line.kind === "added" ? line.after : line.before) : undefined;

  return (
    <>
      <td
        onClick={() => n && look.pick(`${side}:${n}`)}
        title="Copy the path and this line"
        className={`w-10 shrink-0 cursor-pointer border-r border-edge/60 px-1.5 text-right text-faint select-none hover:text-said ${
          edge ? "border-l" : ""
        } ${line ? ground(line, look, side) : "bg-panel/30"}`}
      >
        {n ?? ""}
      </td>
      {/* max-w-0 with a fixed table is what keeps a long line on its own
          side of the divide: without it the text runs across the middle and
          over the other side's numbers. What is cut is reachable — the Wrap
          switch folds it, and the whole line is in the row's title. */}
      <td
        title={line?.text}
        className={`w-1/2 max-w-0 overflow-hidden pr-3 pl-1.5 ${flow(look)} ${
          line ? ground(line, look, side) : "bg-panel/30"
        }`}
      >
        {line ? ink(line, look.language) : ""}
      </td>
    </>
  );
}

function Gutter({ n, side, look }: { n?: number; side: "before" | "after"; look: Look }) {
  return (
    <td
      onClick={() => n && look.pick(`${side}:${n}`)}
      title="Copy the path and this line"
      className="w-10 cursor-pointer border-r border-edge/60 px-1.5 text-right text-faint tabular-nums select-none hover:text-said"
    >
      {n ?? ""}
    </td>
  );
}

// flow is whether a long line is folded into the column or runs off it.
function flow(look: Look): string {
  return look.wrap ? "break-all whitespace-pre-wrap" : "whitespace-pre";
}

// ground is what a row is laid on: what happened to it, and a ring when it
// is the line the reader picked.
function ground(line: Line, look: Look, side?: "before" | "after"): string {
  const picked =
    look.at &&
    ((side !== "after" && look.at === `before:${line.before}`) ||
      (side !== "before" && look.at === `after:${line.after}`));

  const on = picked ? " outline outline-accent/60 -outline-offset-1" : "";

  // Strong enough to read at a glance, because in split there is no plus or
  // minus in front of the line: the tint is the whole of what says which
  // side you are looking at.
  if (line.kind === "added") return `bg-ok/15${on}`;
  if (line.kind === "removed") return `bg-bad/15${on}`;

  return on.trim();
}

function mark(line: Line): string {
  if (line.kind === "added") return "+";
  if (line.kind === "removed") return "−";

  return " ";
}

// ink is the line as it is read: coloured by the language it is written in,
// with the part the edit touched standing out of it.
function ink(line: Line, language?: string) {
  const strong =
    line.kind === "added"
      ? "bg-ok/30 rounded-sm"
      : line.kind === "removed"
        ? "bg-bad/30 rounded-sm"
        : "";

  return <>{paint(line.text, line.parts, language, strong)}</>;
}
