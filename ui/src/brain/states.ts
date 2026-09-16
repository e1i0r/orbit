// The five states a rule can be in, and one set of words for them.
//
// The fold is internal/knowledge's — whether a rule is waiting to be decided
// about, whether somebody stopped it applying, and what it would do if it
// were are three questions in the record, and every surface folds them the
// same way. What is here is only the spelling, and it is the spelling the
// cockpit uses, so a rule read in one and then the other reads the same.

export type State = "waiting" | "blocks" | "says" | "paused" | "off";

/** The order a list draws them: what wants an answer, then what is working,
 *  then what is not. */
export const states: State[] = ["waiting", "blocks", "says", "paused", "off"];

export const mark: Record<State, string> = {
  waiting: "🛑",
  blocks: "⚡",
  says: "💬",
  paused: "😴",
  off: "🚫",
};

/** Five adjectives and not five verbs. A state is where a rule ended up
 *  because of something somebody did to it, so the word says how it stands
 *  and not what it is in the middle of doing. */
export const named: Record<State, string> = {
  waiting: "PENDING",
  blocks: "BLOCKED",
  says: "ACTIVE",
  paused: "PAUSED",
  off: "TURNED OFF",
};

/** The tint each one is said in, and it is the same tint wherever it is
 *  said: only the first is ever worth the loud colour, because it is the
 *  only one asking for something. */
export const tint: Record<State, string> = {
  waiting: "text-wait",
  blocks: "text-bad",
  says: "text-live",
  paused: "text-wait",
  off: "text-faint",
};

export const tone: Record<State, "wait" | "bad" | "live" | "quiet"> = {
  waiting: "wait",
  blocks: "bad",
  says: "live",
  paused: "wait",
  off: "quiet",
};

/** What a rule says about where it came from, in the two or three words a
 *  card has room for. */
export const from: Record<string, string> = {
  "read off the code": "the code",
  "said by a person": "you",
  "learned from a run": "a model",
  "from an incident": "production",
  "the project already said it": "the project",
  "the history says so": "the history",
  "what this repo enforces": "this repo's gates",
  unsourced: "nowhere",
};
