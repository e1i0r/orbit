// What this change reaches beyond the files it touched.
//
// Three claims of three different weights, and the sections are kept apart
// so a reader can tell which is which. The coupling is the repository's own
// history — what usually comes along, and did not this time. The contracts
// are sentences somebody wrote as test names. The last one is the engine's
// account of its own work, and nothing verified it.
//
// Each section says what it is before it says what it found: a list of
// filenames under a heading nobody understands is a list nobody acts on.

import { useState } from "react";
import { api, type Contract, type Coupled, type Did, type Impact, type Side } from "../api";
import { Empty } from "../parts/Empty";

export function ImpactView({ impact, task }: { impact?: Impact; task: string }) {
  if (!impact) return <p className="text-xs text-aside">Reading the history…</p>;

  if (impact.missing) {
    return (
      <Empty
        said="No checkout to weigh"
        next="A task has a worktree from the moment its first phase runs. Start it, and this reading lands here."
      />
    );
  }

  if (impact.failed) {
    return (
      <div className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2.5">
        <p className="text-xs text-bad">{impact.failed}</p>
      </div>
    );
  }

  const changed = impact.changed ?? [];
  const coupled = impact.coupled ?? [];
  const contracts = impact.contracts ?? [];
  const delta = impact.delta;

  return (
    <div className="flex flex-col gap-4">
      <p className="text-[11px] text-faint">
        <span className="font-mono text-said">{changed.length}</span>{" "}
        {changed.length === 1 ? "file" : "files"} changed · read against the last{" "}
        <span className="font-mono text-said">{impact.commits}</span> commits
      </p>

      <Section
        name="What usually comes along"
        about="Files this repository has committed together with the ones this task changed, and that it did not touch this time. It is what the history does, not a rule — a file left out on purpose is the normal case."
      >
        {coupled.length > 0 ? (
          <Coupling changed={changed} coupled={coupled} />
        ) : changed.length === 0 ? (
          <Nothing said="This task has changed no files yet, so there is nothing to follow." />
        ) : impact.commits < 20 ? (
          <Nothing
            said={`This repository has ${impact.commits} commits behind it. The history has not seen enough of these files together to say anything worth acting on.`}
          />
        ) : (
          <Nothing said="Nothing else follows these files often enough to mention." good />
        )}
      </Section>

      <Section
        name="What those tests say they hold"
        about="The names of the tests in the files above, read as sentences. Nothing here was run: it is what somebody wrote down that the code guarantees."
      >
        {contracts.length > 0 ? (
          <Contracts contracts={contracts} />
        ) : (
          <Nothing
            said={
              changed.length === 0
                ? "This task has changed no files yet."
                : "None of the files this task changed is a test, so there is no sentence here to read."
            }
          />
        )}
      </Section>

      <Section
        name="What the agent says it did"
        about="The engine's own account of what this change asks of its callers and what it now promises them. Nobody verified it — no command can — and the last part is the only place a rejected approach is written down."
      >
        {delta ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <Listed name="Callers must now" said={delta.needs} />
            <Listed name="It now holds" said={delta.guarantees} />
            <Listed name="It took for granted" said={delta.assumes} />
            <Listed name="Considered and not taken" said={delta.instead} />
          </div>
        ) : (
          <Nothing said="No phase of this task has written one. The engine writes it at the end of a phase that changed something, so it lands here on the next run." />
        )}
      </Section>

      <Section
        name="What the checks say on both sides"
        about="The flow's own gates, run once on the base and once on this work. It is the only part of this page that was run rather than read — which is why it is asked for rather than fetched: it checks out the base and runs somebody's test suite twice."
      >
        <Sides task={task} />
      </Section>
    </div>
  );
}

// Nothing is a finding that is not there, and why.
//
// Never an absent section. A claim that disappears when it has nothing to
// say is a claim the reader never learns exists, and these three are the
// most valuable things this page knows — a reader who has not seen the
// heading cannot go looking for the data that would fill it.
function Nothing({ said, good }: { said: string; good?: boolean }) {
  return <p className={`text-[11px] ${good ? "text-ok" : "text-faint"}`}>{said}</p>;
}

