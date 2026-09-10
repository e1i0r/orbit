// Orbit's own settings.
//
// The window has had this screen since it had settings; the browser had
// none, so a reader who opened orbit in a tab could watch the autopilot hold
// their board back and had no way to turn it off without a terminal.
//
// The table is not written here. internal/verb declares every setting —
// what it is called, what it holds now, what it means, and what it will
// accept — and this draws what it is told: a setting added there appears
// here without anybody remembering to come and add it.

import { useEffect, useState } from "react";
import { api, type Setting } from "../api";

export function SettingsScreen() {
  const [all, setAll] = useState<Setting[]>();
  const [failed, setFailed] = useState<string>();
  const [busy, setBusy] = useState<string>();
  const [refused, setRefused] = useState<string>();

  const read = () =>
    api
      .read<Setting[]>("settings")
      .then((got) => setAll(got.saw))
      .catch((e: Error) => setFailed(e.message));

  useEffect(() => {
    void read();
  }, []);

  // What the setting holds is read back from the record rather than kept
  // here: a switch that stored its own idea of being on would go on saying
  // so after the write that failed.
  const set = async (name: string, value: string) => {
    setBusy(name);
    setRefused(undefined);

    try {
      await api.did("set", { key: name, value });
      await read();
    } catch (e) {
      setRefused((e as Error).message);
    } finally {
      setBusy(undefined);
    }
  };

  if (failed) return <p className="text-xs text-bad">{failed}</p>;
  if (!all) return <p className="text-xs text-aside">Reading the settings…</p>;

  return (
    <div className="flex measure flex-col gap-1.5">
      {refused && <p className="text-[11px] text-bad">{refused}</p>}

      {all.map((one) => (
        <Row key={one.name} setting={one} busy={busy === one.name} set={set} />
      ))}
    </div>
  );
}

// Row is one setting: what it is called, what it means, and the control that
// changes it.
//
// A switch where the value is on or off and a box otherwise, decided from
// what the setting currently holds. The alternative was a list here saying
// which settings are switches — a second copy of the settings table, and the
// copy that drifts.
function Row({
  setting,
  busy,
  set,
}: {
  setting: Setting;
  busy: boolean;
  set: (name: string, value: string) => void;
}) {
  const [draft, setDraft] = useState(setting.value === "—" ? "" : setting.value);
  const toggle = setting.value === "on" || setting.value === "off";

  useEffect(() => {
    setDraft(setting.value === "—" ? "" : setting.value);
  }, [setting.value]);

  return (
    <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 rounded border border-edge bg-well/40 px-2.5 py-2">
      <span className="w-40 shrink-0 font-mono text-[11px] text-said">{setting.name}</span>

      {toggle ? (
        <button
          onClick={() => set(setting.name, setting.value === "on" ? "off" : "on")}
          disabled={busy}
          aria-pressed={setting.value === "on"}
          className={`w-14 shrink-0 rounded border px-2 py-0.5 text-[11px] transition-colors disabled:opacity-40 ${
            setting.value === "on"
              ? "border-accent/40 bg-accent/10 text-accent hover:bg-accent/20"
              : "border-edge bg-well text-faint hover:text-said"
          }`}
        >
          {busy ? "…" : setting.value}
        </button>
      ) : (
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && draft !== setting.value && set(setting.name, draft)}
          onBlur={() => draft !== "" && draft !== setting.value && set(setting.name, draft)}
          disabled={busy}
          placeholder="not set"
          className="w-40 shrink-0 rounded border border-edge bg-well px-1.5 py-0.5 font-mono text-[11px] text-said outline-none placeholder:text-faint focus:border-accent/50 disabled:opacity-40"
        />
      )}

      <span className="min-w-0 flex-1 text-[11px] text-faint">{setting.about}</span>
    </div>
  );
}
