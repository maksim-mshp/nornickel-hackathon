import type { Consensus } from "@/shared/api/types";

const WIDTH = 640;
const ROW_H = 26;
const PAD_X = 8;
const AXIS_H = 24;
const LABEL_W = 190;

const VERDICT_LABELS: Record<Consensus["verdict"], string> = {
  consensus: "консенсус",
  majority: "большинство",
  split: "раскол",
  insufficient: "мало данных",
};

const nf = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 });

export function ConsensusSpectrum({ consensus }: { consensus: Consensus }) {
  const values = consensus.sources.flatMap((s) => [s.vmin, s.vmax]);
  const lo = Math.min(...values, consensus.agreedMin);
  const hi = Math.max(...values, consensus.agreedMax);
  const span = hi - lo || 1;
  const plotW = WIDTH - LABEL_W - PAD_X * 2;
  const x = (v: number) => LABEL_W + PAD_X + ((v - lo) / span) * plotW;
  const height = AXIS_H + consensus.sources.length * ROW_H + 12;
  const ticks = [lo, lo + span / 4, lo + span / 2, lo + (3 * span) / 4, hi];

  return (
    <figure className="rounded-sm border border-line bg-bg-1 p-4">
      <figcaption className="flex flex-wrap items-baseline gap-2">
        <span className="text-[13px] font-semibold text-ink-0">
          {consensus.parameter.name}
        </span>
        <span className="font-mono text-[11px] text-ink-2">
          {consensus.unit}
        </span>
        <span className="ml-auto flex items-center gap-2 font-mono text-[10px]">
          <span
            className={
              consensus.verdict === "consensus" ||
              consensus.verdict === "majority"
                ? "text-electrolyte"
                : "text-melt"
            }
          >
            {VERDICT_LABELS[consensus.verdict]}
          </span>
          <span className="text-ink-2">
            пересечение {Math.round(consensus.overlapIndex * 100)}%
          </span>
        </span>
      </figcaption>
      <svg
        viewBox={`0 0 ${WIDTH} ${height}`}
        className="mt-2 w-full"
        role="img"
        aria-label={`Спектр источников по параметру ${consensus.parameter.name}`}
      >
        <rect
          x={x(consensus.agreedMin)}
          y={AXIS_H - 6}
          width={Math.max(x(consensus.agreedMax) - x(consensus.agreedMin), 2)}
          height={consensus.sources.length * ROW_H + 6}
          fill="var(--electrolyte)"
          opacity="0.12"
        />
        {ticks.map((tick) => (
          <g key={tick}>
            <line
              x1={x(tick)}
              y1={AXIS_H - 6}
              x2={x(tick)}
              y2={height - 12}
              stroke="var(--line)"
              strokeWidth="1"
            />
            <text
              x={x(tick)}
              y={12}
              textAnchor="middle"
              fill="var(--ink-2)"
              fontSize="9"
              fontFamily="var(--font-jetbrains)"
            >
              {nf.format(tick)}
            </text>
          </g>
        ))}
        {consensus.sources.map((source, index) => {
          const y = AXIS_H + index * ROW_H + ROW_H / 2;
          const overlaps =
            source.vmax >= consensus.agreedMin &&
            source.vmin <= consensus.agreedMax;
          const color = overlaps ? "var(--electrolyte)" : "var(--melt)";
          return (
            <g key={source.title}>
              <text
                x={0}
                y={y + 3}
                fill="var(--ink-1)"
                fontSize="10"
                fontFamily="var(--font-jetbrains)"
              >
                {source.title.length > 26
                  ? `${source.title.slice(0, 25)}…`
                  : source.title}
              </text>
              <line
                x1={x(source.vmin)}
                y1={y}
                x2={x(source.vmax)}
                y2={y}
                stroke={color}
                strokeWidth="3"
                strokeLinecap="round"
              />
              <line
                x1={x(source.vmin)}
                y1={y - 4}
                x2={x(source.vmin)}
                y2={y + 4}
                stroke={color}
                strokeWidth="1.5"
              />
              <line
                x1={x(source.vmax)}
                y1={y - 4}
                x2={x(source.vmax)}
                y2={y + 4}
                stroke={color}
                strokeWidth="1.5"
              />
            </g>
          );
        })}
        <line
          x1={x(consensus.agreedMin)}
          y1={AXIS_H - 6}
          x2={x(consensus.agreedMin)}
          y2={height - 12}
          stroke="var(--electrolyte)"
          strokeWidth="1.5"
          strokeDasharray="4 3"
        />
        <line
          x1={x(consensus.agreedMax)}
          y1={AXIS_H - 6}
          x2={x(consensus.agreedMax)}
          y2={height - 12}
          stroke="var(--electrolyte)"
          strokeWidth="1.5"
          strokeDasharray="4 3"
        />
      </svg>
      <p className="mt-1 font-mono text-[10px] text-ink-2">
        agreed range: {nf.format(consensus.agreedMin)}–
        {nf.format(consensus.agreedMax)} {consensus.unit}
      </p>
    </figure>
  );
}
