"use client";

import { useEffect, useId, useState } from "react";
import { CircleHelp, Loader2, Trash2, TriangleAlert, X } from "lucide-react";
import { DialogShell } from "@/components/DialogShell";

export type ConfirmDialogTone = "default" | "warning" | "danger";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  message: string;
  details?: string[];
  confirmLabel?: string;
  cancelLabel?: string;
  pendingLabel?: string;
  tone?: ConfirmDialogTone;
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
};

export function ConfirmDialog({
  open,
  title,
  message,
  details = [],
  confirmLabel = "确认",
  cancelLabel = "取消",
  pendingLabel = "处理中...",
  tone = "default",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const titleId = useId();
  const messageId = useId();
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState("");

  useEffect(() => {
    if (!open) {
      setPending(false);
      setActionError("");
    }
  }, [open]);

  const handleConfirm = async () => {
    if (pending) return;
    setPending(true);
    setActionError("");
    try {
      await onConfirm();
      setPending(false);
      onCancel();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "操作失败，请稍后重试。");
      setPending(false);
    }
  };
  const Icon = tone === "danger" ? Trash2 : tone === "warning" ? TriangleAlert : CircleHelp;

  return (
    <DialogShell
      open={open}
      onClose={onCancel}
      rootClassName="confirm-modal"
      backdropClassName="confirm-modal-backdrop"
      panelClassName={`confirm-modal-content confirm-modal-content--${tone}`}
      role="alertdialog"
      ariaLabelledBy={titleId}
      ariaDescribedBy={messageId}
      zIndex={90}
      closeOnBackdrop={!pending}
      closeOnEscape={!pending}
    >
      <div className="confirm-modal-head">
        <div className="confirm-modal-title-row">
          <span className="confirm-modal-icon" data-tone={tone}>
            <Icon size={20} />
          </span>
          <div className="confirm-modal-copy">
            <h2 id={titleId}>{title}</h2>
            <p id={messageId}>{message}</p>
          </div>
        </div>
        <button
          type="button"
          className="icon-button"
          onClick={onCancel}
          aria-label="关闭确认框"
          disabled={pending}
        >
          <X size={16} />
        </button>
      </div>
      <div className="confirm-modal-body">
        {details.length > 0 ? (
          <div className="confirm-modal-details">
            {details.map((detail, index) => (
              <span key={`${detail}-${index}`}>{detail}</span>
            ))}
          </div>
        ) : null}
        {actionError ? (
          <div className="confirm-modal-error" role="alert">
            {actionError}
          </div>
        ) : null}
      </div>
      <div className="confirm-modal-actions">
        <button
          type="button"
          className="btn-secondary"
          onClick={onCancel}
          disabled={pending}
          data-dialog-initial-focus
        >
          {cancelLabel}
        </button>
        <button
          type="button"
          className={tone === "danger" ? "btn-danger" : "btn-primary"}
          onClick={handleConfirm}
          disabled={pending}
        >
          {pending ? (
            <Loader2 size={16} className="dialog-spin" style={{ marginRight: 6 }} />
          ) : tone === "danger" ? (
            <Trash2 size={16} style={{ marginRight: 6 }} />
          ) : null}
          {pending ? pendingLabel : confirmLabel}
        </button>
      </div>
    </DialogShell>
  );
}
