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

// Verb is a name from internal/verb's declaration. It is not a closed list
// here on purpose: the server takes any verb the declaration carries, and a
// union repeated in this file would be a fifth opinion about the vocabulary.
export type Verb = string;

// What a verb was asked with: its fields, by the names internal/verb
// declares them under.
//
// Not two named fields. Every verb takes what it says it takes — a note
// takes text, a join takes name, a permit takes yes — and a body keyed by
// what this page felt like calling it is a body the verb reads as empty.
export type Says = Record<string, string | boolean | undefined>;

export interface Did {
  did: Verb;
  said: string;
  of?: string[];
  /** saw is what a verb read, where it read something. */
  saw?: unknown;
}

/** Side is one of the flow's checks, run on both sides of the change. */
export interface Side {
  name: string;
  command: string;
  base: Ran;
  now: Ran;
  /** broke is a check this change turned from passing to failing. */
  broke?: boolean;
  fixed?: boolean;
}

/** Ran is what one command answered on one side. */
export interface Ran {
  exit: number;
  out?: string;
  /** failed is why it could not be run at all, which is not a check failing. */
  failed?: string;
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

// Cell is one node of the repository's tree: a directory or a file, with
// what the task did under it summed upward. It arrives whole, once — the
// shape is paths and counts, no contents — so every gesture on the map after
// that is local.
export interface Cell {
  name: string;
  path: string;
  cells?: Cell[];
  /** How many files under this one the task touched. */
  changed?: number;
  /** What it wrote in them, added and deleted together. */
  lines?: number;
  /** The siblings the repository's history moves this one with, strongest
   *  first. It is what a honeycomb seats its cells by: a drawing whose
   *  cells touch claims that touching means something. */
  with?: { path: string; times: number }[];
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

/** Setting is one of Orbit's own settings, as internal/verb reports it. */
export interface Setting {
  name: string;
  /** value is what it holds now, and "—" for one nobody has chosen. */
  value: string;
  about: string;
}

export interface RepoDetail {
  name: string;
  path: string;
  tasks: number;
  bands: Partial<Record<Band, number>>;
}

export interface Written {
  id: string;
  text: string;
  repo?: string;
  flow?: string;
  start?: boolean;
}

export interface Wrote {
  /** What was done, and what it was done to: the id of the new task. */
  did: string;
  said: string;
  of?: string[];
}

export interface Told {
  id: string;
  text?: string;
  read: boolean;
  failed?: string;
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
async function tell<T>(path: string, sent: unknown = {}): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sent),
  });
  const back = await res.json();

  if (!res.ok) {
    throw new Error(back?.error ?? res.statusText);
  }

  return back as T;
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
  history: (id: string) => ask<Told>(`/api/tasks/${encodeURIComponent(id)}/history`),
  impact: (id: string) => ask<Impact>(`/api/tasks/${encodeURIComponent(id)}/impact`),
  // Through the engine's own route rather than one written for this screen:
  // internal/verb declares the reading and every way in offers it, so the
  // page asks for it by name like the command line and the MCP server do.
  tree: (id: string) =>
    ask<{ said: string; saw: Cell }>(`/api/read/tree/${encodeURIComponent(id)}`),
  flows: () => ask<{ flows: FlowShape[] }>("/api/flows"),
  knowledge: () => ask<{ facts: Fact[]; read: boolean }>("/api/knowledge"),
  supervisor: () =>
    ask<{ chats: Chat[]; said: Said[]; read: boolean; failed?: string }>("/api/supervisor"),
  engines: () => ask<{ engines: EngineInfo[]; settled?: string; read: boolean }>("/api/engines"),
  repos: () => ask<{ root: string; repos: RepoDetail[] }>("/api/repos"),
  do: (id: string, verb: Verb, says?: Says) =>
    tell<Did>(`/api/tasks/${encodeURIComponent(id)}/${verb}`, says),
  // The verbs that are not about one task: said to the supervisor, written
  // down about the code, changed in the settings. Same engine, same names —
  // what differs is only that there is no task in the path.
  did: (verb: Verb, says?: Says) => tell<Did>(`/api/do/${verb}`, says),
  // And the readings of the same kind, which change nothing and so are a
  // GET: settings, quota, the thread, whatever the declaration carries.
  read: <T>(verb: Verb) => ask<{ said: string; saw: T }>(`/api/read/${verb}`),
  write: (one: Written) => tell<Wrote>("/api/do/new", { ...one, run: one.start }),
};
