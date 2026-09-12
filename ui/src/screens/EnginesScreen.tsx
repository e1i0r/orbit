// The engines this build knows, and what this machine can do with them.
//
// Availability is the first thing a row says, because it is the one fact
// here about this machine rather than about the catalogue: an engine that is
// not installed shows the steps to install it in place of its dials, which
// is what somebody looking at this screen is about to need.
//
// The quota is drawn beside the name because this is the screen where the
// choice between engines is made, and "which one has room left" is most of
// that choice.

import { useEffect, useState } from "react";
import { api, type EngineInfo } from "../api";
import { Empty } from "../parts/Empty";

export function EnginesScreen() {
  const [engines, setEngines] = useState<EngineInfo[]>();
  const [settled, setSettled] = useState<string>();
  const [failed, setFailed] = useState<string>();

  useEffect(() => {
    let stale = false;

    api
      .engines()
      .then((got) => {
        if (stale) return;
        setEngines(got.engines);
        setSettled(got.settled);
      })
      .catch((e: Error) => !stale && setFailed(e.message));

    return () => {
      stale = true;
    };
  }, []);

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!engines) return <p className="text-xs text-aside">Reading the engines…</p>;

  if (engines.length === 0) {
    return (
      <Empty
        said="No engine answers"
        next="This build knows four. If this is empty, the catalogue is not being read."
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {engines.map((engine) => (
        <section key={engine.name} className="rounded-md border border-edge bg-panel">
          <header className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2">
            <span
              className={`size-1.5 shrink-0 rounded-full ${engine.available ? "bg-ok" : "bg-edge"}`}
              aria-hidden
            />
            <h2 className="font-mono text-xs text-said">{engine.name}</h2>
            {engine.name === settled && (
              <span className="rounded bg-accent/15 px-1.5 py-px text-[10px] text-accent">
                used when nothing says otherwise
              </span>
            )}
            <span className="text-[10px] text-faint">
              {engine.available ? "installed" : "not on this machine"}
              {engine.canThink ? " · can show its thinking" : ""}
              {engine.money ? " · billed" : ""}
            </span>

            {(engine.quota ?? []).length > 0 && (
              <span className="ml-auto flex flex-wrap items-center gap-2">
                {(engine.quota ?? []).map((w) => (
                  <span key={w.label} className="flex items-center gap-1.5" title={w.label}>
                    <span className="font-mono text-[10px] text-faint">{w.label}</span>
                    <span className="h-[3px] w-14 overflow-hidden rounded-full bg-well">
                      <span
                        className={`block h-full rounded-full ${w.pct > 85 ? "bg-bad" : w.pct > 60 ? "bg-wait" : "bg-ok"}`}
                        style={{ width: `${Math.min(100, Math.max(2, w.pct))}%` }}
                      />
                    </span>
                    <span className="font-mono text-[10px] text-aside tabular-nums">
                      {Math.round(w.pct)}%
                    </span>
                    {w.resetsIn > 0 && (
                      <span className="text-[10px] text-faint">back in {rest(w.resetsIn)}</span>
                    )}
                  </span>
                ))}
              </span>
            )}
          </header>

          {engine.available ? (
            <div className="flex flex-col gap-2 border-t border-edge px-3 py-2">
              <Dial name="Models" of={engine.models} />
              <Dial name="Efforts" of={engine.efforts} />
              {(engine.quota ?? []).length > 0 && !engine.sourced && (
                <p className="text-[10px] text-faint">
                  The quota above is Orbit's own count, not the engine's — it has not been asked.
                </p>
              )}
            </div>
          ) : (
            <div className="border-t border-edge px-3 py-2">
              <p className="mb-1 text-[11px] text-faint">
                To make it available
              </p>
              <ol className="flex flex-col gap-0.5">
                {(engine.setup ?? ["Install it and put it on PATH."]).map((step, i) => (
                  <li key={i} className="font-mono text-[11px] text-aside">
                    {step}
                  </li>
                ))}
              </ol>
            </div>
          )}
        </section>
      ))}
    </div>
  );
}

// Dial is one knob and everything it can be turned to. Long lists are cut,
// because forty model names is a wall rather than a choice.
function Dial({ name, of }: { name: string; of?: string[] }) {
  const [all, setAll] = useState(false);
  const list = of ?? [];

  if (list.length === 0) return null;

  const shown = all ? list : list.slice(0, 8);

  return (
    <div className="grid gap-1 sm:grid-cols-[80px_1fr] sm:gap-3">
      <span className="text-[11px] text-faint">{name}</span>
      <p className="flex flex-wrap gap-1">
        {shown.map((one) => (
          <span key={one} className="rounded bg-well px-1.5 py-px font-mono text-[10px] text-aside">
            {one}
          </span>
        ))}
        {list.length > 8 && (
          <button
            onClick={() => setAll(!all)}
            className="px-1 text-[10px] text-accent hover:text-said"
          >
            {all ? "fewer" : `${list.length - 8} more`}
          </button>
        )}
      </p>
    </div>
  );
}

// rest is how long until a window comes back, in the largest unit that is
// still worth acting on.
function rest(secs: number): string {
  if (secs < 60) return `${secs}s`;

  const mins = Math.round(secs / 60);
  if (mins < 60) return `${mins}m`;

  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ${mins % 60}m`;

  return `${Math.floor(hours / 24)}d ${hours % 24}h`;
}
