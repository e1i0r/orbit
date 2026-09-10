// A card is one subject, with a label saying which.

export function Card({
  label,
  right,
  children,
}: {
  label?: string;
  right?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-lg border border-edge bg-panel">
      {(label || right) && (
        <header className="flex items-center justify-between gap-4 border-b border-edge px-3 py-2">
          {label && (
            <h2 className="text-[11px] font-medium text-faint">
              {label}
            </h2>
          )}
          {right}
        </header>
      )}
      <div className="p-3">{children}</div>
    </section>
  );
}
