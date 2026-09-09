// Where the cells of a honeycomb go.
//
// Flat-top hexagons on an axial lattice, laid out in rings from the centre
// outward. The centre is the heaviest cell of the level, so the eye lands on
// the change before it lands on anything else — which is the whole reason
// this drawing exists rather than a list.
//
// The arithmetic is the standard axial one: a flat-top hexagon of radius r
// is 2r wide and √3·r tall, columns step by 1.5r, and every step in q lifts
// the row by half a cell. Nothing here is tuned by eye; a lattice that is
// almost right is a lattice with gaps in it.

export interface Axial {
  q: number;
  r: number;
}

export interface Placed extends Axial {
  x: number;
  y: number;
}

// The six neighbours of a cell, in the order a ring is walked. They are the
// neighbours of *this* projection and not a set copied from a diagram: with
// x = 1.5r·q and y = √3r·(r + q/2), those six and no others land one cell
// away. A set that is almost right draws a lattice with cells on top of each
// other, which reads as a smaller repository rather than as a bug.
const around: Axial[] = [
  { q: 1, r: 0 },
  { q: 1, r: -1 },
  { q: 0, r: -1 },
  { q: -1, r: 0 },
  { q: -1, r: 1 },
  { q: 0, r: 1 },
];

// spiral is n cells from the centre outward: the origin, then each ring
// walked in turn.
//
// Rings rather than rows because a level is a set of siblings with no order
// of its own beyond the one we give it. Rows would put the last child a
// screen away from the first; a spiral keeps every sibling within a cell or
// two of the middle, which is what makes a level readable at a glance.
export function spiral(n: number): Axial[] {
  const out: Axial[] = [{ q: 0, r: 0 }];

  for (let ring = 1; out.length < n; ring++) {
    // A ring is walked from the corner one step out along the fifth
    // direction, then six sides of `ring` steps each. Starting anywhere else
    // leaves the walk short of where it began.
    let at: Axial = { q: around[4].q * ring, r: around[4].r * ring };

    for (const step of around) {
      for (let i = 0; i < ring; i++) {
        if (out.length >= n) return out;

        out.push(at);
        at = { q: at.q + step.q, r: at.r + step.r };
      }
    }
  }

  return out;
}

/** touching is the six coordinates around one, in lattice order. */
export function touching(at: Axial): Axial[] {
  return around.map((d) => ({ q: at.q + d.q, r: at.r + d.r }));
}

/** place turns axial coordinates into pixels for a hexagon of radius r. */
export function place(cells: Axial[], r: number): Placed[] {
  const tall = Math.sqrt(3) * r;

  return cells.map((c) => ({
    ...c,
    x: 1.5 * r * c.q,
    y: tall * (c.r + c.q / 2),
  }));
}

/** corners is the outline of one flat-top hexagon of radius r, centred on 0. */
export function corners(r: number): string {
  return [0, 1, 2, 3, 4, 5]
    .map((i) => {
      const a = (Math.PI / 180) * (60 * i);
      return `${(r * Math.cos(a)).toFixed(2)},${(r * Math.sin(a)).toFixed(2)}`;
    })
    .join(" ");
}

/** bounds is the box the placed cells occupy, with room for their corners. */
export function bounds(cells: Placed[], r: number) {
  const tall = Math.sqrt(3) * r;
  const xs = cells.map((c) => c.x);
  const ys = cells.map((c) => c.y);
  const pad = 2;

  return {
    x: Math.min(...xs) - r - pad,
    y: Math.min(...ys) - tall / 2 - pad,
    w: Math.max(...xs) - Math.min(...xs) + 2 * r + 2 * pad,
    h: Math.max(...ys) - Math.min(...ys) + tall + 2 * pad,
  };
}
