// Every flow a task can be started under.
//
// A flow is a shape of work, so the shape is what the row leads with: the
// phases in order, the loops as blocks, and under them what each phase runs
// on and has to satisfy. Where it came from is said in a word, because a
// file of yours hiding a built-in of the same name is the one thing here
// somebody can get wrong without noticing.

import { useEffect, useState } from "react";
import { api, type FlowShape } from "../api";
import { Empty } from "../parts/Empty";
import { Rail } from "../task/flow/Rail";

const origins: Record<FlowShape["origin"], { said: string; tone: string }> = {
  builtin: { said: "shipped with orbit", tone: "text-faint" },
  yours: { said: "yours", tone: "text-accent" },
  shadow: { said: "yours, hiding a built-in of the same name", tone: "text-wait" },
  unknown: { said: "unknown", tone: "text-faint" },
};

export function FlowsScreen() {
  const [flows, setFlows] = useState<FlowShape[]>();
  const [failed, setFailed] = useState<string>();

  useEffect(() => {
    let stale = false;

    api
      .flows()
      .then((got) => !stale && setFlows(got.flows))
      .catch((e: Error) => !stale && setFailed(e.message));

    return () => {
      stale = true;
    };
  }, []);

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!flows) return <p className="text-xs text-aside">Reading the flows…</p>;

  if (flows.length === 0) {
    return (
      <Empty
        said="No flow answers"
        next="Orbit ships five. If this is empty, the flow directory is not being read."
      />
    );
  }

  return (
    <div className="flex measure flex-col gap-2">
      {flows.map((flow) => (
        <section key={flow.name} className="rounded-md border border-edge bg-panel px-3 py-2.5">
          <header className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <h2 className="font-mono text-xs text-said">{flow.name}</h2>
            <span className={`text-[10px] ${origins[flow.origin].tone}`}>
              {origins[flow.origin].said}
            </span>
            <span className="text-[10px] text-faint">
              {(flow.phases ?? []).length}{" "}
              {(flow.phases ?? []).length === 1 ? "phase" : "phases"} · up to {flow.attempts}{" "}
              {flow.attempts === 1 ? "attempt" : "attempts"} each
              {flow.diffBudget ? ` · ${flow.diffBudget}-line budget` : ""}
            </span>
          </header>

          {flow.failed ? (
            <p className="mt-1.5 text-[11px] text-bad">{flow.failed}</p>
          ) : (
            <>
              {flow.description && (
                <p className="mt-1 text-[11px] text-aside">{flow.description}</p>
              )}

              <div className="mt-2.5 border-t border-edge pt-2.5">
                <Rail phases={flow.phases ?? []} />
              </div>
            </>
          )}
        </section>
      ))}
    </div>
  );
}