// Coupling groups what follows under the changed file it follows, because
// the pair is the claim: this file moved, that one usually moves with it.
function Coupling({ changed, coupled }: { changed: string[]; coupled: Coupled[] }) {
  const under = new Map<string, Coupled[]>();

  for (const one of coupled) {
    under.set(one.with, [...(under.get(one.with) ?? []), one]);
  }

  return (
    <ul className="flex flex-col gap-2.5">
      {changed
        .filter((file) => under.has(file))
        .map((file) => (
          <li key={file}>
            <p className="font-mono text-[11px] text-said">
              {file} <span className="text-faint">changed here</span>
            </p>
            <ul className="mt-1 flex flex-col gap-0.5">
              {(under.get(file) ?? []).map((one) => (
                <li key={one.file} className="flex items-center gap-2">
                  <span className="w-10 shrink-0 font-mono text-[10px] text-wait tabular-nums">
                    {Math.round(one.ratio * 100)}%
                  </span>
                  <span className="h-[3px] w-16 shrink-0 overflow-hidden rounded-full bg-well">
                    <span
                      className="block h-full rounded-full bg-wait/70"
                      style={{ width: `${Math.max(4, one.ratio * 100)}%` }}
                    />
                  </span>
                  <span className="min-w-0 truncate font-mono text-[11px] text-aside">
                    {one.file}
                  </span>
                  <span className="shrink-0 font-mono text-[10px] text-faint tabular-nums">
                    {one.times}/{one.of}
                  </span>
                </li>
              ))}
            </ul>
          </li>
        ))}
    </ul>
  );
}

function Contracts({ contracts }: { contracts: Contract[] }) {
  const under = new Map<string, string[]>();

  for (const one of contracts) {
    under.set(one.file, [...(under.get(one.file) ?? []), one.says]);
  }

  return (
    <ul className="flex flex-col gap-2">
      {[...under].map(([file, says]) => (
        <li key={file}>
          <p className="font-mono text-[11px] text-said">{file}</p>
          <ul className="mt-0.5 flex flex-col gap-0.5">
            {says.map((one, i) => (
              <li key={i} className="font-mono text-[11px] text-aside">
                <span className="text-faint">· </span>
                {one}
              </li>
            ))}
          </ul>
        </li>
      ))}
    </ul>
  );
}

// Section is one claim: its name, a rule, what it means, and then the
// finding. The rule is what says the three are separate things — a reader
// who cannot see where one ends reads the third with the weight of the
// first.
function Section({
  name,
  about,
  children,
}: {
  name: string;
  about: string;
  children: React.ReactNode;
}) {
  return (
    <section>
      <div className="flex items-center gap-2">
        <h2 className="shrink-0 text-[10px] font-semibold tracking-[0.09em] text-accent uppercase">
          {name}
        </h2>
        <span className="h-px flex-1 bg-edge" aria-hidden />
      </div>
      <p className="mt-1 max-w-[86ch] text-[11px] text-faint">{about}</p>
      <div className="mt-2.5">{children}</div>
    </section>
  );
}

function Listed({ name, said }: { name: string; said?: string[] }) {
  if (!said || said.length === 0) return null;

  return (
    <div>
      <h3 className="text-[10px] tracking-[0.09em] text-faint uppercase">{name}</h3>
      <ul className="mt-1 flex flex-col gap-0.5">
        {said.map((one, i) => (
          <li key={i} className="text-[11px] text-aside">
            {one}
          </li>
        ))}
      </ul>
    </div>
  );
}

// Sides is the one section with a button, because it is the one thing on
// this page that has to be run rather than read.
//
// It is not fetched with the rest. Checking out the base and running a test
// suite twice takes minutes and executes whatever the flow's gates say, and
// something that costs that much is asked for on purpose — never by a page
// that happened to be refreshed.
function Sides({ task }: { task: string }) {
  const [running, setRunning] = useState(false);
  const [got, setGot] = useState<Did>();
  const [failed, setFailed] = useState<string>();

  const run = () => {
    setRunning(true);
    setFailed(undefined);

    api
      .do(task, "compare")
      .then(setGot)
      .catch((e: Error) => setFailed(e.message))
      .finally(() => setRunning(false));
  };

  const sides = (got?.saw as Side[] | undefined) ?? [];

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <button
          onClick={run}
          disabled={running}
          className="rounded border border-accent/40 bg-accent/10 px-2 py-0.5 text-[11px] text-accent transition-colors hover:bg-accent/20 disabled:opacity-50"
        >
          {running ? "Running on both sides…" : got ? "Run again" : "Run the checks"}
        </button>

        {running && (
          <span className="text-[11px] text-faint">
            The base is checked out and each gate runs twice. This takes as long as your suite does.
          </span>
        )}
      </div>

      {failed && <p className="text-[11px] text-bad">{failed}</p>}

      {got && sides.length === 0 && <Nothing said={got.said} />}

      {sides.length > 0 && (
        <ul className="flex flex-col gap-1">
          {sides.map((one) => (
            <li key={one.name} className="flex items-baseline gap-2">
              <span
                className={`w-16 shrink-0 text-[10px] tracking-[0.06em] uppercase ${
                  one.broke ? "text-bad" : one.fixed ? "text-ok" : "text-faint"
                }`}
              >
                {one.broke ? "broke" : one.fixed ? "fixed" : one.base.failed || one.now.failed ? "no answer" : "same"}
              </span>
              <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-said">{one.name}</span>
              <span className="shrink-0 font-mono text-[10px] text-faint tabular-nums">
                {one.base.failed || one.now.failed ? "—" : `${one.base.exit} → ${one.now.exit}`}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
