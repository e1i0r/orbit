// The frame every screen is drawn in: where you are on the left, what is
// true right now across the top.
//
// The shape is frauddi's — a grouped rail, a bar, a page header — because
// that is a product people already read. What is in it is Orbit's.

import type { Board } from "../api";

const screens = [
  { group: "The board", items: [{ id: "board", name: "Tasks" }] },
  {
    group: "What Orbit knows",
    items: [
      { id: "supervisor", name: "Supervisor" },
      { id: "knowledge", name: "Facts" },
      { id: "flows", name: "Flows" },
    ],
  },
  {
    group: "The machine",
    items: [
      { id: "engines", name: "Engines" },
      { id: "repos", name: "Repositories" },
      { id: "settings", name: "Settings" },
    ],
  },
];

export function Shell({
  at,
  go,
  board,
  children,
}: {
  at: string;
  go: (id: string) => void;
  board?: Board;
  children: React.ReactNode;
}) {
  return (
    <div className="grid h-screen grid-cols-[188px_1fr] bg-page">
      <nav className="flex flex-col overflow-y-auto border-r border-edge bg-panel">
        <div className="flex h-11 shrink-0 items-center gap-2 border-b border-edge px-4">
          <span className="text-sm leading-none text-accent">◉</span>
          <span className="text-sm font-semibold tracking-tight">orbit</span>
        </div>

        {screens.map((section) => (
          <div key={section.group} className="px-2">
            <h2 className="px-2 pt-4 pb-1 text-[10px] font-semibold tracking-[0.09em] text-faint uppercase">
              {section.group}
            </h2>
            {section.items.map((item) => (
              <button
                key={item.id}
                onClick={() => go(item.id)}
                aria-current={at === item.id ? "page" : undefined}
                className={`flex w-full items-center rounded-md px-2 py-1.5 text-left text-sm transition-colors ${
                  at === item.id
                    ? "bg-accent/12 font-medium text-accent"
                    : "text-aside hover:bg-hover hover:text-said"
                }`}
              >
                {item.name}
              </button>
            ))}
          </div>
        ))}

        <div className="mt-auto border-t border-edge px-4 py-3">
          <p className="truncate font-mono text-[10px] text-faint" title={board?.root}>
            {board?.root ?? "…"}
          </p>
        </div>
      </nav>

      <div className="flex min-w-0 flex-col overflow-hidden">
        <header className="flex h-11 shrink-0 items-center justify-end gap-5 border-b border-edge px-5">
          {board && (
            <>
              <Fact n={board.repos?.length ?? 0} of="repository" plural="repositories" />
              <Fact n={board.tasks?.length ?? 0} of="task" plural="tasks" />
            </>
          )}
        </header>

        <main className="min-w-0 flex-1 overflow-y-auto">{children}</main>
      </div>
    </div>
  );
}

// Fact is one number the top bar keeps, with the word that says what it
// counts — a number on its own is a number nobody can check.
function Fact({ n, of, plural }: { n: number; of: string; plural: string }) {
  return (
    <span className="text-[11px] text-aside">
      <span className="font-mono text-said">{n}</span> {n === 1 ? of : plural}
    </span>
  );
}
