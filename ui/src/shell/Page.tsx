// A page says what it is and, under that, what it is for. The tabs below
// belong to the page rather than to the app: they are how this subject is
// looked at, not where the reader can go.

export function Page({
  title,
  said,
  does,
  tabs,
  at,
  go,
  children,
}: {
  title: React.ReactNode;
  said?: string;
  /** What can be done to this subject, drawn opposite its name. */
  does?: React.ReactNode;
  tabs?: { id: string; name: string }[];
  at?: string;
  go?: (id: string) => void;
  children: React.ReactNode;
}) {
  return (
    <div className="mx-auto max-w-[1400px]">
      <div className="px-5 pt-5">
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between md:gap-4">
          <div className="min-w-0">
            <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
            {said && <p className="mt-0.5 text-xs text-aside">{said}</p>}
          </div>
          {does}
        </div>

        {tabs && (
          <div className="mt-4 flex gap-4 overflow-x-auto border-b border-edge">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => go?.(tab.id)}
                className={`-mb-px shrink-0 border-b-2 pb-2 text-xs transition-colors ${
                  at === tab.id
                    ? "border-accent text-accent"
                    : "border-transparent text-aside hover:text-said"
                }`}
              >
                {tab.name}
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="px-5 py-4">{children}</div>
    </div>
  );
}
