import type { RaciActivity, RaciValue } from "../api/types";

/** Click order of a RACI cell (spec 6.7). */
export const RACI_ORDER: RaciValue[] = ["", "R", "A", "C", "I"];

export function nextRaci(v: RaciValue): RaciValue {
  return RACI_ORDER[(RACI_ORDER.indexOf(v) + 1) % RACI_ORDER.length]!;
}

/** "No A", "Several A", "No R" or "" (same rule as domain.RaciCheck). */
export function raciCheck(a: RaciActivity, seatIds: string[]): string {
  const v = seatIds.map((id) => a.cells[id] ?? "");
  const A = v.filter((x) => x === "A").length;
  const R = v.filter((x) => x === "R").length;
  if (A !== 1) return A === 0 ? "No A" : "Several A";
  if (R < 1) return "No R";
  return "";
}
