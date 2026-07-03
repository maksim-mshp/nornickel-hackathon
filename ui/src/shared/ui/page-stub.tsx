export function PageStub({
  title,
  description,
  plannedBlocks,
}: {
  title: string;
  description: string;
  plannedBlocks: string[];
}) {
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-10">
      <section className="rise-in">
        <h1 className="font-display text-xl font-extrabold text-ink-0">
          {title}
        </h1>
        <p className="mt-1 text-[13px] text-ink-1">{description}</p>
      </section>
      <section
        className="rise-in rounded-sm border border-line bg-bg-1 p-5"
        style={{ animationDelay: "40ms" }}
      >
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-ink-2">
          раздел в разработке · состав экрана
        </p>
        <ul className="mt-3 flex flex-col gap-2">
          {plannedBlocks.map((block) => (
            <li key={block} className="flex items-center gap-3">
              <span className="hatch h-4 w-10 shrink-0 rounded-sm" />
              <span className="text-[13px] text-ink-1">{block}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
