export function Isolines({ className = "" }: { className?: string }) {
  return (
    <svg
      className={`pointer-events-none absolute inset-0 h-full w-full ${className}`}
      preserveAspectRatio="xMidYMid slice"
      viewBox="0 0 800 400"
      aria-hidden
    >
      <g fill="none" stroke="var(--void)" strokeWidth="1" opacity="0.06">
        <path d="M-20 320 C 120 260, 200 340, 340 290 S 620 220, 820 280" />
        <path d="M-20 280 C 140 220, 240 300, 380 250 S 640 180, 820 240" />
        <path d="M-20 240 C 160 180, 280 260, 420 210 S 660 140, 820 200" />
        <path d="M-20 200 C 180 140, 320 220, 460 170 S 680 100, 820 160" />
        <path d="M-20 160 C 200 100, 360 180, 500 130 S 700 60, 820 120" />
        <path d="M100 400 C 180 330, 300 380, 420 340 S 700 300, 820 340" />
        <ellipse cx="240" cy="120" rx="90" ry="34" />
        <ellipse cx="240" cy="120" rx="60" ry="21" />
        <ellipse cx="240" cy="120" rx="32" ry="10" />
        <ellipse cx="600" cy="80" rx="70" ry="26" />
        <ellipse cx="600" cy="80" rx="42" ry="14" />
      </g>
    </svg>
  );
}
