// Remembers the last opened project so the sidebar can show it on
// /projects and /profile (per-browser convenience only).
const KEY = "teamload.currentProject";

export function rememberedProject(): string | undefined {
  try {
    return localStorage.getItem(KEY) ?? undefined;
  } catch {
    return undefined;
  }
}

export function rememberProject(id: string) {
  try {
    localStorage.setItem(KEY, id);
  } catch {
    /* storage unavailable */
  }
}

export function forgetProject() {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* storage unavailable */
  }
}
