// What `orbit web` answers, and the one place that asks it.
//
// The shapes here mirror internal/web/answer.go, which is deliberately not
// internal/view's own structs: those are the fold of the record and they
// change when the record does. This file changes when that one does, and
// nowhere else in the app knows the wire.

export type Band = "needs_you" | "running" | "todo" | "done";

export interface TaskSummary {
  id: string;
  title: string;
  band: Band;
  repo: string;
  repos: string[] | null;
  flow: string;
  phase: string;
  engine: string;
  model: string;
}

export interface Entry {
  at: string;
  kind: string;
  phase: string;
  attempt: number;
  text?: string;
  engine?: string;
  model?: string;
  cost?: number;
  tool?: string;
  gate?: string;
  exit?: string;
  story?: Story;
  delta?: Delta;
  truncated?: boolean;
}

export interface Board {
  root: string;
  repos: { name: string; path: string }[] | null;
  bands: { name: Band; count: number }[];
  tasks: TaskSummary[] | null;
  readAt: string;
}

export interface Story {
  entry?: string;
  purpose?: string;
  symptom?: string;
  cause?: string;
  fix?: string;
}

export interface Delta {
  needs?: string[];
  guarantees?: string[];
  assumes?: string[];
  instead?: string[];
}

export interface Step {
  path: string;
  touches: number;
  read: number;
}

export interface Task extends TaskSummary {
  entries: Entry[] | null;
  walk: Step[] | null;
  spent: number;
  /** Whether a process is running this task right now. Not the band: a run
   *  parked at a gate is in "needs you" and is very much alive. */
  held: boolean;
  /** The libraries the dependency gate is waiting on somebody to accept. */
  pending?: string[];
}

export type Verb =
  | "start"
  | "pause"
  | "resume"
  | "continue"
  | "skip"
  | "cancel"
  | "note"
  | "direct"
  | "requeue"
  | "approve";

/** What a verb carries, for the three that take the reader's own words. */
export interface Says {
  text?: string;
  restart?: boolean;
}

export interface Did {
  did: Verb;
  said: string;
  of?: string[];
}

export type Standing =
  | "pending"
  | "running"
  | "looping"
  | "waiting"
  | "done"
  | "failed"
  | "cancelled";

export interface Gate {
  name: string;
  command: string;
}

export interface Phase {
  name: string;
  standing: Standing;
  engine?: string;
  model?: string;
  effort?: string;
  thinking?: string;
  waits?: boolean;
  permissions?: string[];
  gates?: Gate[];
  loop?: Loop;
  cost?: number;
  started?: string;
  ended?: string;
  cause?: string;
  exit?: string;
  said?: string;
}

export interface Loop {
  max: number;
  turns: number;
  until?: Gate[];
  phases?: Phase[];
}

export interface Flow {
  id: string;
  name: string;
  description?: string;
  attempts: number;
  diffBudget?: number;
  phases?: Phase[];
  failed?: string;
}

export interface Coupled {
  file: string;
  with: string;
  times: number;
  of: number;
  ratio: number;
}

export interface Contract {
  file: string;
  says: string;
}

export interface Impact {
  id: string;
  missing?: boolean;
  failed?: string;
  changed?: string[];
  commits: number;
  coupled?: Coupled[];
  contracts?: Contract[];
  delta?: Delta;
}

export interface FlowShape extends Flow {
  origin: "builtin" | "yours" | "shadow" | "unknown";
}

export interface Fact {
  phrase: string;
  scope: string;
  source: string;
  action: "stops" | "warns";
  check?: string;
  ref?: string;
  repo?: string;
  at: string;
  used: number;
  off?: boolean;
}

export interface Chat {
  id: string;
  title: string;
  first: string;
  last: string;
  turns: number;
}

export interface Said {
  at: string;
  kind: string;
  by?: string;
  channel?: string;
  task?: string;
  repo?: string;
  text?: string;
  conversation?: string;
}

export interface Window {
  label: string;
  pct: number;
  /** Seconds. */
  resetsIn: number;
}

export interface EngineInfo {
  name: string;
  available: boolean;
  models?: string[];
  efforts?: string[];
  canThink?: boolean;
  setup?: string[];
  money?: boolean;
  sourced?: boolean;
  quota?: Window[];
}

export interface RepoDetail {
  name: string;
  path: string;
  tasks: number;
  bands: Partial<Record<Band, number>>;
}

export interface FileText {
  id: string;
  path: string;
  text?: string;
  missing?: boolean;
}

export interface Diff {
  id: string;
  text?: string;
  empty?: boolean;
  missing?: boolean;
  failed?: string;
}

async function ask<T>(path: string): Promise<T> {
  const res = await fetch(path);
  const body = await res.json();

  if (!res.ok) {
    throw new Error(body?.error ?? res.statusText);
  }

  return body as T;
}

// tell asks the server to do something rather than to say something.
//
// The content type is not decoration: it is half of what stops another site
// from posting here. A cross-origin form can only send three types and none
// of them is JSON, so this makes the browser ask permission first — and the
// server never grants it. See internal/web/verbs.go.
async function tell<T>(path: string, says: Says = {}): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(says),
  });
  const body = await res.json();

  if (!res.ok) {
    throw new Error(body?.error ?? res.statusText);
  }

  return body as T;
}

export const api = {
  board: () => ask<Board>("/api/board"),
  task: (id: string) => ask<Task>(`/api/tasks/${encodeURIComponent(id)}`),
  diff: (id: string, how?: { space?: "ignore" }) =>
    ask<Diff>(
      `/api/tasks/${encodeURIComponent(id)}/diff${how?.space ? `?space=${how.space}` : ""}`,
    ),
  file: (id: string, path: string) =>
    ask<FileText>(
      `/api/tasks/${encodeURIComponent(id)}/file?path=${encodeURIComponent(path)}`,
    ),
  flow: (id: string) => ask<Flow>(`/api/tasks/${encodeURIComponent(id)}/flow`),
  impact: (id: string) => ask<Impact>(`/api/tasks/${encodeURIComponent(id)}/impact`),
  flows: () => ask<{ flows: FlowShape[] }>("/api/flows"),
  knowledge: () => ask<{ facts: Fact[]; read: boolean }>("/api/knowledge"),
  supervisor: () =>
    ask<{ chats: Chat[]; said: Said[]; read: boolean; failed?: string }>("/api/supervisor"),
  engines: () => ask<{ engines: EngineInfo[]; settled?: string; read: boolean }>("/api/engines"),
  repos: () => ask<{ root: string; repos: RepoDetail[] }>("/api/repos"),
  do: (id: string, verb: Verb, says?: Says) =>
    tell<Did>(`/api/tasks/${encodeURIComponent(id)}/${verb}`, says),
};
