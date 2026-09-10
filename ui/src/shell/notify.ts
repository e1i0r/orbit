// The board, when nobody is looking at it.
//
// A page is only the cockpit while it is on screen. Installed on a phone and
// left in a pocket, it has nothing to say — and the whole reason to carry it
// there is the two moments that cannot wait: a task that needs a person, and
// a task that broke.
//
// So the board's own poll is read for changes rather than only drawn. A task
// that moves into needs_you, into a failure, or to done is a notification,
// and everything else — a phase starting, a tool call, a cost going up — is
// not. Notify on everything and it is muted by Wednesday.
//
// What this is not: push. The notification is raised by the page, so it
// arrives while the tab is open, in the background, or while the installed
// app is in the switcher — and not with it closed. Reaching a closed app
// needs a push service and a server holding a key, which is the day orbit
// has a server; this is the whole of what a board on your own machine can
// do, and it is most of it.

import type { Band, TaskSummary } from "../api";

// asked is what the reader chose, kept where the browser keeps it so that
// the choice survives a reload. The permission itself is the browser's
// answer and is asked for separately: a reader can want notifications from
// this board and have said no to the browser, and the two must not be
// mistaken for each other.
const asked = "orbit.notify";

export function wanted(): boolean {
  return localStorage.getItem(asked) === "yes" && permission() === "granted";
}

export function permission(): NotificationPermission | "unsupported" {
  return "Notification" in window ? Notification.permission : "unsupported";
}

// want turns them on, which means asking the browser if it has not been
// asked. It answers what the reader ended up with, so the switch draws the
// truth rather than what was pressed.
export async function want(on: boolean): Promise<boolean> {
  if (!on) {
    localStorage.setItem(asked, "no");

    return false;
  }

  if (permission() === "unsupported") return false;

  const said = permission() === "default" ? await Notification.requestPermission() : permission();

  localStorage.setItem(asked, said === "granted" ? "yes" : "no");

  return said === "granted";
}

// worth is what a reader would want to be interrupted for, and the sentence
// that says it. A task arriving in the band it is already in is not news.
function worth(was: Band | undefined, now: Band): string {
  if (was === now || was === undefined) return "";

  switch (now) {
    case "needs_you":
      return "needs you";
    case "done":
      return "finished";
    default:
      return "";
  }
}

// tell raises what changed between two readings of the board.
//
// The id and the title, because a notification that says "a task needs you"
// is a notification the reader has to open the app to understand. The tag is
// the task's own id, so a task that changes twice replaces its own
// notification rather than stacking two.
export function tell(before: TaskSummary[], after: TaskSummary[]): void {
  if (!wanted()) return;

  const was = new Map(before.map((t) => [t.id, t.band]));

  for (const task of after) {
    const said = worth(was.get(task.id), task.band);
    if (said === "") continue;

    new Notification(`${task.id} ${said}`, {
      body: task.title,
      tag: task.id,
      icon: "/icon-192.png",
    });
  }
}
