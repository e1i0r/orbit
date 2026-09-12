// Markdown as its own words, for the two lines a preview has room for.
//
// An engine answers in markdown, and the timeline drew those two lines
// exactly as written: the first thing a reader saw under "phase finished"
// was "## Decisions", which is the heading of what was said rather than
// anything that was said. The heading marks, the bullet dashes and the
// blank lines between them are all syntax; under a clamp they are the whole
// of what fits.
//
// The full text is still the full text — the timeline keeps it as the row's
// title, and every pane that shows the answer whole renders the markdown.
export function plain(said: string): string {
  return said
    .split("\n")
    .map((line) => line.replace(/^\s{0,3}#{1,6}\s+/, "").replace(/^\s*[-*+]\s+/, ""))
    .map((line) => line.trim())
    .filter((line) => line !== "")
    .join("\n");
}
