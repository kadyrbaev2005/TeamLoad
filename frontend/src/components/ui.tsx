// Small presentational components ported from the design's markup.
import type { ButtonHTMLAttributes, CSSProperties, HTMLAttributes, ReactNode } from "react";
import { Icon, type IconName } from "./Icon";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "danger" | "ghost";
  size?: "sm";
  icon?: IconName;
};

export function Button({ variant, size, icon, className, children, type = "button", ...rest }: ButtonProps) {
  const cls = ["btn", variant, size, className].filter(Boolean).join(" ");
  return (
    <button type={type} className={cls} {...rest}>
      {icon && <Icon name={icon} />}
      {children}
    </button>
  );
}

export function Panel({ pad, className, children, ...rest }: HTMLAttributes<HTMLDivElement> & { pad?: boolean }) {
  return (
    <div className={["panel", pad && "pad", className].filter(Boolean).join(" ")} {...rest}>
      {children}
    </div>
  );
}

export type ChipTone = "ok" | "warn" | "bad" | "ai" | "line";

export function Chip({ tone, dot, icon, children, className, ...rest }: HTMLAttributes<HTMLSpanElement> & {
  tone?: ChipTone;
  dot?: boolean;
  icon?: IconName;
}) {
  return (
    <span className={["chip", tone, className].filter(Boolean).join(" ")} {...rest}>
      {dot && <span className="dot" />}
      {icon && <Icon name={icon} />}
      {children}
    </span>
  );
}

export type Person = { name: string; initials: string; color: string };

export function Avatar({ person, large }: { person: Person | null | undefined; large?: boolean }) {
  if (!person) return <span className="unassigned" title="Unassigned" />;
  const style: CSSProperties = { background: `var(${person.color})` };
  return (
    <span className={["avatar", large && "lg"].filter(Boolean).join(" ")} style={style} title={person.name}>
      {person.initials}
    </span>
  );
}

export function AvatarStack({ people }: { people: Person[] }) {
  return (
    <div className="avstack">
      {people.map((p, i) => (
        <Avatar key={i} person={p} />
      ))}
    </div>
  );
}

export function Notice({ tone, icon, children }: { tone: "ai" | "warn" | "bad"; icon?: IconName; children: ReactNode }) {
  return (
    <div className={`notice ${tone}`} role={tone === "bad" ? "alert" : undefined}>
      <Icon name={icon ?? (tone === "ai" ? "spark" : "warn")} />
      <div>{children}</div>
    </div>
  );
}

export function Swatch({ color, size }: { color: string; size?: number }) {
  const style: CSSProperties = { background: `var(${color})` };
  if (size) Object.assign(style, { width: size, height: size });
  return <span className="swatch" style={style} />;
}

export function PageHead({ eyebrow, title, sub, actions }: { eyebrow: string; title: ReactNode; sub?: ReactNode; actions?: ReactNode }) {
  return (
    <div className="pagehead">
      <div className="t">
        <span className="eyebrow">{eyebrow}</span>
        <h1>{title}</h1>
        {sub && <p className="muted">{sub}</p>}
      </div>
      <div className="row">{actions}</div>
    </div>
  );
}

export function EffortBar({ parts, label }: { parts: { pct: number; color: string; title?: string }[]; label: string }) {
  return (
    <div className="effort" role="img" aria-label={label}>
      {parts.map((p, i) => (
        <span key={i} style={{ width: `${p.pct}%`, background: p.color }} title={p.title} />
      ))}
    </div>
  );
}

export function Legend({ children }: { children: ReactNode }) {
  return <div className="legend">{children}</div>;
}

export function FitBar({ value }: { value: number }) {
  return (
    <div className="fitbar">
      <i style={{ width: `${value}%` }} />
    </div>
  );
}

export function Skeleton({ kind = "line", style }: { kind?: "line" | "title" | "block"; style?: CSSProperties }) {
  return <div className={`skel ${kind}`} style={style} aria-hidden="true" />;
}

/** Page placeholder shown while a query loads. */
export function PageSkeleton({ blocks = 2 }: { blocks?: number }) {
  return (
    <div className="stack" aria-busy="true" aria-label="Loading">
      <Skeleton kind="line" style={{ width: 90 }} />
      <Skeleton kind="title" />
      <Skeleton kind="line" style={{ width: "min(460px,90%)" }} />
      <div className="stack" style={{ marginTop: 10 }}>
        {Array.from({ length: blocks }, (_, i) => (
          <Skeleton key={i} kind="block" />
        ))}
      </div>
    </div>
  );
}
