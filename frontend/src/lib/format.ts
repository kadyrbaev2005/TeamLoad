// Formatting helpers shared by every screen.

/** Hours with one decimal max and no trailing ".0" (design fmtH). */
export function fmtH(h: number): string {
  return (Math.round(h * 10) / 10).toString().replace(/\.0$/, "");
}

/** Parses an ISO date (YYYY-MM-DD) as a local calendar date. */
export function parseDate(iso: string): Date {
  return new Date(iso.slice(0, 10) + "T00:00:00");
}

/** "11 Dec 2026" (en-GB, as in the design). */
export function fmtDate(iso: string): string {
  return parseDate(iso).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
}

/** Today's date as YYYY-MM-DD in the browser's time zone. */
export function todayISO(now: Date = new Date()): string {
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

/** Adds days to an ISO date. */
export function addDaysISO(iso: string, days: number): string {
  const d = parseDate(iso);
  d.setDate(d.getDate() + days);
  return todayISO(d);
}

/** Whole days from today to the deadline, never negative. */
export function daysLeft(deadline: string, now: Date = new Date()): number {
  const today = parseDate(todayISO(now));
  return Math.max(0, Math.round((parseDate(deadline).getTime() - today.getTime()) / 864e5));
}

/** "Aruzhan S." for compact person pickers. */
export function shortName(full: string): string {
  const [first, second] = full.trim().split(/\s+/);
  return second ? `${first} ${second[0]}.` : (first ?? full);
}

export function firstName(full: string): string {
  return full.trim().split(/\s+/)[0] ?? full;
}

export function plural(n: number, one: string, many = one + "s"): string {
  return n === 1 ? one : many;
}
