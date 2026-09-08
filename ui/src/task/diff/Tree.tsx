// The files in the change, as a list you can jump from.
//
// A diff of thirty files read top to bottom is a diff nobody finishes. The
// list is what makes it navigable: every file, in its directory, with what
// happened to it and how much — and the ones already read struck through, so
// the question "where was I" has an answer on the screen.

import { useMemo, useState } from "react";
import type { File } from "./parse";
import { anchorOf } from "./FileDiff";

const dots: Record<File["status"], string> = {
  added: "bg-ok",
  deleted: "bg-bad",
  renamed: "bg-accent",
  modified: "bg-wait",
};

export function Tree({
  files,
  viewed,
  jump,
}: {
  files: File[];
  viewed: Record<string, boolean>;
  jump: (name: string) => void;
}) {
  const [like, setLike] = useState("");

  const shown = useMemo(() => {
    const needle = like.trim().toLowerCase();

    return needle ? files.filter((f) => f.name.toLowerCase().includes(needle)) : files;
  }, [files, like]);

  const byDir = useMemo(() => {
    const out = new Map<string, File[]>();

    for (const file of shown) {
      const cut = file.name.lastIndexOf("/");
      const dir = cut < 0 ? "" : file.name.slice(0, cut);
      out.set(dir, [...(out.get(dir) ?? []), file]);
    }

    return [...out];
  }, [shown]);

  return (
    <aside className="sticky top-0 flex max-h-[calc(100vh-8rem)] w-56 shrink-0 flex-col gap-2 self-start overflow-hidden">
      <input
        value={like}
        onChange={(e) => setLike(e.target.value)}
        placeholder="Filter files"
        className="w-full rounded border border-edge bg-well px-2 py-1 text-[11px] text-said placeholder:text-faint"
      />

      <div className="min-h-0 flex-1 overflow-y-auto pr-1">
        {byDir.map(([dir, inIt]) => (
          <div key={dir} className="mb-2">
            {dir && (
              <p className="truncate py-0.5 font-mono text-[10px] text-faint" title={dir}>
                {dir}/
              </p>
            )}
            {inIt.map((file) => (
              <button
                key={file.name}
                onClick={() => jump(file.name)}
                className="flex w-full items-center gap-1.5 rounded px-1 py-0.5 text-left hover:bg-hover"
                title={file.name}
              >
                <span className={`size-1.5 shrink-0 rounded-full ${dots[file.status]}`} aria-hidden />
                <span
                  className={`min-w-0 flex-1 truncate font-mono text-[11px] ${
                    viewed[file.name] ? "text-faint line-through" : "text-aside"
                  }`}
                >
                  {base(file.name)}
                </span>
                <span className="shrink-0 font-mono text-[10px] text-ok tabular-nums">
                  +{file.added}
                </span>
                <span className="shrink-0 font-mono text-[10px] text-bad tabular-nums">
                  −{file.removed}
                </span>
              </button>
            ))}
          </div>
        ))}

        {shown.length === 0 && <p className="px-1 text-[11px] text-faint">No file matches that.</p>}
      </div>
    </aside>
  );
}

export { anchorOf };

function base(name: string): string {
  return name.slice(name.lastIndexOf("/") + 1);
}
