// Everything ever said about a task, and a copy of it to keep.
//
// It is the conversation and not the log: the timeline shows every event,
// the notes show what a person wrote, and this shows the words — a person's
// and each engine's, whichever program they were typed in.
//
// The text arrives already rendered, the same markdown the window draws and
// the same file an engine is handed when a terminal is opened on the task.
// Laying it out again here would be a third opinion about one conversation.
//
// The export is the point as much as the reading. A task walked by claude
// until its quota ran out, carried on by codex and finished by claude again
// has its whole account here, and a person who wants to take that somewhere
// Orbit does not reach should be able to.

import { useEffect, useState } from "react";
import { api, type Told } from "../api";
import { Empty } from "../parts/Empty";

export function HistoryView({ task }: { task: string }) {
  const [told, setTold] = useState<Told>();
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let stale = false;

    api
      .history(task)
      .then((got) => !stale && setTold(got))
      .catch((e: Error) => !stale && setTold({ id: task, read: false, failed: e.message }));

    return () => {
      stale = true;
    };
  }, [task]);

  if (!told) return <p className="text-xs text-aside">Reading the conversation…</p>;

  if (told.failed) {
    return (
      <div className="max-w-[900px] rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
        <p className="text-xs text-bad">{told.failed}</p>
      </div>
    );
  }

  if (!told.read) {
    return (
      <Empty
        said="Nothing was read"
        next="This build has no store to ask, so it says nothing rather than pretending the conversation is empty."
      />
    );
  }

  const text = told.text ?? "";

  const keep = () => {
    // A blob and a click, because the file is built here rather than fetched:
    // what a reader saves is exactly what they are looking at.
    const url = URL.createObjectURL(new Blob([text], { type: "text/markdown" }));
    const a = document.createElement("a");

    a.href = url;
    a.download = `${task}-history.md`;
    a.click();

    URL.revokeObjectURL(url);
  };

  const copy = () => {
    navigator.clipboard?.writeText(text).then(
      () => {
        setCopied(true);
        setTimeout(() => setCopied(false), 1600);
      },
      () => setCopied(false),
    );
  };

  return (
    <div className="flex max-w-[900px] flex-col gap-2.5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-[11px] text-faint">
          Every word said about this task, oldest first, in whichever program it was said. Kept in
          the task's own directory, so it outlives every checkout.
        </p>

        <div className="flex shrink-0 items-center gap-1.5">
          <button
            onClick={copy}
            className="rounded border border-edge px-2 py-0.5 text-[11px] text-aside transition-colors hover:bg-hover hover:text-said"
          >
            {copied ? "copied" : "Copy"}
          </button>
          <button
            onClick={keep}
            className="rounded border border-accent/40 bg-accent/10 px-2 py-0.5 text-[11px] text-accent transition-colors hover:bg-accent/20"
          >
            Export markdown
          </button>
        </div>
      </div>

      <pre className="overflow-x-auto rounded-md border border-edge bg-panel px-3 py-2.5 font-mono text-[11px] leading-[1.6] whitespace-pre-wrap text-aside">
        {text}
      </pre>
    </div>
  );
}
