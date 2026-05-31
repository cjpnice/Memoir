"use client";

import { type ChangeEvent, type DragEvent, useState } from "react";
import { Film, Layers3, Sparkles, Trash2, Upload, X } from "lucide-react";
import { DialogShell } from "@/components/DialogShell";
import { UploadStatus } from "@/components/dashboard/UploadStatus";
import type { UploadState } from "@/components/dashboard/types";
import type { ImageAsset, Project } from "@/lib/types";

type ThemeOption = {
  value: string;
  label: string;
};

type ProjectSetupPanelProps = {
  project: Project;
  themeOptions: ThemeOption[];
  busy: string;
  uploadAccept: string;
  isUploading: boolean;
  uploadState: UploadState;
  uploadHint: string;
  analysisStatusLabel: string;
  analysisProgress: number;
  staleAnalysisCount: number;
  stats: {
    count: number;
    keep: number;
    improve: number;
  };
  pendingAnalysisCount: number;
  canAnalyze: boolean;
  recentImages: ImageAsset[];
  allImages: ImageAsset[];
  isDeletingImage: boolean;
  onDrop: (event: DragEvent<HTMLDivElement>) => void;
  onUploadInputChange: (event: ChangeEvent<HTMLInputElement>) => void;
  onThemeChange: (value: string) => void;
  onAnalyze: () => void;
  onGoReview: () => void;
  onPreviewImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
};

