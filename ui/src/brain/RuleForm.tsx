// Writing a rule down, with the options in front of you.
//
// Offered and not remembered. Where a rule applies used to be an empty line
// you typed a path into, so filing one meant knowing by heart which folders
// the checkout has and spelling one right; the check was an empty line too,
// and a rule that blocks the work is worth nothing without one. Both are
// picked from what is actually there, and typing is what you do when none of
// them is it.

import { useState } from "react";
import type { Checkout, Rule, Unanswered } from "../api";

/** What the form is filling in, whichever gesture opened it. */
export interface Draft {
  /** rule is the one being corrected, and empty for a new one or for a
   *  sentence being kept out of the tray. */
  rule?: Rule;
  said?: Unanswered;
  phrase: string;
  repo: string;
  where: string;
  check: string;
}

export function RuleForm({
  draft,
  repos,
  busy,
  refused,
  onSave,
  onDrop,
  onBack,
}: {
  draft: Draft;
  repos: Checkout[];
  busy?: boolean;
  refused?: string;
  onSave: (one: Draft) => void;
  onDrop?: () => void;
  onBack: () => void;
}) {
  const [one, setOne] = useState(draft);

  const here = repos.find((r) => r.path === one.repo);
  const title = one.rule
    ? `Correcting ${one.rule.id}`
    : one.said
      ? "Keeping what you said"
      : "A new rule";

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        onSave(one);
      }}
    >
      <header className="flex flex-col gap-1">
        <button type="button" onClick={onBack} className="self-start text-xs text-faint hover:text-aside">
          ← back to the list
        </button>
        <h1 className="text-lg">
          {title}{" "}
          <span className="text-sm text-faint">
            {one.rule
              ? "it keeps the name it has had all along"
              : "a sentence about your code, told to every run before it works"}
          </span>
        </h1>
      </header>

      <Group label="THE RULE · what it says, and what it is about">
        <Row label="What it says">
          <textarea
            value={one.phrase}
            onChange={(e) => setOne({ ...one, phrase: e.target.value })}
            rows={2}
            placeholder="write the rule here"
            className="w-full rounded-md border border-edge bg-panel px-2 py-1.5 text-sm"
          />
        </Row>

        {repos.length > 1 && (
          <Row
            label="Which repository"
            hint="the rule is written inside this checkout and travels with it, so whoever clones the project gets it"
          >
            <Choice
              options={[
                ...repos.map((r) => ({ value: r.path, label: r.name })),
                { value: "", label: "none of them", note: "about every project on this machine" },
              ]}
              value={one.repo}
              pick={(value) => setOne({ ...one, repo: value, where: "" })}
            />
          </Row>
        )}

        <Row
          label="Where it applies"
          hint={
            one.where && one.where !== "."
              ? `it is only told when the work is inside ${one.where}`
              : "every run against this checkout is told it, whatever it is touching"
          }
        >
          <Choice
            options={[
              // A dot is how the whole checkout is said out loud, and it is
              // what widens a rule that was narrowed. Empty is the rule
              // staying where it is, which is a different answer.
              { value: ".", label: here ? `all of ${here.name}` : "every repo" },
              ...(here?.folders ?? []).map((d) => ({ value: d, label: `${d}/` })),
            ]}
            value={one.where}
            pick={(value) => setOne({ ...one, where: value })}
            typed={(value) => setOne({ ...one, where: value })}
          />
        </Row>
      </Group>

      <Group label="THE GATE · whether it also blocks the work, or only says it">
        <Row
          label="What it does"
          hint="every rule is put in front of the agent before it works. This one also runs a command at the gate, and blocks the work when that command fails."
        >
          <Choice
            options={[
              { value: "", label: "say", note: "the agent is told it, and the work goes ahead" },
              {
                value: "block",
                label: "block",
                note: "the same, and a command at the gate blocks the work when it fails",
              },
            ]}
            value={one.check ? "block" : ""}
            pick={(value) => setOne({ ...one, check: value ? one.check || here?.checks[0] || "" : "" })}
          />
        </Row>

        {one.check !== "" && (
          <Row
            label="The check"
            hint="the work is blocked when this command does not exit zero. It is yours and never a model's: it runs on every future phase in this repository."
          >
            <Choice
              options={(here?.checks ?? []).map((c) => ({ value: c, label: c }))}
              value={one.check}
              pick={(value) => setOne({ ...one, check: value })}
              typed={(value) => setOne({ ...one, check: value })}
            />
          </Row>
        )}
      </Group>

      {refused && <p className="text-sm text-bad">{refused}</p>}

      <footer className="flex flex-wrap gap-2 border-t border-edge pt-3">
        <button
          type="submit"
          disabled={busy}
          className="rounded-md bg-ok/20 px-3 py-1.5 text-sm text-ok hover:bg-ok/30 disabled:opacity-50"
        >
          {busy ? "…" : one.rule ? "Edit" : "Save"}
        </button>
        {onDrop && (
          <button
            type="button"
            onClick={onDrop}
            disabled={busy}
            className="rounded-md border border-edge px-3 py-1.5 text-sm hover:bg-edge/60 disabled:opacity-50"
          >
            Not a rule
          </button>
        )}
        <button
          type="button"
          onClick={onBack}
          className="rounded-md border border-edge px-3 py-1.5 text-sm hover:bg-edge/60"
        >
          Cancel
        </button>
      </footer>
    </form>
  );
}

function Group({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <header className="flex items-center gap-2">
        <h2 className="text-[11px] font-semibold uppercase tracking-wide text-live">{label}</h2>
        <span className="h-px flex-1 bg-edge" />
      </header>
      {children}
    </section>
  );
}

function Row({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid gap-1 sm:grid-cols-[11rem_1fr] sm:gap-3">
      <label className="pt-1 text-sm text-faint">{label}</label>
      <div className="flex flex-col gap-1">
        {children}
        {hint && <p className="text-xs text-live">{hint}</p>}
      </div>
    </div>
  );
}

/** One row's answers, with the one in force lit and the others beside it.
 *  A row that also takes a path or a command takes it typed, because what is
 *  offered is what somebody would have typed nine times out of ten. */
function Choice({
  options,
  value,
  pick,
  typed,
}: {
  options: { value: string; label: string; note?: string }[];
  value: string;
  pick: (value: string) => void;
  typed?: (value: string) => void;
}) {
  const known = options.some((o) => o.value === value);

  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-wrap gap-1">
        {options.map((o) => (
          <button
            key={o.value || "none"}
            type="button"
            title={o.note}
            onClick={() => pick(o.value)}
            className={`rounded-full px-2 py-0.5 text-xs ${
              o.value === value ? "bg-accent/20 text-accent" : "bg-edge/60 text-aside hover:bg-edge"
            }`}
          >
            {o.label}
          </button>
        ))}
      </div>
      {typed && (
        <input
          value={known ? "" : value}
          onChange={(e) => typed(e.target.value)}
          placeholder="or type another"
          className="mono w-full rounded-md border border-edge bg-panel px-2 py-1 text-xs"
        />
      )}
    </div>
  );
}
