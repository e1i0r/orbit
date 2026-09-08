// A pill is a value that has a state, said in one word.
//
// The tint carries the meaning and the text carries the word: a pill nobody
// can read without knowing the colour code is a colour code, not a label.

type Tone = "accent" | "ok" | "live" | "wait" | "bad" | "quiet";

const tones: Record<Tone, string> = {
  accent: "bg-accent/15 text-accent",
  ok: "bg-ok/15 text-ok",
  live: "bg-live/15 text-live",
  wait: "bg-wait/15 text-wait",
  bad: "bg-bad/15 text-bad",
  quiet: "bg-edge/60 text-aside",
};

export function Pill({ tone = "quiet", children }: { tone?: Tone; children: React.ReactNode }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${tones[tone]}`}
    >
      {children}
    </span>
  );
}
