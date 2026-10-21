import { useId, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from "react";

type FieldProps = {
  label: ReactNode;
  hint?: ReactNode;
  error?: string;
  /** Rendered on the right of the hint row (e.g. a character counter). */
  aside?: ReactNode;
  children: (props: { id: string; "aria-invalid"?: boolean; "aria-describedby"?: string }) => ReactNode;
};

/** Labelled form field with hint and inline error, as in the design. */
export function Field({ label, hint, error, aside, children }: FieldProps) {
  const id = useId();
  const hintId = `${id}-hint`;
  const errId = `${id}-err`;
  const describedBy = [hint ? hintId : "", error ? errId : ""].filter(Boolean).join(" ") || undefined;
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      {children({ id, "aria-invalid": error ? true : undefined, "aria-describedby": describedBy })}
      {(hint || aside) &&
        (aside ? (
          <div className="row" style={{ justifyContent: "space-between" }}>
            <span className="hint" id={hintId}>
              {hint}
            </span>
            {aside}
          </div>
        ) : (
          <span className="hint" id={hintId}>
            {hint}
          </span>
        ))}
      {error && (
        <span className="ferr" id={errId}>
          {error}
        </span>
      )}
    </div>
  );
}

export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input className="input" {...props} />;
}

export function Textarea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className="textarea" {...props} />;
}

/** "N / max" counter; red while below min. */
export function Counter({ value, max, min, suffix }: { value: string; max: number; min?: number; suffix?: string }) {
  const n = value.length;
  const bad = min !== undefined && n < min;
  return (
    <span className={`counter mono${bad ? " bad" : ""}`} aria-live="polite">
      {n} / {max}
      {suffix}
    </span>
  );
}
