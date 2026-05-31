"use client";

import { type ReactNode, useEffect, useState, useMemo, useCallback } from "react";
import {
  ArrowLeft,
  ArrowRight,
  BookImage,
  Check,
  Info,
  Sparkles,
  Star,
  Wand2,
  Undo2,
  X,
  XCircle,
  Zap,
} from "lucide-react";
import type { ImageAsset, Album as AlbumType, Project } from "@/lib/types";

type ReviewGroup = {
  key: string;
  images: ImageAsset[];
  duplicate: boolean;
  bestImage: ImageAsset;
  label: string;
  summary: string;
};

type ReviewViewProps = {
  selectedProject: Project;
  selectedAlbum: AlbumType | null;
  selectedImages: ImageAsset[];
  busy: string;
  visibleImages: ImageAsset[];
  visibleImageIds: string[];
  reviewGroups: ReviewGroup[];
  activeImage: ImageAsset | null;
  activeImageIndex: number;
  selectedImageIds: string[];
  formatCount: (value: number) => string;
  imageDisplayUrl: (image: ImageAsset) => string;
  onSetActiveImageId: (imageId: string) => void;
  onGoToAdjacentImage: (offset: -1 | 1) => void;
  onDecision: (imageId: string, decision: "keep" | "exclude") => void;
  onGenerate: () => void;
  onAnalyze: () => void;
  onGenerateImageEdit: (imageId: string, prompt: string) => void;
  onUndoImageEdit: (imageId: string) => void;
};

