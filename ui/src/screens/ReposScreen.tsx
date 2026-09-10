// The repositories Orbit is watching, and what is happening in each.
//
// Read off the board rather than off the disk, so this screen and the task
// list are one answer: a repository with three tasks needing a person says
// so here in the same numbers the board shows.

import { useEffect, useState } from "react";
import { api, type RepoDetail } from "../api";
import { Empty } from "../parts/Empty";

const bands: { id: "needs_you" | "running" | "todo" | "done"; name: string; tone: string }[] = [
  { id: "needs_you", name: "Needs you", tone: "text-wait" },
  { id: "running", name: "Running", tone: "text-live" },
  { id: "todo", name: "To do", tone: "text-faint" },
  { id: "done", name: "Done", tone: "text-ok" },
];

export function ReposScreen() {
  const [repos, setRepos] = useState<RepoDetail[]>();
  const [root, setRoot] = useState("");
  const [failed, setFailed] = useState<string>();
  const [busy, setBusy] = useState<string>();
  const [said, setSaid] = useState<{ text: string; bad?: boolean }>();
  const [into, setInto] = useState("");

  useEffect(() => {
    let stale = false;

    api
      .repos()
      .then((got) => {
        if (stale) return;
        setRepos(got.repos);
        setRoot(got.root);
      })
      .catch((e: Error) => !stale && setFailed(e.message));

    return () => {
      stale = true;
    };
  }, []);

  // Both answer with a sentence the verb wrote, and both leave it on
  // screen: "every run here is accounted for" is the answer to Reconcile,
  // and a button that said nothing would leave a reader pressing it again.
  const ran = async (verb: "reconcile" | "export", says: Record<string, string>) => {
    setBusy(verb);
    setSaid(undefined);

    try {
      const did = await api.did(verb, says);
      setSaid({ text: did.said });
    } catch (e) {
      setSaid({ text: (e as Error).message, bad: true });
    } finally {
      setBusy(undefined);
    }
  };

  const reconcile = () => ran("reconcile", {});
  const write = () => ran("export", { into: into.trim() });

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!repos) return <p className="text-xs text-aside">Looking for repositories…</p>;

  if (repos.length === 0) {
    return (
      <Empty
        said="No repository under this root"
        next={`Nothing under ${root} looks like a git checkout. Orbit watches the directory it was started in.`}
      />
    );
  }

  return (
    <div className="measure overflow-hidden rounded-md border border-edge">
      <table className="w-full border-collapse text-xs">
        <thead>
          <tr className="bg-panel text-[11px] text-faint">
            <th className="px-3 py-1.5 text-left font-semibold">Repository</th>
            <th className="px-3 py-1.5 text-left font-semibold">Where</th>
            <th className="w-20 px-3 py-1.5 text-right font-semibold">Tasks</th>
            {bands.map((band) => (
              <th key={band.id} className="w-28 px-3 py-1.5 text-right font-semibold whitespace-nowrap">
                {band.name}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {repos.map((repo) => (
            <tr key={repo.path} className="border-t border-edge/60">
              <td className="px-3 py-1.5 font-mono text-[11px] text-said">{repo.name}</td>
              <td className="max-w-0 truncate px-3 py-1.5 font-mono text-[10px] text-faint" title={repo.path}>
                {repo.path.startsWith(root) ? repo.path.slice(root.length + 1) || "." : repo.path}
              </td>
              <td className="px-3 py-1.5 text-right font-mono text-[11px] text-said tabular-nums">
                {repo.tasks || "—"}
              </td>
              {bands.map((band) => (
                <td
                  key={band.id}
                  className={`px-3 py-1.5 text-right font-mono text-[11px] tabular-nums ${
                    repo.bands[band.id] ? band.tone : "text-faint"
                  }`}
                >
                  {repo.bands[band.id] ?? "—"}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>

      {/* The two things that are about the record rather than about one
          task. The window reaches them through its command line; this is
          where a browser does. */}
      <div className="flex flex-wrap items-center gap-2 border-t border-edge bg-panel px-3 py-2">
        <button
          onClick={() => void reconcile()}
          disabled={busy !== undefined}
          title="Close the records of runs whose processes are gone"
          className="rounded border border-edge bg-well px-2 py-0.5 text-[11px] text-aside transition-colors hover:bg-hover hover:text-said disabled:opacity-40"
        >
          {busy === "reconcile" ? "…" : "Reconcile"}
        </button>

        <span className="h-4 w-px bg-edge" aria-hidden />

        <input
          value={into}
          onChange={(e) => setInto(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && void write()}
          placeholder="/a/directory/that/is/empty"
          className="w-60 rounded border border-edge bg-well px-2 py-0.5 font-mono text-[11px] text-said outline-none placeholder:text-faint focus:border-accent/50"
        />
        <button
          onClick={() => void write()}
          disabled={busy !== undefined || into.trim() === ""}
          title="Write the whole record out as JSON lines, one file per task"
          className="rounded border border-edge bg-well px-2 py-0.5 text-[11px] text-aside transition-colors hover:bg-hover hover:text-said disabled:opacity-40"
        >
          {busy === "export" ? "…" : "Export the record"}
        </button>

        {said && (
          <span className={`min-w-0 truncate text-[11px] ${said.bad ? "text-bad" : "text-aside"}`}>
            {said.text}
          </span>
        )}
      </div>

      <p className="border-t border-edge bg-panel px-3 py-1.5 text-[10px] text-faint">
        Watching <span className="font-mono text-aside">{root}</span> · open a task from the board
        to see what it did.
      </p>
    </div>
  );
}

export type { RepoDetail };
