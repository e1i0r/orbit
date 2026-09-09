// One level of the repository, as a honeycomb.
//
// The map is the tree: the root is the project, every cell is a directory or
// a file in it, and going into a cell is going into the folder. Nothing is
// invented — a repository is already built in some shape, and this draws the
// shape that is there.
//
// A cell is lit when the task changed something under it, and how strongly
// says how much: a directory of five files barely edited and one file
// rewritten are different things and read the same by count. So the eye goes
// to the weight, and the weight is where the work happened.
//
// Three gestures, because there are three questions. One click asks "what is
// under here" and answers in the panel beside it, all the way down to the
// files — from the root to the lines without descending. Holding cmd or
// shift asks "and this one too", and the panel answers for all of them at
// once, which is how two halves of a change that live in different corners
// are read side by side. A double click asks "take me in", and the level is
// replaced.

import { useMemo } from "react";
import type { Cell } from "../../api";
import { bounds, corners, place, spiral, touching, type Axial } from "./hex";

/** The radius of one cell, in pixels. */
const r = 46;

interface Props {
  cells: Cell[];
  /** picked is every cell the panel beside this is showing, by path. */
  picked: string[];
  /** also says the click asked to add to the selection rather than replace it. */
  onPick: (cell: Cell, also: boolean) => void;
  onEnter: (cell: Cell) => void;
}

export function Honeycomb({ cells, picked, onPick, onEnter }: Props) {
  // The heaviest cell takes the middle, and everything after it goes where
  // the history says it belongs — see arrange. Two jobs, and the order they
  // are done in is which one wins where they disagree: the change is what
  // the reader came for, so it is never moved off the centre to make a
  // neighbourhood tidier.
  const order = useMemo(() => arrange(cells), [cells]);
  const laid = useMemo(() => place(order.map((o) => o.at), r), [order]);
  const box = useMemo(() => bounds(laid, r), [laid]);
  const shape = useMemo(() => corners(r - 1.5), []);

  // The busiest cell sets the scale. Absolute line counts would make a
  // one-line fix invisible in a repository that has seen a big change, and
  // the question this answers is "where did this task go", not "how big is
  // this task compared to another".
  const most = Math.max(...order.map((o) => o.cell.lines ?? 0), 1);

  if (order.length === 0) return null;

  return (
    <svg
      viewBox={`${box.x} ${box.y} ${box.w} ${box.h}`}
      className="w-full"
      style={{ maxHeight: "min(58vh, 460px)" }}
      role="group"
      aria-label="the repository at this level"
    >
      {order.map(({ cell }, i) => (
        <Comb
          key={cell.path || cell.name}
          cell={cell}
          at={laid[i]}
          shape={shape}
          most={most}
          on={picked.includes(cell.path)}
          onPick={onPick}
          onEnter={onEnter}
        />
      ))}
    </svg>
  );
}

// Comb is one cell: its outline, how lit it is, what it is called, and what
// the task did in it.
function Comb({
  cell,
  at,
  shape,
  most,
  on,
  onPick,
  onEnter,
}: {
  cell: Cell;
  at: { x: number; y: number };
  shape: string;
  most: number;
  on: boolean;
  onPick: (c: Cell, also: boolean) => void;
  onEnter: (c: Cell) => void;
}) {
  const changed = cell.changed ?? 0;
  const inside = (cell.cells?.length ?? 0) > 0;

  // A touched cell is never fainter than a quarter, so that one line in a
  // repository somebody rewrote is still visibly lit rather than technically
  // lit. Nothing else on this drawing is louder than the change.
  const heat = changed > 0 ? 0.25 + 0.75 * Math.min(1, (cell.lines ?? 0) / most) : 0;

  return (
    <g
      transform={`translate(${at.x} ${at.y})`}
      onClick={(e) => onPick(cell, e.metaKey || e.ctrlKey || e.shiftKey)}
      onDoubleClick={() => inside && onEnter(cell)}
      className="cursor-pointer"
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter") onPick(cell, e.metaKey || e.ctrlKey || e.shiftKey);
      }}
    >
      <title>
        {cell.path || cell.name}
        {changed > 0 ? ` — ${changed} changed, ${cell.lines} lines` : ""}
      </title>

      <polygon
        points={shape}
        className={
          changed > 0
            ? "fill-accent stroke-accent"
            : "fill-well stroke-edge"
        }
        style={{ fillOpacity: changed > 0 ? heat * 0.9 : 1, strokeOpacity: changed > 0 ? 1 : 0.7 }}
        strokeWidth={1}
      />

      {/* Chosen is drawn outside the cell rather than by thickening its own
          edge: lit and chosen are different facts, and a reader who has
          picked three cells has to be able to see which three without
          comparing stroke widths. */}
      {on && (
        <polygon
          points={corners(r + 1.5)}
          fill="none"
          className="stroke-said"
          strokeWidth={1.75}
        />
      )}

      {/* The ring says the cell has something inside before anybody double
          clicks it. A file and a folder are different gestures, and finding
          that out by trying is how a map loses a reader. */}
      {inside && (
        <polygon
          points={corners(r - 7)}
          fill="none"
          className={changed > 0 ? "stroke-accent" : "stroke-edge"}
          strokeWidth={0.75}
          strokeOpacity={0.55}
        />
      )}

      {/* Two lines rather than one clipped to death. A cell whose name
          reads "install_te…" is a cell the reader has to hover to identify,
          and hovering every cell is the thing a map is for avoiding. */}
      {wrapped(cell.name).map((line, i, all) => (
        <text
          key={i}
          y={(i - (all.length - 1) / 2) * 11 + (changed > 0 ? -4 : 3.5)}
          textAnchor="middle"
          className={changed > 0 ? "fill-said" : "fill-aside"}
          style={{ fontSize: 10, fontWeight: changed > 0 ? 600 : 400 }}
        >
          {line}
        </text>
      ))}

      {changed > 0 && (
        <text
          y={14}
          textAnchor="middle"
          className="fill-said"
          style={{ fontSize: 9, fontVariantNumeric: "tabular-nums", opacity: 0.7 }}
        >
          {changed} · {cell.lines}
        </text>
      )}
    </g>
  );
}

