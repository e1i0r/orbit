// The rules, in bands that say what each one is doing.
//
// Bands and not a column of states. The question this screen is opened with
// is "is there anything here for me", and that is answered by where a row is
// rather than by a word inside it — which is how the board answers it, and
// how the cockpit's own Brain does.

import type { Rule, Unanswered } from "../api";
import { mark, named, states, tint, type State } from "./states";

export function RuleList({
  rules,
  waiting,
  onRule,
  onSaid,
}: {
  rules: Rule[];
  waiting: Unanswered[];
  onRule: (f: Rule) => void;
  onSaid: (one: Unanswered) => void;
}) {
  return (
    <div className="flex flex-col gap-5">
      {states.map((state) => {
        const held = rules.filter((f) => f.state === state);
        // Every sentence in the tray is a question, so they are all in the
        // first band, above the rules only waiting to be looked at again.
        const said = state === "waiting" ? waiting : [];

        if (held.length + said.length === 0) return null;

        return (
          <section key={state}>
            <Band state={state} count={held.length + said.length} />
            <div className="mt-1 flex flex-col">
              {said.map((one) => (
                <SaidRow key={one.at} one={one} open={() => onSaid(one)} />
              ))}
              {held.map((f) => (
                <RuleRow key={f.id || f.phrase} rule={f} open={() => onRule(f)} />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

// Band is one heading, with the rule out to the edge the cockpit draws.
function Band({ state, count }: { state: State; count: number }) {
  return (
    <header className="flex items-center gap-2">
      <h2 className={`text-xs font-semibold tracking-wide ${tint[state]}`}>
        {mark[state]} {named[state]}
      </h2>
      <span className={state === "waiting" ? "text-xs text-wait" : "text-xs text-faint"}>
        ({count})
      </span>
      <span className="h-px flex-1 bg-edge" />
    </header>
  );
}

// RuleRow is one rule: its name, its sentence, how far it reaches and since
// when. What it does and where it stands are the band it is under.
function RuleRow({ rule, open }: { rule: Rule; open: () => void }) {
  return (
    <button
      type="button"
      onClick={open}
      className="grid grid-cols-[6rem_1fr_auto] items-baseline gap-3 rounded-md px-2 py-1.5 text-left hover:bg-edge/40 sm:grid-cols-[6rem_1fr_11rem_6rem]"
    >
      <span className="mono text-xs text-faint">{rule.id || "—"}</span>
      <span className={rule.state === "off" ? "text-sm text-faint" : "text-sm"}>
        {rule.phrase}
      </span>
      <span className="mono hidden text-xs text-aside sm:block">{rule.scope}</span>
      <span className="mono hidden text-xs text-faint sm:block">{day(rule.at)}</span>
    </button>
  );
}

// SaidRow is a sentence nobody has answered, drawn in the same columns as a
// rule because it is one question away from being one.
function SaidRow({ one, open }: { one: Unanswered; open: () => void }) {
  return (
    <button
      type="button"
      onClick={open}
      className="grid grid-cols-[6rem_1fr_auto] items-baseline gap-3 rounded-md px-2 py-1.5 text-left hover:bg-edge/40 sm:grid-cols-[6rem_1fr_11rem_6rem]"
    >
      <span className="mono text-xs text-faint">—</span>
      <span className="text-sm">{one.text}</span>
      <span className="mono hidden text-xs text-aside sm:block">
        {one.where || one.from || "—"}
      </span>
      <span className="mono hidden text-xs text-faint sm:block">{day(one.at)}</span>
    </button>
  );
}

// day is the date a rule was written down, as the record has it and not as
// the reader's own clock reads it: two people reading one repository's rules
// should see the same column.
export function day(at: string) {
  if (!at) return "";

  const on = new Date(at);

  return Number.isNaN(on.getTime()) ? "" : on.toISOString().slice(0, 10);
}
