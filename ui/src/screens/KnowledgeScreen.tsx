// The Brain: everything Orbit has learned about your code.
//
// Three places and one screen: the rules in bands that say what each is
// doing, one rule opened with everything about it and the decisions under
// that, and the form a rule is written in. It is the cockpit's own shape,
// because a reader who has used one should not have to learn the other.
//
// Every gesture here goes through internal/verb by name — `rules keep`,
// `rules pause`, `rules off` — which is the same door the command line and a
// model's tool call reach. There is no verb on this page that is only here.

import { useCallback, useEffect, useState } from "react";
import { api, type Checkout, type Rule, type Unanswered, type Verb } from "../api";
import { Empty } from "../parts/Empty";
import { RuleCard } from "../brain/RuleCard";
import { RuleForm, type Draft } from "../brain/RuleForm";
import { RuleList } from "../brain/RuleList";

type Open =
  | { at: "list" }
  | { at: "rule"; id: string }
  | { at: "form"; draft: Draft };

export function KnowledgeScreen() {
  const [rules, setRules] = useState<Rule[]>();
  const [waiting, setWaiting] = useState<Unanswered[]>([]);
  const [repos, setRepos] = useState<Checkout[]>([]);
  const [read, setRead] = useState(false);
  const [failed, setFailed] = useState<string>();
  const [open, setOpen] = useState<Open>({ at: "list" });
  const [busy, setBusy] = useState<string>();
  const [said, setSaid] = useState<string>();
  const [refused, setRefused] = useState<string>();

  const load = useCallback(async () => {
    try {
      const got = await api.knowledge();

      setRules(got.rules);
      setWaiting(got.waiting ?? []);
      setRepos(got.repos ?? []);
      setRead(got.read);
    } catch (e) {
      setFailed((e as Error).message);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // Every verb lands the same way: it says what it did, the store is read
  // again, and the sentence stays on screen — a button that answered nothing
  // is a button somebody presses twice.
  const ask = async (verb: Verb, says: Record<string, string | undefined>) => {
    setBusy(verb);
    setRefused(undefined);

    try {
      const did = await api.did(verb, says);

      setSaid(did.said);
      await load();

      return true;
    } catch (e) {
      setRefused((e as Error).message);

      return false;
    } finally {
      setBusy(undefined);
    }
  };

  if (failed) return <Empty said="What Orbit knows could not be read." next={failed} />;
  if (!rules) return <Empty said="Reading what Orbit knows…" />;

  if (!read) {
    return <Empty said="This build has no store to ask." next="Run orbit from a workspace." />;
  }

  if (open.at === "rule") {
    const one = rules.find((f) => f.id === open.id);

    if (!one) return <Empty said="That rule is no longer there." />;

    return (
      <RuleCard
        rule={one}
        busy={busy}
        said={said}
        onBack={() => setOpen({ at: "list" })}
        onDo={async (verb) => {
          if (await ask(verb, { rule: one.id })) setOpen({ at: "list" });
        }}
        onEdit={() =>
          setOpen({
            at: "form",
            draft: {
              rule: one,
              phrase: one.phrase,
              repo: one.repo ?? "",
              where: one.path || ".",
              check: one.check ?? "",
            },
          })
        }
      />
    );
  }

  if (open.at === "form") {
    const { draft } = open;

    return (
      <RuleForm
        draft={draft}
        repos={repos}
        busy={busy !== undefined}
        refused={refused}
        onBack={() => setOpen({ at: "list" })}
        onDrop={
          draft.said
            ? async () => {
                if (await ask("rules drop", { at: draft.said?.at })) setOpen({ at: "list" });
              }
            : undefined
        }
        onSave={async (one) => {
          // Correcting names the rule; keeping names the sentence. Both
          // take the same three fields, because both are answering the
          // same three questions about the same thing.
          const done = one.rule
            ? await ask("rules correct", {
                rule: one.rule.id,
                text: one.phrase,
                in: one.where,
                check: one.check,
              })
            : await ask("rules keep", {
                at: one.said?.at,
                text: one.phrase,
                repo: one.repo || undefined,
                in: one.where,
                check: one.check,
              });

          if (done) setOpen({ at: "list" });
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <p className="text-xs text-faint">
            Every rule is put in front of the agent before it works. The ones that block also run a
            command, and send the work back when it fails.
          </p>
        </div>
        <button
          type="button"
          onClick={() =>
            setOpen({
              at: "form",
              draft: { phrase: "", repo: repos[0]?.path ?? "", where: "", check: "" },
            })
          }
          className="rounded-md border border-edge px-3 py-1.5 text-sm hover:bg-edge/60"
        >
          + A new rule
        </button>
      </header>

      {said && <p className="text-sm text-ok">{said}</p>}

      {rules.length + waiting.length === 0 ? (
        <Empty
          said="Nothing written down yet."
          next="Say a rule to the supervisor, or write one here."
        />
      ) : (
        <RuleList
          rules={rules}
          waiting={waiting}
          onRule={(f) => {
            setSaid(undefined);
            setOpen({ at: "rule", id: f.id });
          }}
          onSaid={(one) =>
            setOpen({
              at: "form",
              draft: {
                said: one,
                phrase: one.text,
                repo: one.repo ?? repos[0]?.path ?? "",
                where: one.where ?? "",
                check: "",
              },
            })
          }
        />
      )}
    </div>
  );
}