export function ReviewView({
  selectedProject,
  selectedAlbum,
  selectedImages,
  busy,
  visibleImages,
  visibleImageIds,
  reviewGroups,
  activeImage,
  activeImageIndex,
  selectedImageIds,
  formatCount,
  imageDisplayUrl,
  onSetActiveImageId,
  onGoToAdjacentImage,
  onDecision,
  onGenerate,
  onAnalyze,
  onGenerateImageEdit,
  onUndoImageEdit,
}: ReviewViewProps) {
  const [showInsights, setShowInsights] = useState(false);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [showEditPrompt, setShowEditPrompt] = useState(false);
  const [editPrompt, setEditPrompt] = useState("");
  const [selectedSuggestions, setSelectedSuggestions] = useState<Set<number>>(new Set());
  const [isLightbox, setIsLightbox] = useState(false);

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!activeImage) return;

      // Escape closes lightbox first, then panels
      if (e.key === "Escape") {
        e.preventDefault();
        if (isLightbox) {
          setIsLightbox(false);
        } else if (showInsights) {
          setShowInsights(false);
        } else if (showEditPrompt) {
          setShowEditPrompt(false);
          setEditPrompt("");
        }
        return;
      }

      switch (e.key) {
        case "ArrowLeft":
          if (!isLightbox) {
            e.preventDefault();
            handlePrev();
          }
          break;
        case "ArrowRight":
          if (!isLightbox) {
            e.preventDefault();
            handleNext();
          }
          break;
        case "k":
        case "K":
          if (!isLightbox) {
            e.preventDefault();
            handleKeep();
          }
          break;
        case "x":
        case "X":
          if (!isLightbox) {
            e.preventDefault();
            handleExclude();
          }
          break;
        case "i":
        case "I":
          if (!isLightbox) {
            e.preventDefault();
            setShowInsights(!showInsights);
          }
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activeImage, showInsights, showEditPrompt, isLightbox]);

  const handlePrev = useCallback(() => {
    if (activeImageIndex > 0) {
      setIsTransitioning(true);
      onGoToAdjacentImage(-1);
      setTimeout(() => setIsTransitioning(false), 300);
    }
  }, [activeImageIndex, onGoToAdjacentImage]);

  const handleNext = useCallback(() => {
    if (activeImageIndex < visibleImages.length - 1) {
      setIsTransitioning(true);
      onGoToAdjacentImage(1);
      setTimeout(() => setIsTransitioning(false), 300);
    }
  }, [activeImageIndex, visibleImages.length, onGoToAdjacentImage]);

  const handleKeep = useCallback(() => {
    if (activeImage) {
      onDecision(activeImage.id, "keep");
      setTimeout(() => handleNext(), 400);
    }
  }, [activeImage, onDecision, handleNext]);

  const handleExclude = useCallback(() => {
    if (activeImage) {
      onDecision(activeImage.id, "exclude");
      setTimeout(() => handleNext(), 400);
    }
  }, [activeImage, onDecision, handleNext]);

  const toggleSuggestion = useCallback((index: number) => {
    setSelectedSuggestions((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  }, []);

  const handleBatchOptimize = useCallback(() => {
    if (!activeImage || selectedSuggestions.size === 0) return;
    const suggestions = activeImage.analysis?.editSuggestions || [];
    const prompts = Array.from(selectedSuggestions).map(
      (i) => `${suggestions[i]?.actionLabel || suggestions[i]?.type}: ${suggestions[i]?.reason}`
    );
    onGenerateImageEdit(activeImage.id, prompts.join("；"));
    setSelectedSuggestions(new Set());
    setShowEditPrompt(false);
  }, [activeImage, selectedSuggestions, onGenerateImageEdit]);

  // Empty state
  if (selectedImages.length === 0) {
    return (
      <div className="review-empty">
        <div className="empty-content">
          <h2>还没有照片</h2>
          <p>导入照片后，AI 会帮你分析并推荐入册候选</p>
          <button className="btn-primary" onClick={onAnalyze}>
            开始分析
          </button>
        </div>
      </div>
    );
  }

  // Analyzing state
  if (selectedProject.status === "analyzing") {
    return (
      <div className="review-analyzing">
        <div className="analyzing-content">
          <div className="analyzing-spinner" />
          <h2>AI 正在分析照片</h2>
          <p>进度 {selectedProject.analysisProgress}%</p>
        </div>
      </div>
    );
  }

  return (
    <div className="review-cinema">
      {/* Main image stage */}
      <div className="review-stage">
        {activeImage && (
          <div
            className={`review-image-wrap ${isTransitioning ? 'transitioning' : ''}`}
            onClick={() => setIsLightbox(true)}
            role="button"
            tabIndex={0}
            title="点击放大查看"
          >
            <img
              src={imageDisplayUrl(activeImage)}
              alt={activeImage.fileName}
              className="review-image-main"
            />

            {/* Status badge */}
            {activeImage.status && (
              <div className={`review-status-badge status-${activeImage.status}`}>
                {activeImage.status === "approved" && <Check size={16} />}
                {activeImage.status === "excluded" && <X size={16} />}
                {activeImage.status === "keep" && <Star size={16} />}
                <span>
                  {activeImage.status === "approved" ? "已采纳" :
                   activeImage.status === "excluded" ? "已排除" :
                   activeImage.status === "keep" ? "保留" :
                   activeImage.status === "improve_then_keep" ? "优化后保留" :
                   activeImage.status === "review" ? "待审核" :
                   activeImage.status === "reject_suggested" ? "建议排除" :
                   activeImage.status === "analyzed" ? "已分析" :
                   activeImage.status === "analyzing" ? "分析中" :
                   activeImage.status === "uploaded" ? "已上传" :
                   activeImage.status}
                </span>
              </div>
            )}

            {/* Best in group badge */}
            {activeImage.analysis?.similarGroupBest && (
              <div className="review-best-badge">
                <Star size={16} fill="currentColor" />
                <span>本组最佳</span>
              </div>
            )}

            {/* Zoom hint */}
            <div className="review-zoom-hint">
              点击放大
            </div>
          </div>
        )}

        {/* Navigation arrows */}
        <button
          className="review-nav-arrow review-nav-prev"
          onClick={handlePrev}
          disabled={activeImageIndex === 0}
          aria-label="上一张"
        >
          <ArrowLeft size={32} />
        </button>
        <button
          className="review-nav-arrow review-nav-next"
          onClick={handleNext}
          disabled={activeImageIndex === visibleImages.length - 1}
          aria-label="下一张"
        >
          <ArrowRight size={32} />
        </button>

        {/* Counter - top right */}
        <div className="review-counter-float">
          {activeImageIndex + 1} / {visibleImages.length}
        </div>

        {/* Action Bar: floating capsule at bottom of stage */}
        <div className="review-action-bar">
          <div className="action-group">
            <button
              className={`action-btn action-btn-info ${showInsights ? 'active' : ''}`}
              onClick={() => setShowInsights(!showInsights)}
              aria-label="AI 洞察"
              title="AI 洞察 (I)"
            >
              <Info size={18} />
              <span>洞察</span>
              <kbd>I</kbd>
            </button>
            <button
              className={`action-btn action-btn-magic ${showEditPrompt ? 'active' : ''}`}
              onClick={() => {
                setShowEditPrompt(!showEditPrompt);
                setSelectedSuggestions(new Set());
              }}
              aria-label="AI 优化"
              title="AI 图像优化"
            >
              <Wand2 size={18} />
              <span>优化</span>
            </button>
          </div>

          <div className="action-divider" />

          <div className="action-group">
            <button
              className="action-btn action-btn-keep"
              onClick={handleKeep}
              aria-label="保留 (K)"
              title="保留 (K)"
            >
              <Check size={18} />
              <span>保留</span>
              <kbd>K</kbd>
            </button>
            <button
              className="action-btn action-btn-exclude"
              onClick={handleExclude}
              aria-label="排除 (X)"
              title="排除 (X)"
            >
              <X size={18} />
              <span>排除</span>
              <kbd>X</kbd>
            </button>
          </div>
        </div>
      </div>

      {/* Filmstrip: thumbnails only */}
      <div className="review-filmstrip">
        <div className="filmstrip-track">
          {visibleImages.map((image, index) => {
            const prevImage = index > 0 ? visibleImages[index - 1] : null;
            const isNewGroup = image.analysis?.similarGroupId &&
              (!prevImage || prevImage.analysis?.similarGroupId !== image.analysis.similarGroupId);
            const isGroupStart = isNewGroup;
            const isGroupEnd = !image.analysis?.similarGroupId ||
              index === visibleImages.length - 1 ||
              visibleImages[index + 1]?.analysis?.similarGroupId !== image.analysis.similarGroupId;

            return (
              <button
                key={image.id}
                className={`filmstrip-thumb ${index === activeImageIndex ? 'active' : ''} ${
                  image.status === 'approved' ? 'status-keep' :
                  image.status === 'excluded' ? 'status-exclude' : ''
                } ${image.analysis?.similarGroupId ? 'in-group' : ''} ${isGroupStart ? 'group-start' : ''} ${isGroupEnd ? 'group-end' : ''}`}
                onClick={() => onSetActiveImageId(image.id)}
                title={image.analysis?.similarGroupLabel || image.fileName}
              >
                <img src={image.thumbnailUrl} alt={image.fileName} />
                {image.analysis?.similarGroupBest && (
                  <div className="filmstrip-best">
                    <Star size={10} fill="currentColor" />
                  </div>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* AI Insights Panel */}
      {showInsights && activeImage?.analysis && (
        <div className="review-insights-panel">
          <div className="insights-header">
            <h3>AI 洞察</h3>
            <button
              className="insights-close"
              onClick={() => setShowInsights(false)}
              aria-label="关闭"
            >
              <X size={20} />
            </button>
          </div>

          <div className="insights-content">
            {/* Scores */}
            <div className="insights-scores">
              <div className="score-item">
                <div className="score-circle score-quality">
                  <span>{activeImage.analysis.qualityScore}</span>
                </div>
                <label>质量</label>
              </div>
              <div className="score-item">
                <div className="score-circle score-story">
                  <span>{activeImage.analysis.storyScore}</span>
                </div>
                <label>故事性</label>
              </div>
              <div className="score-item">
                <div className="score-circle score-preservation">
                  <span>{activeImage.analysis.preservationScore}</span>
                </div>
                <label>保存价值</label>
              </div>
            </div>

            {/* Reasons */}
            {activeImage.analysis.reasons && activeImage.analysis.reasons.length > 0 && (
              <div className="insights-section">
                <h4>推荐理由</h4>
                <ul className="insights-list">
                  {activeImage.analysis.reasons.map((reason, i) => (
                    <li key={i}>{reason}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* Tags */}
            {activeImage.analysis.detectedContent && (
              <div className="insights-section">
                <h4>识别内容</h4>
                <div className="insights-tags">
                  {activeImage.analysis.detectedContent.scenes?.map((scene, i) => (
                    <span key={i} className="insight-tag">{scene}</span>
                  ))}
                  {activeImage.analysis.detectedContent.mood?.map((mood, i) => (
                    <span key={i} className="insight-tag tag-mood">{mood}</span>
                  ))}
                </div>
              </div>
            )}

            {/* Similar group info */}
            {activeImage.analysis.similarGroupId && (
              <div className="insights-section">
                <h4>相似照片组</h4>
                <p className="insights-text">
                  {activeImage.analysis.similarGroupReason || "这张照片与其他照片相似，AI 已标记最佳版本"}
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {/* AI Edit Prompt Panel - centered modal with backdrop */}
      {showEditPrompt && activeImage && (
        <>
          <div className="edit-panel-backdrop" onClick={() => {
            setShowEditPrompt(false);
            setEditPrompt("");
            setSelectedSuggestions(new Set());
          }} />
          <div className="review-edit-panel">
          <div className="edit-panel-header">
            <Wand2 size={18} />
            <h4>AI 图像优化</h4>
            <button
              className="edit-panel-close"
              onClick={() => {
                setShowEditPrompt(false);
                setEditPrompt("");
                setSelectedSuggestions(new Set());
              }}
            >
              <X size={16} />
            </button>
          </div>

          {/* AI Edit Suggestions - multi-select */}
          {activeImage.analysis?.editSuggestions && activeImage.analysis.editSuggestions.length > 0 && (
            <div className="edit-suggestions">
              <div className="edit-suggestions-head">
                <h5>AI 建议优化</h5>
                <span className="suggestion-count">
                  {selectedSuggestions.size > 0
                    ? `已选 ${selectedSuggestions.size} 项`
                    : "点击选择"}
                </span>
              </div>
              <div className="suggestion-list">
                {activeImage.analysis.editSuggestions.map((suggestion, i) => (
                  <button
                    key={i}
                    className={`suggestion-btn ${selectedSuggestions.has(i) ? 'selected' : ''}`}
                    onClick={() => toggleSuggestion(i)}
                    disabled={busy.startsWith('generate_edit:')}
                  >
                    <span className="suggestion-check">
                      {selectedSuggestions.has(i) ? <Check size={14} /> : null}
                    </span>
                    <div className="suggestion-content">
                      <strong>{suggestion.actionLabel || suggestion.type}</strong>
                      <span>{suggestion.reason}</span>
                    </div>
                  </button>
                ))}
              </div>
              {selectedSuggestions.size > 0 && (
                <button
                  className="batch-optimize-btn"
                  onClick={handleBatchOptimize}
                  disabled={busy.startsWith('generate_edit:')}
                >
                  <Sparkles size={14} />
                  {busy.startsWith('generate_edit:')
                    ? '优化中...'
                    : `一键优化 (${selectedSuggestions.size})`}
                </button>
              )}
            </div>
          )}

          {/* Custom Edit */}
          <div className="edit-panel-body">
            <h5>自定义优化</h5>
            <textarea
              value={editPrompt}
              onChange={(e) => setEditPrompt(e.target.value)}
              placeholder="描述你想要的优化效果，例如：提升亮度、去除模糊、调整色彩..."
              rows={2}
            />
            <div className="edit-panel-actions">
              {activeImage.editHistory && activeImage.editHistory.length > 0 && (
                <button
                  className="btn-secondary"
                  onClick={() => {
                    onUndoImageEdit(activeImage.id);
                    setShowEditPrompt(false);
                  }}
                  disabled={busy.startsWith('undo_edit:')}
                >
                  <Undo2 size={14} />
                  撤销
                </button>
              )}
              <button
                className="btn-primary"
                onClick={() => {
                  if (editPrompt.trim()) {
                    onGenerateImageEdit(activeImage.id, editPrompt);
                    setShowEditPrompt(false);
                    setEditPrompt("");
                  }
                }}
                disabled={!editPrompt.trim() || busy.startsWith('generate_edit:')}
              >
                <Sparkles size={14} />
                {busy.startsWith('generate_edit:') ? '优化中...' : '开始优化'}
              </button>
            </div>
          </div>
        </div>
        </>
      )}

      {/* Lightbox overlay */}
      {isLightbox && activeImage && (
        <div className="review-lightbox" onClick={() => setIsLightbox(false)}>
          <button
            className="lightbox-close"
            onClick={() => setIsLightbox(false)}
            aria-label="关闭"
          >
            <XCircle size={28} />
          </button>
          <img
            src={imageDisplayUrl(activeImage)}
            alt={activeImage.fileName}
            className="lightbox-image"
            onClick={(e) => e.stopPropagation()}
          />
          <div className="lightbox-info">
            <span>{activeImage.fileName}</span>
            <span>ESC 关闭</span>
          </div>
        </div>
      )}

      {/* Top action bar */}
      <div className="review-topbar">
        <div className="topbar-stats">
          <span className="stat-chip">
            <Star size={14} />
            {formatCount(selectedImageIds.length)} 已选
          </span>
          <span className="stat-chip">
            {formatCount(visibleImages.length)} 总计
          </span>
        </div>
        <button
          className="btn-primary"
          onClick={onGenerate}
          disabled={busy === "generate"}
        >
          <BookImage size={18} />
          生成相册
        </button>
      </div>
    </div>
  );
}
