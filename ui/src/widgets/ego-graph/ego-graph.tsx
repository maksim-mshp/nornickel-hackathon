import type { EvidencePack, QueryPlan } from "@/shared/api/types";

type GraphNode = {
  id: string;
  label: string;
  kind: "process" | "material" | "parameter" | "document" | "expert";
  x: number;
  y: number;
};

type GraphEdge = {
  from: string;
  to: string;
  contradicts?: boolean;
};

const W = 340;
const H = 300;
const CX = W / 2;
const CY = H / 2;

const KIND_COLORS: Record<GraphNode["kind"], string> = {
  process: "var(--electrolyte)",
  material: "var(--electrolyte)",
  parameter: "var(--focus)",
  document: "var(--ink-2)",
  expert: "var(--anode)",
};

const KIND_LEGEND: { kind: GraphNode["kind"]; label: string }[] = [
  { kind: "process", label: "сущность" },
  { kind: "parameter", label: "параметр" },
  { kind: "document", label: "документ" },
  { kind: "expert", label: "эксперт" },
];

function ring(
  items: { id: string; label: string; kind: GraphNode["kind"] }[],
  radius: number,
  phase: number,
): GraphNode[] {
  return items.map((item, index) => {
    const angle = phase + (index / Math.max(items.length, 1)) * Math.PI * 2;
    return {
      ...item,
      x: CX + radius * Math.cos(angle),
      y: CY + radius * Math.sin(angle),
    };
  });
}

export function EgoGraph({
  plan,
  pack,
}: {
  plan: QueryPlan;
  pack: EvidencePack;
}) {
  const center: GraphNode = {
    id: "center",
    label: plan.entities.processes[0]?.name ?? plan.entities.materials[0]?.name ?? "запрос",
    kind: "process",
    x: CX,
    y: CY,
  };

  const inner = ring(
    [
      ...plan.entities.materials.map((entity) => ({
        id: entity.slug,
        label: entity.name,
        kind: "material" as const,
      })),
      ...plan.entities.properties.map((entity) => ({
        id: entity.slug,
        label: entity.name,
        kind: "parameter" as const,
      })),
    ],
    62,
    -Math.PI / 2,
  );

  const documents = [
    ...new Map(
      pack.facts.map((fact) => [fact.provenance.documentId, fact.provenance]),
    ).values(),
  ];
  const outer = ring(
    [
      ...documents.map((provenance) => ({
        id: provenance.documentId,
        label: `№ ${provenance.documentId.replace("doc_", "")} · ${provenance.year}`,
        kind: "document" as const,
      })),
      ...pack.experts.map((expert) => ({
        id: expert.id,
        label: expert.name,
        kind: "expert" as const,
      })),
    ],
    118,
    -Math.PI / 2 + 0.35,
  );

  const nodes = [center, ...inner, ...outer];
  const byId = new Map(nodes.map((node) => [node.id, node]));

  const contradictionEdges: GraphEdge[] = pack.contradictions.flatMap(
    (contradiction) => {
      const a = pack.facts.find((fact) => fact.ref === contradiction.aFactRef);
      const b = pack.facts.find((fact) => fact.ref === contradiction.bFactRef);
      if (!a || !b) return [];
      const from = a.provenance.documentId;
      const to = b.provenance.documentId;
      if (from === to || !byId.has(from) || !byId.has(to)) return [];
      return [{ from, to, contradicts: true }];
    },
  );

  const edges: GraphEdge[] = [
    ...inner.map((node) => ({ from: "center", to: node.id })),
    ...outer.map((node) => ({ from: "center", to: node.id })),
    ...contradictionEdges,
  ];

  if (inner.length + outer.length === 0) {
    return (
      <p className="rounded-sm border border-line bg-bg-1 p-4 text-[12px] text-ink-2">
        Недостаточно связей для ego-графа по этому запросу.
      </p>
    );
  }

  return (
    <div>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="w-full"
        role="img"
        aria-label="Ego-граф сущностей запроса"
      >
        {edges.map((edge, index) => {
          const from = byId.get(edge.from);
          const to = byId.get(edge.to);
          if (!from || !to) return null;
          return (
            <line
              key={`edge-${index}-${edge.from}-${edge.to}`}
              x1={from.x}
              y1={from.y}
              x2={to.x}
              y2={to.y}
              stroke={edge.contradicts ? "var(--melt)" : "var(--line)"}
              strokeWidth={edge.contradicts ? 1.5 : 1}
              className={edge.contradicts ? "edge-pulse" : undefined}
            />
          );
        })}
        {nodes.map((node, index) => (
          <g key={`node-${index}-${node.id}`}>
            <circle
              cx={node.x}
              cy={node.y}
              r={node.id === "center" ? 8 : 5}
              fill="var(--bg-1)"
              stroke={KIND_COLORS[node.kind]}
              strokeWidth="1.5"
            />
            <text
              x={node.x}
              y={node.y + (node.y >= CY ? 16 : -10)}
              textAnchor="middle"
              fill="var(--ink-1)"
              fontSize="8"
              fontFamily="var(--font-jetbrains)"
            >
              {node.label.length > 24
                ? `${node.label.slice(0, 23)}…`
                : node.label}
            </text>
          </g>
        ))}
      </svg>
      <div className="mt-2 flex flex-wrap gap-3">
        {KIND_LEGEND.map(({ kind, label }) => (
          <span
            key={kind}
            className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-wider text-ink-2"
          >
            <span
              className="h-2 w-2 rounded-full border"
              style={{ borderColor: KIND_COLORS[kind] }}
            />
            {label}
          </span>
        ))}
        <span className="flex items-center gap-1.5 font-mono text-[9px] uppercase tracking-wider text-melt">
          <span className="h-px w-4 bg-melt" />
          противоречие
        </span>
      </div>
    </div>
  );
}
