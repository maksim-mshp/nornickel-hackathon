const ruRules = new Intl.PluralRules("ru-RU");

export function plural(
  count: number,
  one: string,
  few: string,
  many: string,
): string {
  const category = ruRules.select(count);
  if (category === "one") return one;
  if (category === "few") return few;
  return many;
}

export function pluralCount(
  count: number,
  one: string,
  few: string,
  many: string,
): string {
  return `${count} ${plural(count, one, few, many)}`;
}
