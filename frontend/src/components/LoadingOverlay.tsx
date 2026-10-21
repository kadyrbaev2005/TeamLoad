import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Icon } from "./Icon";

const STEP_MS = 700;
const CLOSE_MS = 450;

type State = { title: string; steps: string[]; done: number } | null;

function Overlay({ title, steps, done }: { title: string; steps: string[]; done: number }) {
  return createPortal(
    <div className="overlay">
      <div className="loading" role="status" aria-live="polite">
        <div className="spin" />
        <h3>{title}</h3>
        <ul className="steps">
          {steps.map((s, i) => (
            <li key={i} className={i < done ? "done" : undefined}>
              <Icon name="check" />
              <span>{s}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>,
    document.body,
  );
}

/**
 * Runs an AI request behind the design's loading card. Steps tick on a timer
 * while the request runs (all but the last); when the response arrives every
 * step is ticked and the card closes. Errors close the card and are rethrown.
 */
export function useAiOverlay(): {
  run: <T>(title: string, steps: string[], fn: () => Promise<T>) => Promise<T>;
  overlay: ReactNode;
} {
  const [state, setState] = useState<State>(null);
  const timers = useRef<number[]>([]);

  const clear = () => {
    timers.current.forEach((t) => window.clearTimeout(t));
    timers.current = [];
  };
  useEffect(() => clear, []);

  const run = useCallback(async <T,>(title: string, steps: string[], fn: () => Promise<T>): Promise<T> => {
    clear();
    setState({ title, steps, done: 0 });
    for (let i = 1; i < steps.length; i++) {
      timers.current.push(window.setTimeout(() => setState((s) => (s ? { ...s, done: Math.max(s.done, i) } : s)), STEP_MS * i));
    }
    try {
      const res = await fn();
      clear();
      setState((s) => (s ? { ...s, done: steps.length } : s));
      await new Promise((r) => window.setTimeout(r, CLOSE_MS));
      setState(null);
      return res;
    } catch (e) {
      clear();
      setState(null);
      throw e;
    }
  }, []);

  return { run, overlay: state ? <Overlay {...state} /> : null };
}
