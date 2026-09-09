// Reading a unified diff into the shape a reader looks at.
//
// git writes one stream; a person reads files, and inside a file they read
// hunks, and inside a hunk they read one line against another. This turns
// the stream into that, keeping both line numbers, because "which line is
// this" is the first question anybody asks of a diff.

export type LineKind = "context" | "added" | "removed" | "meta";

export interface Line {
  kind: LineKind;
  text: string;
  /** The number this line has in the file before the change, when it has one. */
  before?: number;
  /** And after it. */
  after?: number;
  /** Which parts of the text changed, when this line was paired with another. */
  parts?: Part[];
}

export interface Part {
  text: string;
  changed: boolean;
}

export interface Hunk {
  /** git's own @@ header, and what it says the hunk is inside of. */
  header: string;
  inside: string;
  lines: Line[];
  /** Where the hunk begins and how long it is, on each side. It is what
   *  says how big the gap to the hunk before it is, which is the whole of
   *  what "expand" needs to know. */
  beforeAt: number;
  beforeFor: number;
  afterAt: number;
  afterFor: number;
}

/** What happened to the file as a whole. git says it in the header, and a
 *  reader who cannot see it reads a new file as a rewrite of an old one. */
export type Status = "added" | "deleted" | "renamed" | "modified";

export interface File {
  name: string;
  /** was is the old path, and it differs from name only on a rename. */
  was?: string;
  status: Status;
  /** mode is the two file modes when only they changed, "100644 → 100755". */
  mode?: string;
  added: number;
  removed: number;
  binary: boolean;
  hunks: Hunk[];
}

const header = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@ ?(.*)$/;

export function parse(text: string): File[] {
  const files: File[] = [];
  let file: File | undefined;
  let hunk: Hunk | undefined;
  let before = 0;
  let after = 0;

  for (const line of text.split("\n")) {
    if (line.startsWith("diff --git ")) {
      file = {
        name: pathOf(line),
        status: "modified",
        added: 0,
        removed: 0,
        binary: false,
        hunks: [],
      };
      files.push(file);
      hunk = undefined;
      continue;
    }

    if (!file) continue;

    if (line.startsWith("new file mode")) {
      file.status = "added";
      continue;
    }

    if (line.startsWith("deleted file mode")) {
      file.status = "deleted";
      continue;
    }

    if (line.startsWith("rename from ")) {
      file.was = line.slice("rename from ".length);
      file.status = "renamed";
      continue;
    }

    if (line.startsWith("old mode ")) {
      file.mode = line.slice("old mode ".length);
      continue;
    }

    if (line.startsWith("new mode ") && file.mode) {
      file.mode = `${file.mode} → ${line.slice("new mode ".length)}`;
      continue;
    }

    if (line.startsWith("Binary files")) {
      file.binary = true;
      continue;
    }

    const at = header.exec(line);
    if (at) {
      before = Number(at[1]);
      after = Number(at[3]);
      hunk = {
        header: line.slice(0, line.lastIndexOf("@@") + 2),
        inside: at[5] ?? "",
        lines: [],
        beforeAt: before,
        beforeFor: at[2] === undefined ? 1 : Number(at[2]),
        afterAt: after,
        afterFor: at[4] === undefined ? 1 : Number(at[4]),
      };
      file.hunks.push(hunk);
      continue;
    }

    if (!hunk) continue;

    if (line.startsWith("+")) {
      hunk.lines.push({ kind: "added", text: line.slice(1), after: after++ });
      file.added++;
    } else if (line.startsWith("-")) {
      hunk.lines.push({ kind: "removed", text: line.slice(1), before: before++ });
      file.removed++;
    } else if (line.startsWith("\\")) {
      hunk.lines.push({ kind: "meta", text: line });
    } else {
      hunk.lines.push({ kind: "context", text: line.slice(1), before: before++, after: after++ });
    }
  }

  for (const one of files) {
    for (const h of one.hunks) pairUp(h);
  }

  return files;
}

// pairUp marks what actually changed inside a line.
//
// A run of removed lines followed by the same number of added ones is a
// person editing those lines, and saying which words moved is the difference
// between reading a diff and comparing two paragraphs by eye. Runs of
// different lengths are left alone: pairing them would be a guess about
// which line became which.
function pairUp(hunk: Hunk) {
  const lines = hunk.lines;

  for (let i = 0; i < lines.length; i++) {
    if (lines[i]?.kind !== "removed") continue;

    let gone = i;
    while (lines[gone]?.kind === "removed") gone++;

    let came = gone;
    while (lines[came]?.kind === "added") came++;

    const removed = gone - i;
    const added = came - gone;

    if (removed > 0 && removed === added) {
      for (let n = 0; n < removed; n++) {
        const was = lines[i + n];
        const now = lines[gone + n];
        if (!was || !now) continue;

        const [left, right] = words(was.text, now.text);
        was.parts = left;
        now.parts = right;
      }
    }

    i = came - 1;
  }
}

// words is the two sides of one edited line, each cut into what stayed and
// what moved.
//
// Split on word boundaries rather than characters: a character diff of two
// lines of code marks every bracket and reads as noise, and what a person is
// looking for is the identifier that changed.
export function words(was: string, now: string): [Part[], Part[]] {
  const a = was.match(/\w+|\s+|[^\w\s]/g) ?? [];
  const b = now.match(/\w+|\s+|[^\w\s]/g) ?? [];

  let head = 0;
  while (head < a.length && head < b.length && a[head] === b[head]) head++;

  let tail = 0;
  while (
    tail < a.length - head &&
    tail < b.length - head &&
    a[a.length - 1 - tail] === b[b.length - 1 - tail]
  ) {
    tail++;
  }

  const same = (from: string[], at: number, to: number) => from.slice(at, to).join("");

  const side = (from: string[]): Part[] =>
    [
      { text: same(from, 0, head), changed: false },
      { text: same(from, head, from.length - tail), changed: true },
      { text: same(from, from.length - tail, from.length), changed: false },
    ].filter((p) => p.text !== "");

  return [side(a), side(b)];
}

// pathOf reads the file's name off git's own header, taking the side it
// ended up on.
function pathOf(header: string): string {
  const to = header.split(" ").pop() ?? "";

  return to.replace(/^b\//, "");
}
