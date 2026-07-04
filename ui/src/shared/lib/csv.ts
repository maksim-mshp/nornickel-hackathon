const BOM = "﻿";

function escapeCell(value: string): string {
  let cell = value;
  if (/^[=+\-@\t\r]/.test(cell)) cell = `'${cell}`;
  if (/["\n\r;,]/.test(cell)) cell = `"${cell.replace(/"/g, '""')}"`;
  return cell;
}

export function toCsv(rows: (string | number)[][], delimiter = ";"): string {
  const body = rows
    .map((row) => row.map((cell) => escapeCell(String(cell))).join(delimiter))
    .join("\r\n");
  return `${BOM}${body}`;
}
