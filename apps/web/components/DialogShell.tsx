"use client";

import { type ReactNode, useEffect, useRef } from "react";
import { createPortal } from "react-dom";

const focusableSelector = [
  "a[href]",
  "button:not([disabled])",
  "textarea:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
].join(",");

let openDialogCount = 0;
let previousBodyOverflow = "";
let previousBodyPaddingRight = "";

function lockBodyScroll() {
  if (typeof document === "undefined") return;
  if (openDialogCount === 0) {
    previousBodyOverflow = document.body.style.overflow;
    previousBodyPaddingRight = document.body.style.paddingRight;
    const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth;
    document.body.style.overflow = "hidden";
    if (scrollbarWidth > 0) {
      document.body.style.paddingRight = `${scrollbarWidth}px`;
    }
  }
  openDialogCount += 1;
}

function unlockBodyScroll() {
  if (typeof document === "undefined" || openDialogCount === 0) return;
  openDialogCount -= 1;
  if (openDialogCount > 0) return;
  document.body.style.overflow = previousBodyOverflow;
  document.body.style.paddingRight = previousBodyPaddingRight;
}

function focusTopMostDialog() {
  const dialogs = Array.from(
    document.querySelectorAll<HTMLElement>('section[aria-modal="true"]'),
  );
  const panel = dialogs[dialogs.length - 1];
  if (!panel) return;
  const initialFocus =
    panel.querySelector<HTMLElement>("[data-dialog-initial-focus]") ??
    getFocusableElements(panel)[0] ??
    panel;
  initialFocus.focus({ preventScroll: true });
}

type DialogShellProps = {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  rootClassName: string;
  backdropClassName: string;
  panelClassName: string;
  role?: "dialog" | "alertdialog";
  ariaLabel?: string;
  ariaLabelledBy?: string;
  ariaDescribedBy?: string;
  zIndex?: number;
  closeOnBackdrop?: boolean;
  closeOnEscape?: boolean;
};

export function DialogShell({
  open,
  onClose,
  children,
  rootClassName,
  backdropClassName,
  panelClassName,
  role = "dialog",
  ariaLabel,
  ariaLabelledBy,
  ariaDescribedBy,
  zIndex,
  closeOnBackdrop = true,
  closeOnEscape = true,
}: DialogShellProps) {
  const panelRef = useRef<HTMLElement | null>(null);
  const onCloseRef = useRef(onClose);

  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  useEffect(() => {
    if (!open) return;
    const previousActiveElement =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;
    lockBodyScroll();
    const focusHandle = window.setTimeout(() => {
      const panel = panelRef.current;
      const initialFocus =
        panel?.querySelector<HTMLElement>("[data-dialog-initial-focus]") ??
        getFocusableElements(panel)[0] ??
        panel;
      initialFocus?.focus({ preventScroll: true });
    }, 0);

    return () => {
      window.clearTimeout(focusHandle);
      unlockBodyScroll();
      if (openDialogCount > 0) {
        window.setTimeout(() => {
          focusTopMostDialog();
        }, 0);
        return;
      }
      if (previousActiveElement && document.contains(previousActiveElement)) {
        previousActiveElement.focus({ preventScroll: true });
      }
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && closeOnEscape) {
        event.preventDefault();
        onCloseRef.current();
        return;
      }

      if (event.key !== "Tab") return;
      const focusableElements = getFocusableElements(panelRef.current);
      if (focusableElements.length === 0) {
        event.preventDefault();
        panelRef.current?.focus({ preventScroll: true });
        return;
      }

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];
      const activeElement = document.activeElement;

      if (event.shiftKey && (activeElement === firstElement || activeElement === panelRef.current)) {
        event.preventDefault();
        lastElement.focus({ preventScroll: true });
        return;
      }

      if (!event.shiftKey && activeElement === lastElement) {
        event.preventDefault();
        firstElement.focus({ preventScroll: true });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [closeOnEscape, open]);

  if (!open || typeof document === "undefined") return null;

  return createPortal(
    <div
      className={rootClassName}
      role="presentation"
      onClick={closeOnBackdrop ? onClose : undefined}
      style={zIndex !== undefined ? { zIndex } : undefined}
    >
      <div className={backdropClassName} />
      <section
        ref={panelRef}
        className={panelClassName}
        role={role}
        aria-modal="true"
        aria-label={ariaLabel}
        aria-labelledby={ariaLabelledBy}
        aria-describedby={ariaDescribedBy}
        tabIndex={-1}
        onClick={(event) => event.stopPropagation()}
      >
        {children}
      </section>
    </div>,
    document.body,
  );
}

function getFocusableElements(root: HTMLElement | null) {
  if (!root) return [];
  return Array.from(root.querySelectorAll<HTMLElement>(focusableSelector)).filter((element) => {
    const disabled =
      element.matches(":disabled") || element.getAttribute("aria-disabled") === "true";
    const hidden =
      element.hidden ||
      element.getAttribute("aria-hidden") === "true" ||
      window.getComputedStyle(element).visibility === "hidden";
    return !disabled && !hidden;
  });
}
