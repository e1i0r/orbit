// One task, opened. The tabs are the cockpit's twelve, in the cockpit's own
// order: somebody who reads a task in the terminal and then in a browser is
// reading the same task.

import { useCallback, useEffect, useState } from "react";
import { api, type Diff, type Flow, type Impact, type Task } from "../api";
import { Card } from "../parts/Card";
import { Empty } from "../parts/Empty";
import { Page } from "../shell/Page";
import { DiffView } from "../task/DiffView";
import { FlowView } from "../task/FlowView";
import { ImpactView } from "../task/ImpactView";
import { Overview } from "../task/Overview";
import { Artifacts, Cost, Gates, Notes, Refused, Report, Thinking } from "../task/Panes";
import { Timeline } from "../task/Timeline";
import { Verbs } from "../task/Verbs";

const tabs = [
  { id: "overview", name: "Overview" },
  { id: "flow", name: "Flow" },
  { id: "gates", name: "Gates" },
  { id: "cost", name: "Cost" },
  { id: "refused", name: "Refused" },
  { id: "timeline", name: "Timeline" },
  { id: "report", name: "Report" },
  { id: "artifacts", name: "Artifacts" },
  { id: "notes", name: "Notes" },
  { id: "diff", name: "Diff" },
  { id: "impact", name: "Impact" },
  { id: "thinking", name: "Thinking" },
];

export function TaskScreen({ id, back }: { id: string; back: () => void }) {
  const [task, setTask] = useState<Task>();
  const [diff, setDiff] = useState<Diff>();
  const [flow, setFlow] = useState<Flow>();
  const [impact, setImpact] = useState<Impact>();
  const [failed, setFailed] = useState<string>();
  const [at, setAt] = useState("overview");

  // read is the task on its own, asked for again after a verb: what a run
  // did about being paused or started is the record's to say, and this is
  // how the page hears it.
  const read = useCallback(() => {
    api
      .task(id)
      .then(setTask)
      .catch((e: Error) => setFailed(e.message));
  }, [id]);

  useEffect(() => {
    let stale = false;

    api
      .task(id)
      .then((t) => !stale && setTask(t))
      .catch((e: Error) => !stale && setFailed(e.message));

    api
      .diff(id)
      .then((d) => !stale && setDiff(d))
      .catch((e: Error) => !stale && setFailed(e.message));

    api
      .flow(id)
      .then((f) => !stale && setFlow(f))
      .catch((e: Error) => !stale && setFlow({ id, name: "", attempts: 0, failed: e.message }));

    // The history is the slow one — a git log over the whole repository —
    // and it is asked for with the rest rather than when the tab is opened,
    // so the tab is ready by the time somebody reaches it. A failed reading
    // is the pane's to say, not the page's: the other eleven tabs are fine.
    api
      .impact(id)
      .then((i) => !stale && setImpact(i))
      .catch((e: Error) => !stale && setImpact({ id, commits: 0, failed: e.message }));

    return () => {
      stale = true;
    };
  }, [id]);

  if (failed) {
    return (
      <Page title={id}>
        <Card label="What went wrong">
          <p className="text-sm text-bad">{failed}</p>
        </Card>
      </Page>
    );
  }

  return (
    <Page
      title={
        <span className="flex items-baseline gap-3">
          <button
            onClick={back}
            className="text-sm font-normal text-aside transition-colors hover:text-said"
          >
            ← Board
          </button>
          <span className="font-mono">{id}</span>
        </span>
      }
      said={task?.title}
      does={task && <Verbs task={task} again={read} />}
      tabs={tabs}
      at={at}
      go={setAt}
    >
      {at === "overview" && <Overview task={task} />}
      {at === "flow" && <FlowView flow={flow} />}
      {at === "gates" && <Gates task={task} />}
      {at === "cost" && <Cost task={task} />}
      {at === "refused" && <Refused task={task} />}
      {at === "timeline" && <Timeline entries={task?.entries ?? null} />}
      {at === "report" && <Report task={task} />}
      {at === "artifacts" && <Artifacts task={task} />}
      {at === "notes" && <Notes task={task} />}
      {at === "diff" && <DiffView diff={diff} task={id} />}
      {at === "impact" && <ImpactView impact={impact} />}
      {at === "thinking" && <Thinking task={task} />}
    </Page>
  );
}

export { Empty };
