// The design's SVG icon set (design/design.html, const I).
import type { ReactElement } from "react";

const stroke = { fill: "none", stroke: "currentColor", "aria-hidden": true } as const;

const icons = {
  logo: (
    <svg width="26" height="26" viewBox="0 0 26 26" aria-hidden="true">
      <rect x="1" y="14" width="6" height="11" rx="2" fill="var(--accent)" />
      <rect x="10" y="8" width="6" height="17" rx="2" fill="currentColor" />
      <rect x="19" y="3" width="6" height="22" rx="2" fill="var(--accent)" opacity=".45" />
    </svg>
  ),
  spark: (
    <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true">
      <path fill="currentColor" d="M8 0l1.7 5.1L15 7l-5.3 1.9L8 14l-1.7-5.1L1 7l5.3-1.9zM13.5 10.5l.7 1.8 1.8.7-1.8.7-.7 1.8-.7-1.8-1.8-.7 1.8-.7z" />
    </svg>
  ),
  plan: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <rect x="2" y="2" width="6" height="6" rx="1.5" />
      <rect x="10" y="2" width="6" height="6" rx="1.5" />
      <rect x="2" y="10" width="6" height="6" rx="1.5" />
      <rect x="10" y="10" width="6" height="6" rx="1.5" />
    </svg>
  ),
  roles: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <circle cx="6.5" cy="6" r="2.6" />
      <path d="M1.8 15.5c.6-2.8 2.4-4.2 4.7-4.2s4.1 1.4 4.7 4.2" />
      <circle cx="13" cy="6.5" r="2.1" />
      <path d="M12.5 10.8c1.9 0 3.3 1.2 3.8 3.5" />
    </svg>
  ),
  raci: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <rect x="2" y="2.5" width="14" height="13" rx="2" />
      <path d="M2 7h14M2 11h14M7 2.5v13" />
    </svg>
  ),
  board: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <rect x="2" y="2.5" width="4" height="13" rx="1.2" />
      <rect x="7.5" y="2.5" width="4" height="9" rx="1.2" />
      <rect x="13" y="2.5" width="3" height="6" rx="1.2" />
    </svg>
  ),
  dash: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <path d="M2 16h14" />
      <path d="M4.5 13V8M9 13V4M13.5 13V9.5" />
    </svg>
  ),
  gear: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <circle cx="9" cy="9" r="2.4" />
      <path d="M9 1.8v2M9 14.2v2M1.8 9h2M14.2 9h2M3.9 3.9l1.4 1.4M12.7 12.7l1.4 1.4M3.9 14.1l1.4-1.4M12.7 5.3l1.4-1.4" />
    </svg>
  ),
  user: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <circle cx="9" cy="6" r="3" />
      <path d="M3 16c.7-3.2 3-4.8 6-4.8s5.3 1.6 6 4.8" />
    </svg>
  ),
  folder: (
    <svg width="18" height="18" viewBox="0 0 18 18" strokeWidth="1.6" {...stroke}>
      <path d="M2 5.5a1.5 1.5 0 0 1 1.5-1.5h3.3l1.7 1.8h6A1.5 1.5 0 0 1 16 7.3v6.2A1.5 1.5 0 0 1 14.5 15h-11A1.5 1.5 0 0 1 2 13.5z" />
    </svg>
  ),
  plus: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.8" {...stroke}>
      <path d="M8 3v10M3 8h10" />
    </svg>
  ),
  copy: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.6" {...stroke}>
      <rect x="5" y="5" width="9" height="9" rx="1.8" />
      <path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2h-6A1.5 1.5 0 0 0 2 3.5v6A1.5 1.5 0 0 0 3.5 11H5" />
    </svg>
  ),
  check: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="2" {...stroke}>
      <path d="M3 8.5l3.2 3L13 4.5" />
    </svg>
  ),
  warn: (
    <svg width="18" height="18" viewBox="0 0 18 18" aria-hidden="true">
      <path fill="currentColor" d="M9 1.5l8 14H1zM8.2 7v4.4h1.6V7zm0 5.6v1.6h1.6v-1.6z" />
    </svg>
  ),
  refresh: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.7" {...stroke}>
      <path d="M13.5 6.5A5.5 5.5 0 0 0 3 5M2.5 9.5A5.5 5.5 0 0 0 13 11" />
      <path d="M3 1.8V5h3.2M13 14.2V11H9.8" />
    </svg>
  ),
  pen: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.6" {...stroke}>
      <path d="M10.8 2.7l2.5 2.5L5.5 13H3v-2.5z" />
    </svg>
  ),
  cal: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.6" {...stroke}>
      <rect x="2" y="3" width="12" height="11" rx="1.8" />
      <path d="M2 6.5h12M5.5 1.5v3M10.5 1.5v3" />
    </svg>
  ),
  out: (
    <svg width="16" height="16" viewBox="0 0 16 16" strokeWidth="1.6" {...stroke}>
      <path d="M6 2.5H3.5A1.5 1.5 0 0 0 2 4v8a1.5 1.5 0 0 0 1.5 1.5H6M10.5 11l3-3-3-3M13.5 8H6" />
    </svg>
  ),
} satisfies Record<string, ReactElement>;

export type IconName = keyof typeof icons;

export function Icon({ name }: { name: IconName }) {
  return icons[name];
}
