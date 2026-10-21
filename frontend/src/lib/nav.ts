/** Only same-app paths are allowed as ?next= targets. */
export function safeNext(next: string | null): string {
  return next && next.startsWith("/") && !next.startsWith("//") ? next : "/projects";
}
