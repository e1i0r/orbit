// Writing a task down.
//
// It is the one thing the browser could not do at all, and without it the
// web is a window onto work that has to be started somewhere else. Somebody
// who does not want a terminal has to be able to begin here.
//
// So the form asks for what a person knows and nothing else: what the work
// is, where it happens, and how careful to be about it. The identifier is
// the one piece of Orbit's own vocabulary it cannot avoid — a task is filed
// under it, and every command and every line of the record names it — so it
// is asked for first and its rule is said out loud rather than enforced in
// silence.

import { useEffect, useState } from "react";
import { api, type Board, type FlowShape } from "../api";

export function WriteScreen({ board, open }: { board?: Board; open: (id: string) => void }) {
  const [id, setId] = useState("");
  const [text, setText] = useState("");
  const [repo, setRepo] = useState("");
  const [flow, setFlow] = useState("");
  const [start, setStart] = useState(false);
  const [flows, setFlows] = useState<FlowShape[]>([]);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string>();

  useEffect(() => {
    let stale = false;

    api
      .flows()
      .then((got) => !stale && setFlows(got.flows))
      .catch(() => undefined);

    return () => {
      stale = true;
    };
  }, []);

  // The one repository on the board is the one it is against; with more than
  // one, choosing for somebody is how work lands in the wrong project.
  const repos = board?.repos ?? [];

  useEffect(() => {
    if (repos.length === 1 && repo === "") setRepo(repos[0]!.path);
  }, [repos, repo]);

  const short = id.trim() === "" || text.trim() === "";

  const send = async () => {
    setBusy(true);
    setFailed(undefined);

    try {
      const wrote = await api.write({
        id: id.trim(),
        text: text.trim(),
        repo: repo || undefined,
        flow: flow || undefined,
        start,
      });
      // The verb answers what it acted on, which for a write is the task
      // it made. The id typed here is what was asked for; the one that came
      // back is what the store settled on.
      open(wrote.of?.[0] ?? id.trim());
    } catch (e) {
      setFailed((e as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex max-w-[720px] flex-col gap-4">
      <Field
        name="Identifier"
        said="How you and Orbit will both refer to it. A tracker's id if you use one, anything short and unique if you do not."
      >
        <input
          value={id}
          onChange={(e) => setId(e.target.value)}
          placeholder="PAY-14"
          autoFocus
          className="w-56 rounded border border-edge bg-well px-2 py-1 font-mono text-xs text-said placeholder:text-faint"
        />
      </Field>

      <Field
        name="What the work is"
        said="Written for whoever does it. What should be true when it is done, and anything they would otherwise have to guess."
      >
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={7}
          placeholder="The checkout endpoint should reject negative amounts, and say which field was wrong. There are tests beside it in pay/."
          className="w-full resize-y rounded border border-edge bg-well px-2 py-1.5 text-xs text-said placeholder:text-faint"
        />
      </Field>

      {repos.length > 1 && (
        <Field name="Where" said="Which checkout the work happens in.">
          <select
            value={repo}
            onChange={(e) => setRepo(e.target.value)}
            className="rounded border border-edge bg-well px-2 py-1 text-xs text-said"
          >
            <option value="">no repository yet</option>
            {repos.map((one) => (
              <option key={one.path} value={one.path}>
                {one.name}
              </option>
            ))}
          </select>
        </Field>
      )}

      <Field
        name="How carefully"
        said="The shape of the work: how many phases it walks and what has to pass before it goes on."
      >
        <select
          value={flow}
          onChange={(e) => setFlow(e.target.value)}
          className="rounded border border-edge bg-well px-2 py-1 text-xs text-said"
        >
          <option value="">the one you have set as default</option>
          {flows.map((one) => (
            <option key={one.name} value={one.name}>
              {one.name}
              {one.description ? ` — ${one.description.slice(0, 70)}` : ""}
            </option>
          ))}
        </select>
      </Field>

      <label className="flex cursor-pointer items-center gap-2 text-xs text-aside select-none">
        <input
          type="checkbox"
          checked={start}
          onChange={(e) => setStart(e.target.checked)}
          className="size-3 accent-accent"
        />
        Start it as soon as it is written — this runs an engine and spends money
      </label>

      {failed && (
        <p className="rounded-md border border-bad/30 bg-bad/5 px-3 py-2 text-xs text-bad">{failed}</p>
      )}

      <div className="flex items-center gap-2">
        <button
          onClick={() => void send()}
          disabled={short || busy}
          className="rounded border border-accent/40 bg-accent/10 px-3 py-1 text-xs text-accent transition-colors hover:bg-accent/20 disabled:opacity-40"
          title={short ? "It needs an identifier and something written in it" : undefined}
        >
          {busy ? "Writing…" : start ? "Write it down and start" : "Write it down"}
        </button>
        <span className="text-[10px] text-faint">
          Nothing runs until you say so{start ? " — and you have" : ""}.
        </span>
      </div>
    </div>
  );
}

// Field is one thing being asked for, with the sentence that says why.
//
// The sentence is not decoration: this form is the way in for somebody who
// picked the browser because the terminal was too much, and a bare label
// called "Flow" tells them nothing about what to choose.
function Field({
  name,
  said,
  children,
}: {
  name: string;
  said: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-xs font-medium text-said">{name}</label>
      <p className="max-w-[70ch] text-[11px] text-faint">{said}</p>
      <div className="mt-0.5">{children}</div>
    </div>
  );
}
