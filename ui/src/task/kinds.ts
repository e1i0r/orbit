// What an event is, said in words.
//
// The record's kinds are the program's vocabulary — `phase.tool_call`,
// `task.over_budget` — and they are exact. They are not a sentence. A
// timeline of forty rows of dotted lowercase is a log; what a reader wants
// is what happened, so this is the one place that turns one into the other.
//
// The cockpit does the same thing through internal/words, translated. This
// list is English only for now, and the day the web speaks two languages it
// moves behind the same door rather than growing a second copy of it.

import {
  AlertTriangle, Ban, BookOpen, Check, CircleDot, Clock, FileDiff, Flag,
  GitMerge, GitPullRequest, Hand, Hammer, Lightbulb, MessageSquare, Play,
  RotateCcw, Scale, ShieldAlert, Terminal, Trash2, X, type LucideIcon,
} from "lucide-react";

export type Tone = "ok" | "bad" | "wait" | "live" | "quiet" | "accent";

export interface Meaning {
  said: string;
  icon: LucideIcon;
  tone: Tone;
}

const meanings: Record<string, Meaning> = {
  "task.created": { said: "Task written down", icon: CircleDot, tone: "quiet" },
  "task.started": { said: "Run started", icon: Play, tone: "live" },
  "task.finished": { said: "Task finished", icon: Check, tone: "ok" },
  "task.failed": { said: "Task failed", icon: X, tone: "bad" },
  "task.stuck": { said: "Stuck — needs you", icon: Hand, tone: "wait" },
  "task.cancelled": { said: "Cancelled", icon: Ban, tone: "quiet" },
  "task.requeued": { said: "Sent back to the queue", icon: RotateCcw, tone: "wait" },
  "task.merged": { said: "Merged", icon: GitMerge, tone: "ok" },
  "task.noted": { said: "Note left", icon: MessageSquare, tone: "accent" },
  "task.dialogue": { said: "Said in the session", icon: MessageSquare, tone: "accent" },
  "task.over_budget": { said: "Over budget", icon: Scale, tone: "wait" },
  "task.over_diff": { said: "Bigger than agreed", icon: FileDiff, tone: "wait" },
  "task.new_dependency": { said: "New dependency, waiting on you", icon: ShieldAlert, tone: "wait" },
  "task.contradicts": { said: "Goes against a decision", icon: AlertTriangle, tone: "bad" },
  "task.read": { said: "Marked read", icon: Check, tone: "quiet" },
  "phase.started": { said: "Phase started", icon: Play, tone: "live" },
  "phase.finished": { said: "Phase finished", icon: Check, tone: "ok" },
  "phase.failed": { said: "Phase failed", icon: X, tone: "bad" },
  "phase.retried": { said: "Phase tried again", icon: RotateCcw, tone: "wait" },
  "phase.waiting": { said: "Stopped at a gate", icon: Hand, tone: "wait" },
  "phase.cancelled": { said: "Phase cancelled", icon: Ban, tone: "quiet" },
  "phase.tool_call": { said: "Tool call", icon: Terminal, tone: "quiet" },
  "phase.thought": { said: "Thinking", icon: Lightbulb, tone: "quiet" },
  "phase.refused": { said: "Refused by the sandbox", icon: ShieldAlert, tone: "bad" },
  "gate.passed": { said: "Check passed", icon: Check, tone: "ok" },
  "gate.failed": { said: "Check failed", icon: X, tone: "bad" },
  "loop.checked": { said: "Loop checked", icon: RotateCcw, tone: "accent" },
  "decision.made": { said: "Decision recorded", icon: Flag, tone: "accent" },
  "decision.superseded": { said: "Decision replaced", icon: Flag, tone: "quiet" },
  "phase.resumed": { said: "Let through the gate", icon: Play, tone: "live" },
  "task.story": { said: "Story written", icon: BookOpen, tone: "accent" },
  "task.delta": { said: "What the change asks and promises", icon: FileDiff, tone: "accent" },
  "task.critical": { said: "Marked critical", icon: ShieldAlert, tone: "wait" },
  "task.abandoned": { said: "Abandoned", icon: Ban, tone: "wait" },
  "task.timedout": { said: "Timed out", icon: Clock, tone: "bad" },
  "task.deleted": { said: "Deleted", icon: Trash2, tone: "quiet" },
  "deliver.asked": { said: "Asked GitHub", icon: GitPullRequest, tone: "quiet" },
  "deliver.answered": { said: "GitHub answered", icon: GitPullRequest, tone: "quiet" },
  "supervisor.message": { said: "Said to the supervisor", icon: MessageSquare, tone: "accent" },
  "supervisor.briefing": { said: "Briefed the supervisor", icon: MessageSquare, tone: "quiet" },
  "supervisor.debriefing": { said: "Debriefed the supervisor", icon: MessageSquare, tone: "quiet" },
  "supervisor.action": { said: "The supervisor acted", icon: Play, tone: "accent" },
  "supervisor.retracted": { said: "Taken back", icon: RotateCcw, tone: "quiet" },
  unreadable: { said: "A line of the record could not be read", icon: AlertTriangle, tone: "bad" },
};

// meaning is what to say about one kind, and a readable fallback for a kind
// this build has never heard of — a newer Orbit wrote it, and showing it
// plainly is better than showing nothing where it was.
export function meaning(kind: string): Meaning {
  return meanings[kind] ?? { said: plainly(kind), icon: Hammer, tone: "quiet" };
}

function plainly(kind: string): string {
  const word = kind.split(".").pop() ?? kind;
  const said = word.replace(/_/g, " ");

  return said.charAt(0).toUpperCase() + said.slice(1);
}
