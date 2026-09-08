// An empty screen is an invitation, not an apology.
//
// It says what is not there and what would put something there, because a
// person looking at nothing is a person deciding whether the thing is broken.

export function Empty({ said, next }: { said: string; next?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-14 text-center">
      <div className="grid size-10 place-items-center rounded-lg bg-panel">
        <div className="size-4 rounded-full border-2 border-faint" />
      </div>
      <p className="text-xs text-said">{said}</p>
      {next && <p className="max-w-[46ch] text-[11px] text-faint">{next}</p>}
    </div>
  );
}
