import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";

type ToastFn = (message: string) => void;

const ToastContext = createContext<ToastFn>(() => {});

/** Shows the design's bottom toast for 2.4 s; one at a time. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [msg, setMsg] = useState<{ text: string; key: number } | null>(null);
  const timer = useRef<number | undefined>(undefined);

  const show = useCallback<ToastFn>((text) => {
    window.clearTimeout(timer.current);
    setMsg({ text, key: Date.now() });
    timer.current = window.setTimeout(() => setMsg(null), 2400);
  }, []);

  useEffect(() => () => window.clearTimeout(timer.current), []);

  return (
    <ToastContext.Provider value={show}>
      {children}
      <div aria-live="polite">
        {msg && (
          <div className="toast" role="status" key={msg.key}>
            {msg.text}
          </div>
        )}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastFn {
  return useContext(ToastContext);
}

/** Copies text and toasts the result, as the design does. */
export function useCopy(): (text: string) => void {
  const toast = useToast();
  return useCallback(
    (text: string) => {
      const fallback = () => toast("Copy this: " + text);
      try {
        navigator.clipboard.writeText(text).then(() => toast("Copied: " + text), fallback);
      } catch {
        fallback();
      }
    },
    [toast],
  );
}