// wrapped is a name across the two lines a cell has room for.
//
// It breaks where a filename already has a seam — a dot, a dash, an
// underscore — so `install_test.sh` reads as `install_` / `test.sh` rather
// than being cut mid-word. A name with no seam and no room is the only one
// that loses its tail.
function wrapped(name: string): string[] {
  const wide = 12;
  if (name.length <= wide) return [name];

  const seam = Math.max(
    name.lastIndexOf(".", wide),
    name.lastIndexOf("_", wide),
    name.lastIndexOf("-", wide),
  );

  const at = seam > 3 ? seam + 1 : wide;
  const tail = name.slice(at);

  return [name.slice(0, at), tail.length > wide ? tail.slice(0, wide - 1) + "…" : tail];
}

/** Where one cell ended up. */
interface Put {
  cell: Cell;
  at: Axial;
}

// arrange decides which cell goes in which seat.
//
// A honeycomb makes a claim a list does not: cells that touch are next to
// each other, and a reader reads that as meaning something. Ordered by name
// it means the alphabet, which is not a fact about the code. So the seats
// are filled by what the repository's own history says moves together —
// internal/record and internal/words were committed together nineteen
// times, and on this drawing they touch.
//
// The seats are walked in spiral order and each is given the best remaining
// cell, rather than each cell being given its best seat. That keeps the comb
// solid: a cell-first pass leaves holes where a favourite seat was taken,
// and a lattice with holes reads as a repository with gaps in it.
//
// It is a greedy pass and not an optimum. Six neighbours cannot hold every
// pair a busy directory has, so what this promises is that a cell's
// strongest surviving neighbours are ones it moves with — not that every
// pair the history knows about is honoured on the drawing.
function arrange(cells: Cell[]): Put[] {
  if (cells.length === 0) return [];

  // Heaviest first, so the change takes the middle seat, then by name so
  // that two readings of an untouched level seat the same cell the same way.
  const left = [...cells].sort(
    (a, b) =>
      (b.lines ?? 0) - (a.lines ?? 0) ||
      (b.changed ?? 0) - (a.changed ?? 0) ||
      a.name.localeCompare(b.name),
  );

  const taken = new Map<string, Cell>();
  const out: Put[] = [];

  for (const seat of spiral(cells.length)) {
    // What is already beside this seat, to score a pick against.
    const beside = touching(seat)
      .map((n) => taken.get(key(n)))
      .filter((c): c is Cell => c !== undefined);

    let best = 0;

    if (beside.length > 0) {
      let strongest = -1;

      left.forEach((cell, i) => {
        const score = beside.reduce((sum, near) => sum + moves(cell, near), 0);
        if (score > strongest) {
          strongest = score;
          best = i;
        }
      });
    }

    const [cell] = left.splice(best, 1);

    taken.set(key(seat), cell);
    out.push({ cell, at: seat });
  }

  return settle(out);
}

// settle swaps pairs of seats while that puts more of the history side by
// side, and stops when a pass changes nothing.
//
// The greedy pass above fills each seat with the best cell for it at the
// time, which is not the best arrangement: seating `words` beside `record`
// early cost it the seat beside `cli`, and `cli`-with-`words` is the
// strongest pair on that level. A few passes of "would these two be better
// the other way round" find that without anything resembling a solver.
//
// The middle seat never moves. It holds the heaviest cell, which is what
// the reader opened the map to find, and a neighbourhood is not worth
// shifting the change off the centre for.
function settle(put: Put[]): Put[] {
  const seats = put.map((p) => p.at);
  const cells = put.map((p) => p.cell);

  // Which seats touch which, worked out once: the arithmetic does not
  // change between passes and it is the inner loop of every score.
  const beside = seats.map((seat) => {
    const near = new Set(touching(seat).map(key));

    return seats.flatMap((other, i) => (near.has(key(other)) ? [i] : []));
  });

  const around = (i: number, cell: Cell) =>
    beside[i].reduce((sum, j) => sum + moves(cell, cells[j]), 0);

  // Bounded, because this runs on every draw of a level. Three passes over
  // twenty cells is a few hundred comparisons and settles in one or two;
  // what it must never be is a loop whose length depends on the data.
  for (let pass = 0; pass < 3; pass++) {
    let moved = false;

    for (let i = 1; i < cells.length; i++) {
      for (let j = i + 1; j < cells.length; j++) {
        const was = around(i, cells[i]) + around(j, cells[j]);
        const would = around(i, cells[j]) + around(j, cells[i]);

        if (would > was) {
          [cells[i], cells[j]] = [cells[j], cells[i]];
          moved = true;
        }
      }
    }

    if (!moved) break;
  }

  return cells.map((cell, i) => ({ cell, at: seats[i] }));
}

/** moves is how often the history committed these two together. */
function moves(cell: Cell, near: Cell): number {
  return cell.with?.find((n) => n.path === near.path)?.times ?? 0;
}

/** key is one seat, as a map key. */
function key(at: Axial): string {
  return `${at.q},${at.r}`;
}
