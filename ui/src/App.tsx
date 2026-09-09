// Where the reader is, and what that means.
//
// The location is the hash, so a screen can be linked to and the back button
// works, without a router for what is one board and one task.

import { useEffect, useState } from "react";
import { api, type Board } from "./api";
import { BoardScreen } from "./screens/BoardScreen";
import { EnginesScreen } from "./screens/EnginesScreen";
import { FlowsScreen } from "./screens/FlowsScreen";
import { KnowledgeScreen } from "./screens/KnowledgeScreen";
import { ReposScreen } from "./screens/ReposScreen";
import { SettingsScreen } from "./screens/SettingsScreen";
import { WriteScreen } from "./screens/WriteScreen";
import { SupervisorScreen } from "./screens/SupervisorScreen";
import { TaskScreen } from "./screens/TaskScreen";
import { Caught } from "./parts/Caught";
import { Empty } from "./parts/Empty";
import { Page } from "./shell/Page";
import { Shell } from "./shell/Shell";

// every is how often the board is asked again. The cockpit polls twice a
// second because it is watching; a tab left open wants to be current, not
// immediate.
const every = 3000;

export function App() {
  const [board, setBoard] = useState<Board>();
  const [failed, setFailed] = useState<string>();
  const [where, setWhere] = useState(location.hash.slice(1) || "board");

  useEffect(() => {
    const onHash = () => setWhere(location.hash.slice(1) || "board");
    addEventListener("hashchange", onHash);

    return () => removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    let stale = false;

    const read = () =>
      api
        .board()
        .then((b) => {
          if (stale) return;
          setBoard(b);
          setFailed(undefined);
        })
        .catch((e: Error) => !stale && setFailed(e.message));

    read();
    const timer = setInterval(read, every);

    return () => {
      stale = true;
      clearInterval(timer);
    };
  }, []);

  const go = (to: string) => {
    location.hash = to;
  };

  const task = where.startsWith("task/") ? where.slice("task/".length) : "";

  return (
    <Shell at={task ? "board" : where} go={go} board={board}>
      {failed ? (
        <Page title="The board could not be read" said={failed}>
          <Empty
            said="Orbit is not answering"
            next="The server is `orbit web`, and it stops when the terminal it was started in does."
          />
        </Page>
      ) : task ? (
        <Caught back={() => go("board")}>
          <TaskScreen id={task} back={() => go("board")} />
        </Caught>
      ) : (
        <Page
          title={screens[where]?.title ?? named(where)}
          said={screens[where]?.said}
          does={
            where === "board" && (
              <button
                onClick={() => go("write")}
                className="shrink-0 rounded border border-accent/40 bg-accent/10 px-2 py-0.5 text-[11px] text-accent transition-colors hover:bg-accent/20"
              >
                Write a task
              </button>
            )
          }
        >
          <Caught>{screen(where, board, go)}</Caught>
        </Page>
      )}
    </Shell>
  );
}

// What each screen is called and what it is for. One place, so the title,
// the sentence under it and what is drawn cannot come apart.
const screens: Record<string, { title: string; said: string }> = {
  board: {
    title: "Tasks",
    said: "Every task under this root, in the band that says what it waits for.",
  },
  write: {
    title: "Write a task",
    said: "What the work is, where it happens, and how carefully to go about it.",
  },
  supervisor: {
    title: "Supervisor",
    said: "What has been said to Orbit about the board, and what it said back.",
  },
  knowledge: {
    title: "Facts",
    said: "What Orbit has been told, and what each one does when work reaches it.",
  },
  flows: {
    title: "Flows",
    said: "The shapes of work a task can be started under.",
  },
  engines: {
    title: "Engines",
    said: "What this build knows how to run, and what this machine has installed.",
  },
  repos: {
    title: "Repositories",
    said: "The checkouts Orbit is watching, and the work in each.",
  },
  settings: {
    title: "Settings",
    said: "What Orbit does when nobody says otherwise, and what holds a run back.",
  },
};

function screen(where: string, board: Board | undefined, go: (to: string) => void) {
  switch (where) {
    case "board":
      return <BoardScreen board={board} open={(id) => go(`task/${id}`)} />;
    case "write":
      return <WriteScreen board={board} open={(id) => go(`task/${id}`)} />;
    case "supervisor":
      return <SupervisorScreen />;
    case "knowledge":
      return <KnowledgeScreen board={board} />;
    case "flows":
      return <FlowsScreen />;
    case "engines":
      return <EnginesScreen />;
    case "repos":
      return <ReposScreen />;
    case "settings":
      return <SettingsScreen />;
  }

  return (
    <Empty
      said={`There is no ${named(where)} screen`}
      next="The rail on the left is everywhere this build goes."
    />
  );
}

function named(id: string): string {
  return id.charAt(0).toUpperCase() + id.slice(1);
}
