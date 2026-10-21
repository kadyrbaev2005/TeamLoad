import { useState, type KeyboardEvent } from "react";

const MAX_TAGS = 10;
const MAX_LEN = 24;

type Props = {
  id: string;
  tags: string[];
  onChange: (tags: string[]) => void;
  suggestions?: string[];
  describedBy?: string;
};

/** Chip input in the design's .tagbox / .tag / .suggest style. */
export function TagInput({ id, tags, onChange, suggestions = [], describedBy }: Props) {
  const [draft, setDraft] = useState("");
  const has = (t: string) => tags.some((x) => x.toLowerCase() === t.toLowerCase());
  const add = (raw: string) => {
    const t = raw.trim().slice(0, MAX_LEN);
    if (!t || has(t) || tags.length >= MAX_TAGS) return;
    onChange([...tags, t]);
  };
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      add(draft);
      setDraft("");
    } else if (e.key === "Backspace" && draft === "" && tags.length) {
      onChange(tags.slice(0, -1));
    }
  };
  const left = suggestions.filter((s) => !has(s)).slice(0, 6);
  return (
    <>
      <div className="tagbox">
        {tags.map((t) => (
          <span className="tag" key={t}>
            {t}
            <button type="button" aria-label={`Remove ${t}`} onClick={() => onChange(tags.filter((x) => x !== t))}>
              ×
            </button>
          </span>
        ))}
        <input
          id={id}
          value={draft}
          maxLength={MAX_LEN}
          placeholder={tags.length >= MAX_TAGS ? "10 tags maximum" : "Add a skill and press Enter"}
          disabled={tags.length >= MAX_TAGS}
          onChange={(e) => setDraft(e.target.value.replace(",", ""))}
          onKeyDown={onKey}
          onBlur={() => {
            add(draft);
            setDraft("");
          }}
          aria-describedby={describedBy}
        />
      </div>
      {left.length > 0 && tags.length < MAX_TAGS && (
        <div className="suggest" aria-label="Suggested tags">
          {left.map((s) => (
            <button type="button" key={s} onClick={() => add(s)}>
              + {s}
            </button>
          ))}
        </div>
      )}
    </>
  );
}
