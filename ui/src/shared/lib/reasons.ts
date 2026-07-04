const REASON_LABELS: Record<string, string> = {
  no_experiments: "нет экспериментов",
  no_ru_practice: "нет отечественной практики",
  no_foreign_practice: "нет зарубежной практики",
  foreign_only: "только зарубежные данные",
  stale: "устаревшие данные",
  low_validation: "слабая валидация",
  no_experts: "нет экспертов",
  contradictory: "есть противоречия",
};

export function reasonLabel(code: string): string {
  return REASON_LABELS[code] ?? code.replace(/_/g, " ");
}
