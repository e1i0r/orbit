// The repository, and where this task went in it.
//
// A honeycomb of one level on the left and what is under the chosen cell on
// the right. The two answer different questions and neither replaces the
// other: the comb says *where* — a shape lit where the work happened, read
// in one look — and the panel says *what*, down to the file, where the diff
// that already exists takes over.
//
// The panel opens on what changed rather than on everything. A directory of
// four hundred files listed whole is a list nobody reads, and the question
// somebody has when they open a map of their own repository is not "what is
// in here", it is "what did it touch".

import { useEffect, useMemo, useState } from "react";
import { ChevronRight, CornerDownLeft, FileText, Folder } from "lucide-react";
import { api, type Cell, type Diff } from "../../api";
import { Empty } from "../../parts/Empty";
import { parse } from "../diff/parse";
import { FileDiff } from "../diff/FileDiff";
import { Honeycomb } from "./Honeycomb";

export function MapView({ task }: { task: string }) {
  const [root, setRoot] = useState<Cell>();
  const [failed, setFailed] = useState<string>();
  const [diff, setDiff] = useState<Diff>();
  // path is the level being shown, and "" is the root of the checkout.
  const [path, setPath] = useState("");
  // Chosen by path rather than by cell, so that a choice survives going
  // into a level and coming back out — which is what "several, at any
  // level" means. A cell is a reading of one moment; a path is the thing.
  const [chosen, setChosen] = useState<string[]>([]);
  const [file, setFile] = useState<string>();
  const [all, setAll] = useState(false);

  useEffect(() => {
    let stale = false;

    api
      .tree(task)
      .then((t) => !stale && setRoot(t.saw))
      .catch((e: Error) => !stale && setFailed(e.message));

    // The diff is fetched beside the tree rather than when a file is
    // opened: it is one request for the whole change, and asking per file
    // would be a wait in the middle of a gesture that should feel like
    // opening a folder.
    api
      .diff(task)
      .then((d) => !stale && setDiff(d))
      .catch(() => undefined);

    return () => {
      stale = true;
    };
  }, [task]);

  const here = useMemo(() => (root ? at(root, path) : undefined), [root, path]);
  const under = useMemo(
    () => (root ? chosen.map((p) => at(root, p)).filter((c): c is Cell => !!c) : []),
    [root, chosen],
  );
  const files = useMemo(() => (diff?.text ? parse(diff.text) : []), [diff]);

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!root) return <p className="text-xs text-aside">Reading the repository…</p>;

  if (!here || (here.cells ?? []).length === 0) {
    return (
      <Empty
        said="No checkout to map"
        next="A task has a worktree from the moment its first phase runs. Start it, and the repository lands here."
      />
    );
  }

  const open = file ? files.find((f) => f.name === file) : undefined;
  const shown = under.length > 0 ? under : here ? [here] : [];

  return (
    <div className="flex flex-col gap-2.5">
      <Crumbs
        path={path}
        changed={root.changed ?? 0}
        deep={depth(root)}
        level={path === "" ? 0 : path.split("/").length}
        go={(to) => {
          setPath(to);
          setFile(undefined);
        }}
      />

      <div className="grid gap-2.5 lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
        <div className="rounded-md border border-edge bg-well/40 px-2 py-2">
          <Honeycomb
            cells={here.cells ?? []}
            picked={chosen}
            onPick={(c, also) => {
              setChosen((was) =>
                also
                  ? was.includes(c.path)
                    ? was.filter((p) => p !== c.path)
                    : [...was, c.path]
                  : was.length === 1 && was[0] === c.path
                    ? []
                    : [c.path],
              );

              // A file has nothing under it, so choosing one is asking for
              // its diff. A directory is asking what is inside.
              setFile((c.cells?.length ?? 0) === 0 ? c.path : undefined);
            }}
            onEnter={(c) => {
              setPath(c.path);
              setFile(undefined);
            }}
          />
        </div>

        <div className="min-w-0 rounded-md border border-edge bg-well/40 p-2">
          <div className="mb-1.5 flex items-baseline justify-between gap-2">
            <p className="truncate text-[11px] text-said">
              {under.length > 1
                ? `${under.length} chosen`
                : shown[0]?.path || "the whole checkout"}
            </p>

            <div className="flex shrink-0 items-baseline gap-2">
              {under.length > 0 && (
                <button
                  onClick={() => setChosen([])}
                  className="text-[10px] text-faint transition-colors hover:text-accent"
                >
                  clear
                </button>
              )}
              <button
                onClick={() => setAll(!all)}
                className="text-[10px] text-faint transition-colors hover:text-accent"
              >
                {all ? "only changed" : "everything"}
              </button>
            </div>
          </div>

          {/* Each choice keeps its own heading, so two branches from
              different corners of the repository do not read as one. */}
          <div className="flex flex-col gap-1.5">
            {shown.map((cell) =>
              (cell.cells?.length ?? 0) === 0 ? (
                // A chosen file has nothing under it to list. Saying "this
                // task changed nothing under here" about a file it changed
                // is the panel answering a question nobody asked.
                <button
                  key={cell.path}
                  onClick={() => setFile(cell.path)}
                  className={`flex w-full items-center gap-1.5 rounded px-1 py-0.5 text-left text-[11px] transition-colors ${
                    file === cell.path ? "bg-accent/15 text-accent" : "text-said hover:bg-hover"
                  }`}
                >
                  <FileText className="size-3 shrink-0 opacity-70" />
                  <span className="truncate">{cell.path}</span>
                  {(cell.lines ?? 0) > 0 && (
                    <span className="ml-auto shrink-0 tabular-nums text-[10px] text-faint">{cell.lines}</span>
                  )}
                </button>
              ) : (
                <div key={cell.path || "root"}>
                  {under.length > 1 && (
                    <p className="truncate px-1 pb-0.5 text-[10px] text-faint">{cell.path}</p>
                  )}
                  <Branch cell={cell} all={all} on={file} pick={setFile} />
                </div>
              ),
            )}
          </div>
        </div>
      </div>

      {open && (
        <FileDiff
          file={open}
          task={task}
          split={false}
          wrap={false}
          shut={false}
          toggle={() => undefined}
          viewed={false}
          see={() => undefined}
        />
      )}

      {file && !open && (
        <p className="px-1 text-[11px] text-faint">
          <span className="text-aside">{file}</span> is in the repository and not in this change.
        </p>
      )}
    </div>
  );
}

