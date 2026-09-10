// The frame every screen is drawn in: where you are on the left, what is
// true right now across the top.
//
// The shape is frauddi's — a grouped rail, a bar, a page header — because
// that is a product people already read. What is in it is Orbit's.

import { useEffect, useState } from "react";
import { Menu, X } from "lucide-react";

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
  // The rail is a rail on a desk and a drawer in a hand. On a phone 188
  // columns of menu leave 200 for the board, which is not a board — so under
  // the breakpoint it slides in over the page and goes away again the moment
  // it has been used.
  const [open, setOpen] = useState(false);

  // Somewhere else on the page, or the escape key, closes it. A drawer a
  // reader has to hunt for the handle of is a drawer they leave open.
  useEffect(() => {
    if (!open) return;

    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);

    window.addEventListener("keydown", onKey);

    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const choose = (id: string) => {
    go(id);
    setOpen(false);
  };

  return (
    <div className="h-screen bg-page md:grid md:grid-cols-[188px_1fr]">
      {open && (
        <button
          type="button"
          aria-label="Close the menu"
          onClick={() => setOpen(false)}
          className="fixed inset-0 z-20 bg-black/60 md:hidden"
        />
      )}

      <nav
        className={`fixed inset-y-0 left-0 z-30 flex w-[188px] flex-col overflow-y-auto border-r border-edge bg-panel transition-transform md:static md:translate-x-0 ${
          open ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <div className="flex h-11 shrink-0 items-center gap-2 border-b border-edge px-4">
          <span className="text-sm leading-none text-accent">◉</span>
          <span className="text-sm font-semibold tracking-tight">orbit</span>
        </div>

        {screens.map((section) => (
          <div key={section.group} className="px-2">
            <h2 className="px-2 pt-4 pb-1 text-[11px] font-medium text-faint">
              {section.group}
            </h2>
            {section.items.map((item) => (
              <button
                key={item.id}
                onClick={() => choose(item.id)}
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

      <div className="flex h-screen min-w-0 flex-col overflow-hidden md:h-auto">
        <header className="flex h-11 shrink-0 items-center gap-5 border-b border-edge px-5">
          <button
            type="button"
            onClick={() => setOpen(!open)}
            aria-label={open ? "Close the menu" : "Open the menu"}
            aria-expanded={open}
            className="-ml-2 rounded-md p-2 text-aside hover:bg-hover hover:text-said md:hidden"
          >
            {open ? <X size={16} /> : <Menu size={16} />}
          </button>

          <span className="flex items-center gap-2 md:hidden">
            <span className="text-sm leading-none text-accent">◉</span>
            <span className="text-sm font-semibold tracking-tight">orbit</span>
          </span>

          <span className="ml-auto flex items-center gap-5">
            {board && (
            <>
                <Fact n={board.repos?.length ?? 0} of="repository" plural="repositories" />
                <Fact n={board.tasks?.length ?? 0} of="task" plural="tasks" />
              </>
            )}
          </span>
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
