// What Orbit has been told.
//
// Two things decide whether a fact is worth keeping, and the row leads with
// both: what it actually does when work reaches it, and how often it has
// been told. A fact that asked to stop and brought no command to check with
// only warns — it is said plainly here, because a rule that reads as a gate
// and is not one is the worst kind to have written down.

import { useEffect, useMemo, useState } from "react";
import { api, type Board, type Fact } from "../api";
import { Empty } from "../parts/Empty";

export function KnowledgeScreen({ board }: { board?: Board }) {
  const [facts, setFacts] = useState<Fact[]>();
  const [read, setRead] = useState(false);
  const [failed, setFailed] = useState<string>();
  const [like, setLike] = useState("");
  const [shown, setShown] = useState<"told" | "all">("told");
  const [wrote, setWrote] = useState("");
  const [where, setWhere] = useState("");
  const [saving, setSaving] = useState(false);
  const [refused, setRefused] = useState<string>();
  const [again, setAgain] = useState(0);

  useEffect(() => {
    let stale = false;

    api
      .knowledge()
      .then((got) => {
        if (stale) return;
        setFacts(got.facts);
        setRead(got.read);
      })
      .catch((e: Error) => !stale && setFailed(e.message));

    return () => {
      stale = true;
    };
  }, [again]);

  const list = useMemo(() => {
    const needle = like.trim().toLowerCase();

    return (facts ?? [])
      .filter((f) => (shown === "all" ? true : !f.off))
      .filter((f) => !needle || `${f.phrase} ${f.scope} ${f.ref}`.toLowerCase().includes(needle));
  }, [facts, like, shown]);

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!facts) return <p className="text-xs text-aside">Reading what Orbit knows…</p>;

  if (!read) {
    return (
      <Empty
        said="Nothing was read"
        next="This build has no store to ask, so it says nothing rather than pretending it found nothing."
      />
    );
  }

  const off = facts.filter((f) => f.off).length;
  const repos = board?.repos ?? [];

  const learn = async () => {
    const text = wrote.trim();
    if (text === "" || saving) return;

    setSaving(true);
    setRefused(undefined);

    try {
      await api.did("learn", { text, repo: where || repos[0]?.path });
      setWrote("");
      setAgain((n) => n + 1);
    } catch (e) {
      setRefused((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-2.5">
      {/* A fact is about a repository, and which one is asked for rather
          than guessed at wherever there is a choice: a rule filed under the
          wrong checkout is told to the wrong runs, and nothing says so.
          `orbit learn` refuses to pick for you for the same reason. */}
      <div className="flex flex-col gap-1 rounded-md border border-edge bg-well/40 p-2">
        <div className="flex flex-wrap items-end gap-1.5">
          <input
            value={wrote}
            onChange={(e) => setWrote(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && void learn()}
            placeholder="Amounts are in cents everywhere below internal/money."
            className="min-w-0 flex-1 rounded border border-edge bg-well px-2 py-1 text-[11px] text-said outline-none placeholder:text-faint focus:border-accent/50"
          />

          {repos.length > 1 && (
            <select
              value={where}
              onChange={(e) => setWhere(e.target.value)}
              className="rounded border border-edge bg-well px-1.5 py-1 text-[11px] text-said outline-none focus:border-accent/50"
            >
              <option value="">which repository?</option>
              {repos.map((r) => (
                <option key={r.path} value={r.path}>
                  {r.name}
                </option>
              ))}
            </select>
          )}

          <button
            onClick={() => void learn()}
            disabled={saving || wrote.trim() === "" || (repos.length > 1 && where === "")}
            className="shrink-0 rounded border border-accent/40 bg-accent/10 px-2 py-1 text-[11px] text-accent transition-colors hover:bg-accent/20 disabled:opacity-40"
          >
            {saving ? "…" : "Write it down"}
          </button>
        </div>

        {refused && <p className="text-[11px] text-bad">{refused}</p>}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2">
        <input
          value={like}
          onChange={(e) => setLike(e.target.value)}
          placeholder="Filter facts"
          className="w-56 rounded border border-edge bg-well px-2 py-1 text-[11px] text-said placeholder:text-faint"
        />
        <div className="flex items-center gap-2">
          <span className="text-[10px] text-faint">
            {list.length} of {facts.length}
            {off > 0 ? ` · ${off} switched off` : ""}
          </span>
          <button
            onClick={() => setShown(shown === "all" ? "told" : "all")}
            aria-pressed={shown === "all"}
            className={`rounded px-2 py-0.5 text-[11px] transition-colors ${
              shown === "all" ? "bg-accent/15 text-accent" : "text-aside hover:text-said"
            }`}
          >
            Show the ones switched off
          </button>
        </div>
      </div>

      {facts.length === 0 ? (
        <Empty
          said="Orbit has been told nothing yet"
          next="orbit learn writes a fact, and a gate or the supervisor can add one as it goes."
        />
      ) : list.length === 0 ? (
        // Filtered down to nothing is not the same as knowing nothing, and
        // an empty area under a count that says "0 of 1" is a screen that
        // looks broken.
        <p className="rounded-md border border-edge bg-panel px-3 py-2 text-[11px] text-aside">
          {like.trim()
            ? `Nothing here matches "${like.trim()}".`
            : "Every fact Orbit holds is switched off. Show them to see what they said."}
        </p>
      ) : (
        <ul className="flex flex-col gap-1.5">
          {list.map((fact, i) => (
            <li
              key={i}
              className={`rounded-md border border-edge bg-panel px-3 py-2 ${fact.off ? "opacity-50" : ""}`}
            >
              <div className="flex items-start gap-2">
                <span
                  className={`mt-0.5 shrink-0 rounded px-1.5 py-px text-[10px] ${
                    fact.action === "stops" ? "bg-bad/15 text-bad" : "bg-wait/15 text-wait"
                  }`}
                >
                  {fact.action}
                </span>
                <p className="min-w-0 flex-1 text-xs text-said">{fact.phrase}</p>
                <span className="shrink-0 font-mono text-[10px] text-faint tabular-nums">
                  told {fact.used}×
                </span>
              </div>

              <p className="mt-1 flex flex-wrap gap-x-3 text-[10px] text-faint">
                <span className="font-mono">{fact.scope}</span>
                <span>{fact.source}</span>
                {fact.ref && <span className="font-mono">{fact.ref}</span>}
                <span className="tabular-nums">{when(fact.at)}</span>
                {fact.off && <span className="text-wait">switched off</span>}
              </p>

              {fact.check && (
                <p className="mt-1 truncate font-mono text-[10px] text-aside" title={fact.check}>
                  <span className="text-faint">checks with </span>
                  {fact.check}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function when(at: string): string {
  const t = new Date(at);

  return Number.isNaN(t.getTime())
    ? "—"
    : t.toLocaleString([], { day: "numeric", month: "short", year: "numeric" });
}
