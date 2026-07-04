export function formatNumber(value: number, maxFractionDigits = 4): string {
  return new Intl.NumberFormat("ru-RU", {
    maximumFractionDigits: maxFractionDigits,
  }).format(value);
}
