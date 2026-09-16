// One rule, opened: everything about it, and the decisions under that.
//
// The evidence first and the decisions under it, because the whole point of
// this screen is that nothing is decided before it is read. There is no
// score: a rule that works perfectly never blocks anything, so "it blocked
// the work zero times" means two opposite things and no number tells them
// apart. What is written down is the friction.

import { useEffect, useState } from "react";
import { api, type Rule } from "../api";
import { Pill } from "../parts/Pill";
import { day } from "./RuleList";
import { from, mark, named, tint, tone } from "./states";

export function RuleCard({
  rule,
  busy,
  said,
  onDo,
  onEdit,
  onBack,
}: {
  rule: Rule;
  busy?: string;
  said?: string;
  onDo: (verb: "rules resume" | "rules off") => void;
  onEdit: () => void;
  onBack: () => void;
}) {
  const [story, setStory] = useState<string>();

  // What it has put somebody through, read by name off the same verb the
  // command line asks: a second answer written for this page would be a
  // second opinion about what happened.
  useEffect(() => {
    let stale = false;

    if (!rule.id) return;

    api
      .read<unknown>(`rules history?rule=${encodeURIComponent(rule.id)}`)
      .then((got) => !stale && setStory(got.said))
      .catch(() => !stale && setStory(""));

    return () => {
      stale = true;
    };
  }, [rule.id]);

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <button type="button" onClick={onBack} className="self-start text-xs text-faint hover:text-aside">
          ← back to the list
        </button>
        <h1 className="text-lg">{rule.phrase}</h1>
        <p className="mono flex flex-wrap items-center gap-2 text-xs text-faint">
          <span className="text-accent">{rule.id || "no name yet"}</span>
          <span>·</span>
          <span>{rule.scope}</span>
          <span>·</span>
          <span className={tint[rule.state]}>
            {mark[rule.state]} {named[rule.state]}
          </span>
        </p>
      </header>

      <dl className="grid grid-cols-2 gap-2 sm:grid-cols-5">
        <Figure label="status">
          <Pill tone={tone[rule.state]}>{named[rule.state]}</Pill>
        </Figure>
        <Figure label="created by">{from[rule.source] ?? rule.source}</Figure>
        <Figure label="created">{day(rule.at) || "—"}</Figure>
        <Figure label="hits">{rule.used}</Figure>
        <Figure label="reach">{rule.repo ? "with the repo" : "this machine"}</Figure>
      </dl>

      <Section label="what it does">
        {rule.check ? (
          <>
            <span className="text-bad">it blocks the work</span>
            <span className="text-faint"> · the check is </span>
            <span className="mono">{rule.check}</span>
          </>
        ) : rule.stops ? (
          <span>
            it was asked to block the work and has no command to block it with, so it only says
            its sentence. Give it one with Edit.
          </span>
        ) : (
          <span>it is put in front of the agent before every run, and the work goes ahead</span>
        )}
      </Section>

      {rule.state !== "says" && rule.state !== "blocks" && (
        <Section label="why it is not applying">
          {rule.why || "you decided against it. It stays where it is, and nothing is told it."}
        </Section>
      )}

      <Section label="what it has put you through">
        <pre className="mono whitespace-pre-wrap text-xs leading-relaxed">
          {story === undefined ? "reading…" : story || "nothing has happened to this one yet"}
        </pre>
      </Section>

      {said && <p className="text-sm text-ok">{said}</p>}

      <footer className="flex flex-wrap gap-2 border-t border-edge pt-3">
        <Act label="Turn on" busy={busy === "rules resume"} onClick={() => onDo("rules resume")} />
        <Act label="Switch off" busy={busy === "rules off"} onClick={() => onDo("rules off")} />
        <Act label="Edit" onClick={onEdit} />
      </footer>
    </div>
  );
}

function Figure({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-edge bg-panel px-3 py-2">
      <dt className="text-[11px] uppercase tracking-wide text-faint">{label}</dt>
      <dd className="mt-0.5 text-sm">{children}</dd>
    </div>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <section>
      <header className="flex items-center gap-2">
        <h2 className="text-[11px] font-semibold uppercase tracking-wide text-accent">{label}</h2>
        <span className="h-px flex-1 bg-edge" />
      </header>
      <div className="mt-1 pl-3 text-sm">{children}</div>
    </section>
  );
}

function Act({ label, busy, onClick }: { label: string; busy?: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={busy}
      className="rounded-md border border-edge px-3 py-1.5 text-sm hover:bg-edge/60 disabled:opacity-50"
    >
      {busy ? "…" : label}
    </button>
  );
}
