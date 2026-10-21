import type { LoadStatus } from "../api/types";
import { fmtH } from "../lib/format";
import { OVERLOADED_ABOVE, UNDERLOADED_BELOW } from "../lib/load";

export type ChartRow = { name: string; active: number; status: LoadStatus };

const fill = (s: LoadStatus) => (s === "Overloaded" ? "var(--bad)" : s === "Underloaded" ? "var(--warn)" : "var(--ok)");

/** The design's SVG bar chart: balanced band and dashed team-average line. */
export function LoadChart({ rows, average }: { rows: ChartRow[]; average: number }) {
  const W = 720;
  const H = 300;
  const L = 44;
  const Rr = 16;
  const T = 20;
  const B = 48;
  const iw = W - L - Rr;
  const ih = H - T - B;
  const maxV = Math.max(...rows.map((r) => r.active), average * OVERLOADED_ABOVE, 1);
  const top = Math.ceil(maxV / 10) * 10;
  const y = (v: number) => T + ih - (v / top) * ih;
  const bw = iw / Math.max(rows.length, 1);
  const ticks: number[] = [];
  for (let v = 0; v <= top + 1e-9; v += top / 5) ticks.push(v);
  return (
    <svg viewBox={`0 0 ${W} ${H}`} role="img" aria-label="Active hours per member compared to the team average">
      {ticks.map((v) => (
        <g key={v}>
          <line x1={L} x2={W - Rr} y1={y(v)} y2={y(v)} stroke="var(--line)" strokeWidth="1" />
          <text x={L - 8} y={y(v) + 4} textAnchor="end" fontSize="11" fill="var(--muted)" fontFamily="JetBrains Mono,monospace">
            {fmtH(v)}
          </text>
        </g>
      ))}
      <rect
        x={L}
        y={y(average * OVERLOADED_ABOVE)}
        width={iw}
        height={y(average * UNDERLOADED_BELOW) - y(average * OVERLOADED_ABOVE)}
        fill="var(--ok)"
        opacity=".08"
      />
      {rows.map((r, i) => {
        const x = L + i * bw + bw * 0.22;
        const w = bw * 0.56;
        const yy = y(r.active);
        return (
          <g key={i}>
            <rect x={x} y={yy} width={w} height={Math.max(0, T + ih - yy)} rx="5" fill={fill(r.status)} />
            <text x={x + w / 2} y={yy - 7} textAnchor="middle" fontSize="12" fontWeight="700" fill="var(--ink)" fontFamily="JetBrains Mono,monospace">
              {fmtH(r.active)} h
            </text>
            <text x={x + w / 2} y={H - B + 20} textAnchor="middle" fontSize="12.5" fill="var(--ink)" fontWeight="600">
              {r.name}
            </text>
            <text x={x + w / 2} y={H - B + 36} textAnchor="middle" fontSize="11" fill="var(--muted)">
              {r.status}
            </text>
          </g>
        );
      })}
      <line x1={L} x2={W - Rr} y1={y(average)} y2={y(average)} stroke="var(--ink)" strokeWidth="1.5" strokeDasharray="5 4" />
    </svg>
  );
}
