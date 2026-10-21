// Workload thresholds shared with the Go domain package (one constants file).
import thresholds from "@shared/load_thresholds.json";
import type { LoadStatus } from "../api/types";

export const OVERLOADED_ABOVE: number = thresholds.overloadedAbove;
export const UNDERLOADED_BELOW: number = thresholds.underloadedBelow;

/** "75–130%" for the band subtitle and chart. */
export const BAND_LABEL = `${Math.round(UNDERLOADED_BELOW * 100)}–${Math.round(OVERLOADED_ABOVE * 100)}%`;

/** Same rule as domain.LoadStatus (overloaded strictly above the upper bound). */
export function loadStatus(active: number, average: number): LoadStatus {
  if (average <= 0) return "Balanced";
  const eps = 1e-9;
  if (active > average * OVERLOADED_ABOVE + eps) return "Overloaded";
  if (active < average * UNDERLOADED_BELOW - eps) return "Underloaded";
  return "Balanced";
}
