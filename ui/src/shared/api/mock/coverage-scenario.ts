export type CoverageKpi = {
  label: string;
  value: number;
  trend: number[];
};

export type CoverageCell = {
  material: string;
  process: string;
  score: number;
  facts: number;
  experiments: number;
  reasons: string[];
};

export type RiskItem = {
  label: string;
  detail: string;
  question: string;
};

export const COVERAGE_KPIS: CoverageKpi[] = [
  { label: "документов", value: 412, trend: [310, 342, 361, 380, 412] },
  { label: "подтверждённых фактов", value: 8931, trend: [5200, 6100, 7400, 8200, 8931] },
  { label: "противоречий", value: 47, trend: [21, 28, 35, 44, 47] },
  { label: "пробелов", value: 63, trend: [88, 81, 74, 69, 63] },
];

export const COVERAGE_MATERIALS = [
  "никелевый штейн",
  "медный концентрат",
  "католит",
  "шахтная вода",
  "никелевая руда",
  "шлак",
];

export const COVERAGE_PROCESSES = [
  "электроэкстракция",
  "флотация",
  "выщелачивание",
  "обжиг",
  "обессоливание",
];

export const COVERAGE_CELLS: CoverageCell[] = [
  { material: "никелевый штейн", process: "электроэкстракция", score: 86, facts: 468, experiments: 38, reasons: [] },
  { material: "никелевый штейн", process: "флотация", score: 71, facts: 234, experiments: 21, reasons: [] },
  { material: "никелевый штейн", process: "выщелачивание", score: 64, facts: 187, experiments: 14, reasons: [] },
  { material: "никелевый штейн", process: "обжиг", score: 58, facts: 121, experiments: 9, reasons: ["устарело"] },
  { material: "никелевый штейн", process: "обессоливание", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "медный концентрат", process: "электроэкстракция", score: 78, facts: 356, experiments: 29, reasons: [] },
  { material: "медный концентрат", process: "флотация", score: 88, facts: 502, experiments: 44, reasons: [] },
  { material: "медный концентрат", process: "выщелачивание", score: 55, facts: 143, experiments: 11, reasons: [] },
  { material: "медный концентрат", process: "обжиг", score: 62, facts: 168, experiments: 13, reasons: [] },
  { material: "медный концентрат", process: "обессоливание", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "католит", process: "электроэкстракция", score: 74, facts: 289, experiments: 24, reasons: ["противоречия"] },
  { material: "католит", process: "флотация", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "католит", process: "выщелачивание", score: 12, facts: 18, experiments: 1, reasons: ["только зарубежные"] },
  { material: "католит", process: "обжиг", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "католит", process: "обессоливание", score: 31, facts: 44, experiments: 3, reasons: ["устарело"] },
  { material: "шахтная вода", process: "электроэкстракция", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "шахтная вода", process: "флотация", score: 22, facts: 31, experiments: 2, reasons: ["устарело"] },
  { material: "шахтная вода", process: "выщелачивание", score: 18, facts: 26, experiments: 2, reasons: ["только зарубежные"] },
  { material: "шахтная вода", process: "обжиг", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "шахтная вода", process: "обессоливание", score: 67, facts: 176, experiments: 15, reasons: [] },
  { material: "никелевая руда", process: "электроэкстракция", score: 41, facts: 87, experiments: 6, reasons: [] },
  { material: "никелевая руда", process: "флотация", score: 79, facts: 341, experiments: 31, reasons: [] },
  { material: "никелевая руда", process: "выщелачивание", score: 26, facts: 39, experiments: 3, reasons: ["только зарубежные", "холодный климат не изучен"] },
  { material: "никелевая руда", process: "обжиг", score: 69, facts: 201, experiments: 17, reasons: [] },
  { material: "никелевая руда", process: "обессоливание", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "шлак", process: "электроэкстракция", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
  { material: "шлак", process: "флотация", score: 48, facts: 98, experiments: 8, reasons: [] },
  { material: "шлак", process: "выщелачивание", score: 53, facts: 117, experiments: 10, reasons: [] },
  { material: "шлак", process: "обжиг", score: 36, facts: 61, experiments: 4, reasons: ["устарело"] },
  { material: "шлак", process: "обессоливание", score: 0, facts: 0, experiments: 0, reasons: ["нет экспериментов"] },
];

export const COVERAGE_RISKS: {
  contradictory: RiskItem[];
  outdated: RiskItem[];
  foreignOnly: RiskItem[];
} = {
  contradictory: [
    {
      label: "циркуляция католита × плотность тока",
      detail: "2 подтверждённых противоречия, конфаундер: плотность тока",
      question:
        "Какая скорость циркуляции католита оптимальна при разной плотности тока?",
    },
    {
      label: "флотация медного концентрата × реагенты",
      detail: "конфликт диапазонов расхода ксантогената",
      question: "Какой расход ксантогената оптимален при флотации медного концентрата?",
    },
  ],
  outdated: [
    {
      label: "обжиг никелевого штейна",
      detail: "последние данные — 2016 год",
      question: "Какие режимы обжига никелевого штейна описаны в литературе?",
    },
    {
      label: "обессоливание католита",
      detail: "последние данные — 2017 год",
      question: "Какие методы обессоливания католита применялись?",
    },
  ],
  foreignOnly: [
    {
      label: "кучное выщелачивание никелевой руды",
      detail: "12 источников, все зарубежные, холодный климат не изучен",
      question: "Есть ли данные по кучному выщелачиванию никелевой руды в холодном климате?",
    },
    {
      label: "выщелачивание католита",
      detail: "3 источника, все зарубежные",
      question: "Какие методы выщелачивания католита описаны в мировой практике?",
    },
  ],
};
