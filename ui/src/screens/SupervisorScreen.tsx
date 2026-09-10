// What has been said to the supervisor, and what it said back.
//
// A thread and not a table: this is a conversation, and the thing a reader
// wants is to follow it. The conversations are listed beside it because a
// thread that has been going for weeks is not one thread, and the record
// already knows where each one starts.
//
// Who said it is what the row leads with, because it is what changes how the
// line is read: a person asking and an engine answering look the same in a
// list of sentences.

import { useEffect, useMemo, useRef, useState } from "react";
import { Undo2 } from "lucide-react";
import { api, type Chat, type Said } from "../api";
import { Empty } from "../parts/Empty";

// The people. Anything else that speaks here is an engine answering, and it
// is named in the record by the engine's own name.
const people = new Set(["operator", "elio", "you", "human"]);

// earlier is the id given to the thread that has none. It cannot collide
// with a real one: those are timestamps.
const earlier = "\u0000earlier";

export function SupervisorScreen() {
  const [chats, setChats] = useState<Chat[]>();
  const [said, setSaid] = useState<Said[]>([]);
  const [read, setRead] = useState(false);
  const [failed, setFailed] = useState<string>();
  const [at, setAt] = useState<string>();
  const [wrote, setWrote] = useState("");
  const [sending, setSending] = useState(false);
  const [refused, setRefused] = useState<string>();
  const foot = useRef<HTMLDivElement>(null);

  // read is a whole re-read of the thread, which is what a line said or
  // taken back asks for: the record decides what the thread is, and a page
  // that patched its own copy would be a second opinion about it.
  const [again, setAgain] = useState(0);

  useEffect(() => {
    let stale = false;

    api
      .supervisor()
      .then((got) => {
        if (stale) return;
        setChats(got.chats);
        setSaid(got.said);
        setRead(got.read);
        setFailed(got.failed);
        setAt(got.chats.at(-1)?.id);
      })
      .catch((e: Error) => !stale && setFailed(e.message));

    return () => {
      stale = true;
    };
  }, [again]);

  // Every conversation the record lists, and — when there are turns that
  // belong to none of them — one more for those. Conversations were given
  // ids partway through Orbit's life, and everything said before that is a
  // thread with no id: without this the screen simply hid it.
  const threads = useMemo(() => {
    const known = new Set(chats?.map((c) => c.id));
    const loose = said.filter((s) => s.kind === "supervisor.message" && !known.has(s.conversation ?? ""));

    if (loose.length === 0) return chats ?? [];

    return [
      {
        id: earlier,
        title: "Before conversations had ids",
        first: loose[0]?.at ?? "",
        last: loose.at(-1)?.at ?? "",
        turns: loose.length,
      },
      ...(chats ?? []),
    ];
  }, [chats, said]);

  const turns = useMemo(() => {
    const known = new Set((chats ?? []).map((c) => c.id));

    return said.filter((s) => {
      if (s.kind !== "supervisor.message") return false;
      if (at === earlier) return !known.has(s.conversation ?? "");

      return !at || s.conversation === at;
    });
  }, [said, at, chats]);

  // The end of a conversation is where it is happening, so that is where it
  // opens — the same place the cockpit puts you.
  useEffect(() => {
    foot.current?.scrollIntoView({ block: "end" });
  }, [turns]);

  const say = async () => {
    const text = wrote.trim();
    if (text === "" || sending) return;

    setSending(true);
    setRefused(undefined);

    try {
      await api.did("say", { text });
      setWrote("");
      setAgain((n) => n + 1);
    } catch (e) {
      setRefused((e as Error).message);
    } finally {
      setSending(false);
    }
  };

  // The line is pointed at by its number in the listing, which is what the
  // record can be pointed at by: a turn has no id, and the thread only ever
  // grows at the end. `orbit retract` reads it back the same way.
  const takeBack = async (n: number) => {
    setRefused(undefined);

    try {
      await api.did("retract", { line: String(n) });
      setAgain((k) => k + 1);
    } catch (e) {
      setRefused((e as Error).message);
    }
  };

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!chats) return <p className="text-xs text-aside">Reading the thread…</p>;

  if (!read) {
    return (
      <Empty
        said="Nothing was read"
        next="This build has no store to ask, so it says nothing rather than pretending the thread is empty."
      />
    );
  }

  const box = (
    <div className="flex flex-col gap-1">
      <div className="flex items-end gap-1.5">
        <textarea
          value={wrote}
          onChange={(e) => setWrote(e.target.value)}
          onKeyDown={(e) => {
            // Enter says it and shift-enter breaks the line, which is what
            // every other box a person types into does.
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              void say();
            }
          }}
          rows={2}
          placeholder="Ask the supervisor something, or tell it what matters."
          className="min-w-0 flex-1 resize-y rounded border border-edge bg-well px-2 py-1.5 text-xs text-said outline-none placeholder:text-faint focus:border-accent/50"
        />
        <button
          onClick={() => void say()}
          disabled={sending || wrote.trim() === ""}
          className="shrink-0 rounded border border-accent/40 bg-accent/10 px-2.5 py-1.5 text-[11px] text-accent transition-colors hover:bg-accent/20 disabled:opacity-40"
        >
          {sending ? "…" : "Say"}
        </button>
      </div>

      {refused && <p className="text-[11px] text-bad">{refused}</p>}
    </div>
  );

  if (said.length === 0) {
    return (
      <div className="flex measure flex-col gap-3">
        <Empty
          said="Nobody has spoken to the supervisor yet"
          next="Say the first thing here, or run orbit say. It answers with what it can see of the board."
        />
        {box}
      </div>
    );
  }

  return (
    <div className="flex measure gap-4">
      <aside className="sticky top-0 flex max-h-[calc(100vh-8rem)] w-52 shrink-0 flex-col gap-1 self-start overflow-y-auto">
        {[...threads].reverse().map((chat) => (
          <button
            key={chat.id}
            onClick={() => setAt(chat.id)}
            className={`rounded px-2 py-1 text-left transition-colors ${
              at === chat.id ? "bg-accent/12 text-accent" : "text-aside hover:bg-hover"
            }`}
          >
            <span className="block truncate text-[11px]">{chat.title || "Untitled"}</span>
            <span className="block text-[10px] text-faint tabular-nums">
              {chat.turns} turns · {when(chat.last)}
            </span>
          </button>
        ))}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col gap-2">
        {turns.map((turn, i) => (
          <Turn
            key={i}
            said={turn}
            // The number the record answers to, which is the position in the
            // whole thread and not in the conversation being shown.
            line={said.indexOf(turn) + 1}
            takeBack={takeBack}
          />
        ))}
        <div ref={foot} />
        {box}
      </div>
    </div>
  );
}

