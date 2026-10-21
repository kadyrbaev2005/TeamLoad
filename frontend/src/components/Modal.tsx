import { useEffect, useRef, type FormEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Button } from "./ui";

const FOCUSABLE = 'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),[tabindex]:not([tabindex="-1"])';

type ModalProps = {
  title: string;
  onClose: () => void;
  children: ReactNode;
  /** When set, the modal is a <form> and this runs on submit. */
  onSubmit?: (e: FormEvent<HTMLFormElement>) => void;
};

/** Overlay + modal from the design with a focus trap, Esc and focus restore. */
export function Modal({ title, onClose, children, onSubmit }: ModalProps) {
  const ref = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  });

  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null;
    const el = ref.current;
    const first = el?.querySelector<HTMLElement>("[data-autofocus]") ?? el?.querySelector<HTMLElement>(FOCUSABLE);
    first?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab" || !el) return;
      const items = Array.from(el.querySelectorAll<HTMLElement>(FOCUSABLE));
      if (items.length === 0) return;
      const a = items[0]!;
      const z = items[items.length - 1]!;
      if (e.shiftKey && document.activeElement === a) {
        e.preventDefault();
        z.focus();
      } else if (!e.shiftKey && document.activeElement === z) {
        e.preventDefault();
        a.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("keydown", onKey);
      prev?.focus?.();
    };
  }, []);

  const titleId = "modal-title";
  const body = (
    <>
      <h2 id={titleId}>{title}</h2>
      {children}
    </>
  );
  return createPortal(
    <div className="overlay" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      {onSubmit ? (
        <form
          className="modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          ref={(n) => {
            ref.current = n;
          }}
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit(e);
          }}
          noValidate
        >
          {body}
        </form>
      ) : (
        <div
          className="modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          ref={(n) => {
            ref.current = n;
          }}
        >
          {body}
        </div>
      )}
    </div>,
    document.body,
  );
}

type ConfirmProps = {
  title: string;
  message: ReactNode;
  confirmLabel: string;
  cancelLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmModal({ title, message, confirmLabel, cancelLabel = "Cancel", danger, onConfirm, onCancel }: ConfirmProps) {
  return (
    <Modal title={title} onClose={onCancel}>
      <p>{message}</p>
      <div className="row">
        <Button variant={danger ? "danger" : "primary"} onClick={onConfirm} data-autofocus>
          {confirmLabel}
        </Button>
        <Button onClick={onCancel}>{cancelLabel}</Button>
      </div>
    </Modal>
  );
}
