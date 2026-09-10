// What the task changed.
//
// The diff is the thing a terminal draws worst, so this is where the browser
// earns its place: both line numbers, the changed words marked inside the
// line, syntax colour, a list to jump from, the unchanged lines openable,
// and a record of what you have already read. Unified by default because
// that is how a change reads; split when the question is "what did this
// become".

import { useEffect, useMemo, useState } from "react";

import { useDesk } from "../shell/narrow";
import { api, type Diff } from "../api";
import { Empty } from "../parts/Empty";
import { FileDiff, anchorOf } from "./diff/FileDiff";
import { parse } from "./diff/parse";
import { Tree } from "./diff/Tree";

// big is a file nobody wants rendered before they have said so. A rewrite
// of a generated file is thousands of rows of syntax highlighting between
// the reader and the four files they came to look at.
const big = 1500;

export function DiffView({ diff, task }: { diff?: Diff; task: string }) {
  // space is the reader's answer to a change that is half reformatting: ask
  // git again without it, rather than filter here, so the counts and the
  // hunks agree with what is shown.
  const [space, setSpace] = useState<"keep" | "ignore">("keep");
  const [tight, setTight] = useState<Diff>();
  const [split, setSplit] = useState(false);
  // Two columns of forty characters is not a diff, so on a phone the
  // question is not asked: unified, and the switch is not drawn.
  const desk = useDesk();
  const [wrap, setWrap] = useState(false);
  const [shut, setShut] = useState<Record<string, boolean>>({});
  const [viewed, setViewed] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (space === "keep") return;

    let stale = false;

    api
      .diff(task, { space: "ignore" })
      .then((d) => !stale && setTight(d))
      .catch((e: Error) => !stale && setTight({ id: task, failed: e.message }));

    return () => {
      stale = true;
    };
  }, [task, space]);

  const showing = space === "ignore" ? tight : diff;
  const files = useMemo(() => (showing?.text ? parse(showing.text) : []), [showing?.text]);

  // The large ones start shut, and only the large ones: a diff that opens
  // with everything folded is a diff the reader has to unfold before they
  // can see what they came for.
  useEffect(() => {
    setShut(Object.fromEntries(files.filter((f) => f.added + f.removed > big).map((f) => [f.name, true])));
  }, [files]);

  // n and p walk the files, the way they do on a pull request. They are off
  // while somebody is typing in the filter, where n means n.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      if (e.target instanceof HTMLElement && e.target.tagName === "INPUT") return;
      if (e.key !== "n" && e.key !== "p") return;

      const names = files.map((f) => f.name);
      const here = names.findIndex((name) => {
        const box = document.getElementById(anchorOf(name))?.getBoundingClientRect();

        return box !== undefined && box.bottom > 80;
      });

      const next = names[Math.min(names.length - 1, Math.max(0, here + (e.key === "n" ? 1 : -1)))];
      if (next) document.getElementById(anchorOf(next))?.scrollIntoView({ block: "start" });
    };

    addEventListener("keydown", onKey);

    return () => removeEventListener("keydown", onKey);
  }, [files]);

  if (!showing) return <p className="text-xs text-aside">Reading the worktree…</p>;

  if (showing.missing) {
    return (
      <Empty
        said="No checkout to look at"
        next="A task has a worktree from the moment its first phase runs. Start it, and the diff lands here."
      />
    );
  }

  if (showing.failed) {
    return (
      <div className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
        <p className="text-xs text-bad">{showing.failed}</p>
      </div>
    );
  }

  if (showing.empty || files.length === 0) {
    return <Empty said="Nothing changed in this task's worktree" />;
  }

  const added = files.reduce((n, f) => n + f.added, 0);
  const removed = files.reduce((n, f) => n + f.removed, 0);
  const read = files.filter((f) => viewed[f.name]).length;
  const allShut = files.every((f) => shut[f.name]);

  const jump = (name: string) => {
    setShut((was) => ({ ...was, [name]: false }));
    document.getElementById(anchorOf(name))?.scrollIntoView({ block: "start" });
  };

  return (
    <div className="flex flex-col gap-3 lg:flex-row lg:gap-4">
      {/* The list of files is what makes a thirty-file diff navigable, and
          at 390 columns it is half the screen spent on navigation. Each
          file's own card carries its name and its numbers, so on a phone
          the diff is read by scrolling it. */}
      {desk && <Tree files={files} viewed={viewed} jump={jump} />}

      <div className="flex min-w-0 flex-1 flex-col gap-2.5">
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-edge bg-panel px-3 py-2">
          <p className="text-xs text-aside">
            <span className="text-said">{files.length}</span>{" "}
            {files.length === 1 ? "file" : "files"} changed,{" "}
            <span className="font-mono text-ok">+{added}</span>{" "}
            <span className="font-mono text-bad">−{removed}</span>
            <span className="text-faint">
              {" · "}
              {read} of {files.length} viewed
            </span>
          </p>

          <div className="flex items-center gap-1.5">
            {desk && (
              <Pick
                at={split}
                set={setSplit}
                of={[
                  [false, "Unified"],
                  [true, "Split"],
                ]}
              />
            )}
            <Switch on={wrap} flip={() => setWrap(!wrap)} said="Wrap" />
            <Switch
              on={space === "ignore"}
              flip={() => setSpace(space === "ignore" ? "keep" : "ignore")}
              said="Ignore spacing"
            />
            <button
              onClick={() =>
                setShut(Object.fromEntries(files.map((f) => [f.name, !allShut])))
              }
              className="rounded px-2 py-0.5 text-[11px] text-aside transition-colors hover:text-said"
            >
              {allShut ? "Expand all" : "Collapse all"}
            </button>
          </div>
        </div>

        {files.map((file) => (
          <FileDiff
            key={file.name}
            file={file}
            task={task}
            split={split && desk}
            wrap={wrap}
            shut={shut[file.name] ?? false}
            toggle={() => setShut((was) => ({ ...was, [file.name]: !was[file.name] }))}
            viewed={viewed[file.name] ?? false}
            see={(yes) => {
              setViewed((was) => ({ ...was, [file.name]: yes }));
              setShut((was) => ({ ...was, [file.name]: yes }));
            }}
          />
        ))}
      </div>
    </div>
  );
}

// Pick is one choice of two, as a segmented control.
function Pick<T extends string | boolean>({
  at,
  set,
  of,
}: {
  at: T;
  set: (one: T) => void;
  of: [T, string][];
}) {
  return (
    <div className="flex items-center gap-0.5 rounded-md bg-well p-0.5">
      {of.map(([id, name]) => (
        <button
          key={String(id)}
          onClick={() => set(id)}
          className={`rounded px-2 py-0.5 text-[11px] transition-colors ${
            at === id ? "bg-panel text-said" : "text-aside hover:text-said"
          }`}
        >
          {name}
        </button>
      ))}
    </div>
  );
}

// Switch is a setting that is on or off, said in its own word rather than in
// a symbol nobody can read back.
function Switch({ on, flip, said }: { on: boolean; flip: () => void; said: string }) {
  return (
    <button
      onClick={flip}
      aria-pressed={on}
      className={`rounded px-2 py-0.5 text-[11px] transition-colors ${
        on ? "bg-accent/15 text-accent" : "text-aside hover:text-said"
      }`}
    >
      {said}
    </button>
  );
}