// Turn is one thing said: who, when, and the words.
function Turn({
  said,
  line,
  takeBack,
}: {
  said: Said;
  line: number;
  takeBack: (n: number) => void;
}) {
  const person = people.has((said.by ?? "").toLowerCase());

  return (
    <article
      className={`rounded-md border px-3 py-2 ${
        person ? "border-accent/25 bg-accent/5" : "border-edge bg-panel"
      }`}
    >
      <header className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
        <span className={`font-mono text-[11px] ${person ? "text-accent" : "text-said"}`}>
          {said.by || "orbit"}
        </span>
        {said.channel && <span className="text-[10px] text-faint">via {said.channel}</span>}
        {said.task && <span className="font-mono text-[10px] text-aside">{said.task}</span>}
        {said.repo && <span className="font-mono text-[10px] text-faint">{said.repo}</span>}
        <time className="ml-auto shrink-0 text-[10px] text-faint tabular-nums">
          {when(said.at)}
        </time>

        {/* Only what a person said. Taking back an engine's answer would be
            editing the record of what it told you, which is the one thing
            this thread is for keeping. */}
        {person && (
          <button
            onClick={() => takeBack(line)}
            title={`Take back line ${line}`}
            className="shrink-0 text-faint transition-colors hover:text-bad"
          >
            <Undo2 className="size-3" />
          </button>
        )}
      </header>

      <p className="mt-1 measure text-xs whitespace-pre-wrap text-aside">{said.text}</p>
    </article>
  );
}

function when(at: string): string {
  const t = new Date(at);

  return Number.isNaN(t.getTime())
    ? "—"
    : t.toLocaleString([], { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
}