function formatCount(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function imageStatusLabel(status?: string) {
  switch (status) {
    case "uploaded":
      return "已导入";
    case "analyzing":
      return "分析中";
    case "analyzed":
      return "已分析";
    case "keep":
      return "推荐保留";
    case "improve_then_keep":
      return "优化后保留";
    case "review":
      return "待确认";
    case "reject_suggested":
      return "建议不入册";
    case "approved":
      return "已保留";
    case "excluded":
      return "不入册";
    default:
      return status || "待处理";
  }
}

export function ProjectSetupPanel({
  project,
  themeOptions,
  busy,
  uploadAccept,
  isUploading,
  uploadState,
  uploadHint,
  analysisStatusLabel,
  analysisProgress,
  staleAnalysisCount,
  stats,
  pendingAnalysisCount,
  canAnalyze,
  recentImages,
  allImages,
  isDeletingImage,
  onDrop,
  onUploadInputChange,
  onThemeChange,
  onAnalyze,
  onGoReview,
  onPreviewImage,
  onDeleteImage,
}: ProjectSetupPanelProps) {
  const [showAllImages, setShowAllImages] = useState(false);

  return (
    <div className="setup-view">
      <div className="setup-focus-grid">
        <section className="panel import-panel">
          <div className="panel-header">
            <div>
              <div className="panel-title">导入照片</div>
              <div className="panel-subtitle">
                支持 JPG、PNG、HEIC、HEIF；大文件会显示上传与处理状态。
              </div>
            </div>
            <Upload size={18} />
          </div>

          <div
            className="upload-zone"
            onDragOver={(event) => event.preventDefault()}
            onDrop={onDrop}
          >
            <Upload size={24} />
            <div>
              <strong>选择照片或拖拽到这里</strong>
              <span>HEIC 会在后端转换成可预览和可供大模型识别的 JPEG。</span>
            </div>
            <label className={`btn-primary file-trigger${isUploading ? " is-busy" : ""}`}>
              <Upload size={16} style={{ marginRight: 6 }} />
              导入照片
              <input
                type="file"
                accept={uploadAccept}
                multiple
                onChange={onUploadInputChange}
                disabled={isUploading}
                style={{ display: "none" }}
              />
            </label>
          </div>

          <UploadStatus state={uploadState} hint={uploadHint} />
        </section>

        <aside className="panel prep-panel">
          <div className="panel-header">
            <div>
              <div className="panel-title">准备状态</div>
              <div className="panel-subtitle">{project.currentStep}</div>
            </div>
            <Film size={18} />
          </div>

          <div className="metric-grid">
            <div className="stat">
              <strong>{formatCount(stats.count)}</strong>
              <span>已导入</span>
            </div>
            <div className="stat">
              <strong>{analysisProgress}%</strong>
              <span>{analysisStatusLabel}</span>
            </div>
            <div className="stat">
              <strong>{formatCount(stats.keep + stats.improve)}</strong>
              <span>推荐入册</span>
            </div>
          </div>

          <label className="setup-field">
            <span>默认相册主题</span>
            <select
              value={project.themeId}
              onChange={(event) => onThemeChange(event.target.value)}
              disabled={busy === "theme"}
            >
              {themeOptions.map((theme) => (
                <option key={theme.value} value={theme.value}>
                  {theme.label}
                </option>
              ))}
            </select>
          </label>

          <div className="progress-block">
            <div className="progress-label">
              <span>AI 分析进度</span>
              <strong>{analysisProgress}%</strong>
            </div>
            <div className="progress-track">
              <span style={{ width: `${Math.min(analysisProgress, 100)}%` }} />
            </div>
          </div>

          {staleAnalysisCount > 0 ? (
            <div className="status-note" data-tone="warn">
              AI 配置或提示词已更新，建议重新分析 {formatCount(staleAnalysisCount)} 张旧结果。
            </div>
          ) : null}

          <div className="action-stack">
            <button
              type="button"
              className="btn-primary"
              onClick={onAnalyze}
              disabled={!canAnalyze || busy === "analyze" || isUploading}
            >
              <Sparkles size={16} style={{ marginRight: 6 }} />
              {pendingAnalysisCount > 0
                ? staleAnalysisCount > 0
                  ? `重新分析 ${formatCount(pendingAnalysisCount)} 张照片`
                  : `分析 ${formatCount(pendingAnalysisCount)} 张新增照片`
                : "暂无新增照片可分析"}
            </button>
            <button
              type="button"
              className="btn-secondary"
              onClick={onGoReview}
              disabled={project.images.length === 0 || project.analysisStatus !== "done"}
            >
              <Layers3 size={16} style={{ marginRight: 6 }} />
              去审核
            </button>
          </div>

          {project.lastError ? (
            <div className="status-note" data-tone="bad">
              {project.lastError}
            </div>
          ) : null}
        </aside>
      </div>

      {recentImages.length > 0 ? (
        <section className="panel recent-panel">
          <div className="panel-header">
            <div>
              <div className="panel-title">最近导入</div>
              <div className="panel-subtitle">这里只放少量预览，完整筛选在 AI 审核阶段完成。</div>
            </div>
            {allImages.length > recentImages.length && (
              <button
                type="button"
                className="btn-text"
                onClick={() => setShowAllImages(true)}
              >
                查看全部 {formatCount(allImages.length)} 张
              </button>
            )}
          </div>
          <div className="thumb-strip">
            {recentImages.map((image) => (
              <article className="mini-thumb-card" key={image.id}>
                <button
                  type="button"
                  className="mini-thumb"
                  onClick={() => onPreviewImage(image.id)}
                  aria-label={`放大查看 ${image.fileName}`}
                >
                  <img src={image.thumbnailUrl} alt={image.fileName} loading="lazy" />
                  <span>{imageStatusLabel(image.status)}</span>
                </button>
                <button
                  type="button"
                  className="mini-thumb-delete"
                  onClick={() => onDeleteImage(image.id)}
                  disabled={isDeletingImage || project.status === "analyzing"}
                  aria-label={`删除 ${image.fileName}`}
                >
                  <Trash2 size={14} />
                </button>
              </article>
            ))}
          </div>
        </section>
      ) : null}

      {/* All images dialog */}
      <DialogShell
        open={showAllImages}
        onClose={() => setShowAllImages(false)}
        rootClassName="all-images-dialog"
        backdropClassName="all-images-dialog-backdrop"
        panelClassName="all-images-dialog-panel"
        ariaLabel="全部已导入照片"
        zIndex={70}
      >
        <div className="dialog-header">
          <div>
            <h3>全部已导入照片</h3>
            <p>共 {formatCount(allImages.length)} 张照片</p>
          </div>
          <button
            type="button"
            className="dialog-close"
            onClick={() => setShowAllImages(false)}
            aria-label="关闭"
          >
            <X size={20} />
          </button>
        </div>

        <div className="all-images-grid">
          {allImages.map((image) => (
            <article className="all-images-card" key={image.id}>
              <button
                type="button"
                className="all-images-thumb"
                onClick={() => {
                  setShowAllImages(false);
                  onPreviewImage(image.id);
                }}
                aria-label={`放大查看 ${image.fileName}`}
              >
                <img src={image.thumbnailUrl} alt={image.fileName} loading="lazy" />
              </button>
              <div className="all-images-info">
                <span className="all-images-name">{image.fileName}</span>
                <span className="all-images-status">{imageStatusLabel(image.status)}</span>
              </div>
              <button
                type="button"
                className="all-images-delete"
                onClick={() => onDeleteImage(image.id)}
                disabled={isDeletingImage || project.status === "analyzing"}
                aria-label={`删除 ${image.fileName}`}
              >
                <Trash2 size={14} />
              </button>
            </article>
          ))}
        </div>
      </DialogShell>
    </div>
  );
}
