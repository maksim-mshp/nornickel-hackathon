import type { Consensus, Expert } from "@/shared/api/types";
import { CATHOLYTE_PACK } from "@/shared/api/mock/catholyte-scenario";

export type EntityRelation = {
  group: string;
  items: { slug: string; name: string; weight: number }[];
};

export type EntityTimelinePoint = {
  year: number;
  facts: number;
};

export type EntityCard = {
  slug: string;
  type: string;
  nameRu: string;
  nameEn: string;
  synonyms: { value: string; pending: boolean }[];
  counters: { documents: number; facts: number; experiments: number; experts: number };
  consensus: Consensus[];
  relations: EntityRelation[];
  experts: Expert[];
  timeline: EntityTimelinePoint[];
};

const NICKEL_ELECTROWINNING: EntityCard = {
  slug: "process:nickel-electrowinning",
  type: "процесс",
  nameRu: "электроэкстракция никеля",
  nameEn: "nickel electrowinning",
  synonyms: [
    { value: "электроэкстракция Ni", pending: false },
    { value: "ЭЭ никеля", pending: false },
    { value: "nickel EW", pending: true },
  ],
  counters: { documents: 87, facts: 1243, experiments: 64, experts: 9 },
  consensus: CATHOLYTE_PACK.consensus,
  relations: [
    {
      group: "материалы",
      items: [
        { slug: "material:catholyte", name: "католит", weight: 0.9 },
        { slug: "material:nickel-matte", name: "никелевый штейн", weight: 0.7 },
        { slug: "material:anolyte", name: "анолит", weight: 0.5 },
      ],
    },
    {
      group: "оборудование",
      items: [
        { slug: "equipment:diaphragm-cell", name: "диафрагменная ячейка", weight: 0.8 },
        { slug: "equipment:collector", name: "распределительный коллектор", weight: 0.6 },
      ],
    },
    {
      group: "параметры",
      items: [
        { slug: "parameter:catholyte-flow-rate", name: "скорость циркуляции католита", weight: 0.9 },
        { slug: "parameter:current-density", name: "плотность тока", weight: 0.8 },
        { slug: "parameter:temperature", name: "температура электролита", weight: 0.7 },
      ],
    },
  ],
  experts: CATHOLYTE_PACK.experts,
  timeline: [
    { year: 2018, facts: 84 },
    { year: 2019, facts: 121 },
    { year: 2020, facts: 96 },
    { year: 2021, facts: 178 },
    { year: 2022, facts: 203 },
    { year: 2023, facts: 312 },
    { year: 2024, facts: 164 },
    { year: 2025, facts: 85 },
  ],
};

const ENTITIES: Record<string, EntityCard> = {
  [NICKEL_ELECTROWINNING.slug]: NICKEL_ELECTROWINNING,
};

export function getEntity(slug: string): EntityCard | null {
  return ENTITIES[slug] ?? null;
}