// Crumbs is where the reader is: the path back to the root, and the two
// numbers that keep a zoom from disorienting — how deep the repository goes,
// and how much of the change is in it at all.
function Crumbs({
  path,
  changed,
  deep,
  level,
  go,
}: {
  path: string;
  changed: number;
  deep: number;
  level: number;
  go: (to: string) => void;
}) {
  const parts = path === "" ? [] : path.split("/");

  return (
    <div className="flex flex-wrap items-center gap-x-1 gap-y-0.5 text-[11px]">
      <button onClick={() => go("")} className="text-aside transition-colors hover:text-accent">
        root
      </button>

      {parts.map((name, i) => (
        <span key={i} className="flex items-center gap-1">
          <ChevronRight className="size-3 text-faint" />
          <button
            onClick={() => go(parts.slice(0, i + 1).join("/"))}
            className={i === parts.length - 1 ? "text-said" : "text-aside transition-colors hover:text-accent"}
          >
            {name}
          </button>
        </span>
      ))}

      <span className="ml-auto flex items-baseline gap-2 text-faint">
        <span className="hidden sm:inline">cmd-click for several</span>
        <span>
          level {level} of {deep} · {changed} file{changed === 1 ? "" : "s"} changed
        </span>
      </span>
    </div>
  );
}

// Branch is what is under one cell, all the way to the files.
//
// This is the fast path the comb exists to feed: one click on a cell of the
// root and the whole tree beneath it is here, ending in files that open
// their own diff — from the top of the repository to the lines without
// walking a level at a time.
function Branch({
  cell,
  all,
  on,
  pick,
  depth = 0,
}: {
  cell: Cell;
  all: boolean;
  on?: string;
  pick: (path: string) => void;
  depth?: number;
}) {
  const cells = (cell.cells ?? []).filter((c) => all || (c.changed ?? 0) > 0);

  if (cells.length === 0) {
    if (depth === 0) {
      return (
        <p className="px-1 py-2 text-[11px] text-faint">
          {all ? "Nothing in here." : "This task changed nothing under here."}
        </p>
      );
    }

    return null;
  }

  return (
    <ul className={depth === 0 ? "flex flex-col" : "flex flex-col border-l border-edge/60 pl-2"}>
      {cells.map((c) => {
        const leaf = (c.cells?.length ?? 0) === 0;
        const lit = (c.changed ?? 0) > 0;

        return (
          <li key={c.path}>
            <button
              onClick={() => leaf && pick(c.path)}
              className={`flex w-full items-center gap-1.5 rounded px-1 py-0.5 text-left text-[11px] transition-colors ${
                on === c.path ? "bg-accent/15 text-accent" : leaf ? "hover:bg-hover" : ""
              } ${lit ? "text-said" : "text-faint"}`}
            >
              {leaf ? (
                <FileText className="size-3 shrink-0 opacity-70" />
              ) : (
                <Folder className="size-3 shrink-0 opacity-70" />
              )}
              <span className="truncate">{c.name}</span>
              {lit && (
                <span className="ml-auto shrink-0 tabular-nums text-[10px] text-faint">
                  {leaf ? c.lines : `${c.changed} · ${c.lines}`}
                </span>
              )}
              {leaf && on === c.path && <CornerDownLeft className="size-3 shrink-0" />}
            </button>

            {!leaf && <Branch cell={c} all={all} on={on} pick={pick} depth={depth + 1} />}
          </li>
        );
      })}
    </ul>
  );
}

/** at is the cell one path names, and the root for an empty path. */
function at(root: Cell, path: string): Cell | undefined {
  if (path === "") return root;

  let here: Cell | undefined = root;

  for (const name of path.split("/")) {
    here = here?.cells?.find((c) => c.name === name);
    if (!here) return undefined;
  }

  return here;
}

/** depth is how many levels the deepest branch of the tree goes down. */
function depth(cell: Cell): number {
  const cells = cell.cells ?? [];
  if (cells.length === 0) return 0;

  return 1 + Math.max(...cells.map(depth));
}
