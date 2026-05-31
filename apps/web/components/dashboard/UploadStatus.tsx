"use client";

import type { UploadState } from "@/components/dashboard/types";

type UploadStatusProps = {
  state: UploadState;
  hint: string;
};

function uploadPhaseLabel(phase: UploadState["phase"]) {
  switch (phase) {
    case "uploading":
      return "上传中";
    case "processing":
      return "处理中";
    case "done":
      return "已导入";
    case "failed":
      return "导入失败";
    default:
      return "等待导入";
  }
}

function uploadTone(phase: UploadState["phase"]) {
  switch (phase) {
    case "done":
      return "good";
    case "failed":
      return "bad";
    case "uploading":
    case "processing":
      return "warn";
    default:
      return "accent";
  }
}

function formatBytes(value: number) {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** index;
  return `${amount >= 10 || index === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[index]}`;
}

export function UploadStatus({ state, hint }: UploadStatusProps) {
  if (state.phase === "idle") return null;

  const progress = state.totalBytes > 0 ? state.percent : 0;
  return (
    <section className="upload-status" data-phase={state.phase}>
      <div className="upload-status-head">
        <div>
          <div className="panel-title">导入状态</div>
          <div className="panel-subtitle">{state.message || hint}</div>
        </div>
        <span className="pill" data-tone={uploadTone(state.phase)}>
          {uploadPhaseLabel(state.phase)}
        </span>
      </div>
      <div className="progress-block">
        <div className="progress-label">
          <span>
            {state.fileCount} 张 · {formatBytes(state.loadedBytes)} /{" "}
            {formatBytes(state.totalBytes || state.loadedBytes)}
          </span>
          <strong>{progress}%</strong>
        </div>
        <div className="progress-track">
          <span style={{ width: `${Math.min(progress, 100)}%` }} />
        </div>
      </div>
      {state.fileNames.length > 0 ? (
        <div className="tiny-row">
          {state.fileNames.map((name, index) => (
            <span className="pill" data-tone="accent" key={`${name}-${index}`}>
              {name}
            </span>
          ))}
        </div>
      ) : null}
    </section>
  );
}
