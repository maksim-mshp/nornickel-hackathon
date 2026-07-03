export type Operator =
  | "eq"
  | "lt"
  | "lte"
  | "gt"
  | "gte"
  | "range"
  | "approx"
  | "from"
  | "to"
  | "pm";

export type Geography = "ru" | "foreign" | "global" | "unknown";

export type ValidationStatus =
  | "machine_extracted"
  | "weak_evidence"
  | "multi_source"
  | "expert_validated"
  | "contradicted";

export type EntityRef = {
  slug: string;
  name: string;
};

export type NumericValue = {
  operator: Operator;
  vmin?: number;
  vmax?: number;
  unit: string;
};

export type Provenance = {
  documentId: string;
  title: string;
  docType: string;
  page: number;
  quote: string;
  year: number;
};

export type ScoreComponents = {
  match: number;
  rerank: number;
  source: number;
  validation: number;
  freshness: number;
};

export type Fact = {
  id: string;
  ref: string;
  subject: EntityRef;
  parameter: EntityRef;
  value: NumericValue;
  si: NumericValue;
  conditions: Record<string, string>;
  geography: Geography;
  provenance: Provenance;
  extractionMethod: "deterministic" | "llm" | "hybrid" | "catalog";
  extractorVersion: string;
  confidence: number;
  validationStatus: ValidationStatus;
  score: number;
  scoreComponents: ScoreComponents;
};

export type ParamConstraint = {
  parameter: EntityRef;
  value: NumericValue;
};

export type QueryPlan = {
  intent:
    | "technology_search"
    | "experiment_search"
    | "literature_review"
    | "expert_search"
    | "gap_analysis"
    | "contradiction_analysis"
    | "comparison"
    | "entity_lookup";
  entities: {
    materials: EntityRef[];
    processes: EntityRef[];
    properties: EntityRef[];
  };
  paramConstraints: ParamConstraint[];
  geography: "any" | "ru" | "foreign" | "compare";
  yearFrom?: number;
  yearTo?: number;
  parser: "llm" | "rules";
  confidence: number;
};

export type ConsensusSource = {
  title: string;
  year: number;
  geography: Geography;
  vmin: number;
  vmax: number;
};

export type Consensus = {
  parameter: EntityRef;
  unit: string;
  verdict: "consensus" | "majority" | "split" | "insufficient";
  agreedMin: number;
  agreedMax: number;
  overlapIndex: number;
  sources: ConsensusSource[];
};

export type Contradiction = {
  id: string;
  aFactRef: string;
  bFactRef: string;
  aStatement: string;
  bStatement: string;
  cause: string;
  confounders: string[];
  status: "judge_confirmed" | "expert_confirmed" | "suspected";
  confidence: number;
};

export type GapCell = {
  label: string;
  score: number;
  reasons: string[] | null;
  neighbors: string[] | null;
};

export type Expert = {
  id: string;
  name: string;
  lab: string;
  weight: number;
  reports: number;
  experiments: number;
  lastYear: number;
};

export type EvidenceStats = {
  sources: number;
  ruSources: number;
  foreignSources: number;
  yearFrom: number;
  yearTo: number;
};

export type EvidencePack = {
  facts: Fact[];
  consensus: Consensus[];
  contradictions: Contradiction[];
  gaps: GapCell[];
  experts: Expert[];
  stats: EvidenceStats;
};

export type GuardReport = {
  numbersChecked: number;
  violations: number;
  degraded: boolean;
};

export type AnswerMethod = {
  name: string;
  applicability: string;
  citations: string[];
};

export type AnswerDoc = {
  summary: string;
  confidence: number;
  methods: AnswerMethod[];
  guard: GuardReport;
};

export type AskEvent =
  | { type: "plan"; plan: QueryPlan }
  | { type: "evidence"; pack: EvidencePack }
  | { type: "answer.delta"; text: string }
  | { type: "answer.done"; answer: AnswerDoc }
  | { type: "error"; message: string };
