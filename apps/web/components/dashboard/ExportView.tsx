"use client";

import { BookImage, Check, Clipboard, Images, Share2 } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useMemo, useState } from "react";
import type { Album as AlbumType, AlbumExport, AlbumSocialPost, GitHubPublishProgress, ImageAsset, Project } from "@/lib/types";

export type ExportOption = {
  value: string;
  label: string;
  icon: LucideIcon;
};

type ExportViewProps = {
  selectedProject: Project | null;
  selectedAlbum: AlbumType | null;
  selectedImages: ImageAsset[];
  activeExport: AlbumExport | null;
  recentExports: AlbumExport[];
  busy: string;
  exportOptions: ExportOption[];
  githubPublishProgress: GitHubPublishProgress | null;
  onExportType: (type: string) => void;
  onGoToAlbum: () => void;
};

function exportDescription(value: string) {
  switch (value) {
    case "long_image":
      return "按 HTML 相册的同款视觉生成一张 PNG images，适合保存或转发";
    case "share_link":
      return "生成一个可直接打开的网页链接，适合发给朋友查看";
    case "github_pages":
      return "将相册发布到 GitHub Pages，可在线访问和分享";
    case "html":
      return "可互动网页相册，照片支持点击放大，适合长期保存";
    default:
      return "网页形式，打开即看";
  }
}

function exportHint(value: string) {
  switch (value) {
    case "long_image":
      return "会生成静态 PNG，不包含相册里的点击放大交互。";
    case "share_link":
      return "内容和 HTML 相册一致，方便直接分享 URL。";
    case "github_pages":
      return "需要在设置中配置 GitHub 仓库信息，首次使用需在仓库中手动启用 GitHub Pages。";
    case "html":
    default:
      return "保留完整网页相册体验，是推荐的归档方式。";
  }
}

function socialPlatformLabel(post: AlbumSocialPost, index: number) {
  const platform = post.platform?.toLowerCase();
  if (platform === "xiaohongshu") return "小红书";
  if (platform === "moments" || platform === "wechat" || platform === "wechat_moments") return "朋友圈";
  if (post.title.includes("小红书") || index === 1) return "小红书";
  return "朋友圈";
}

function socialPlatformTone(post: AlbumSocialPost, index: number) {
  return socialPlatformLabel(post, index) === "小红书" ? "accent" : "good";
}

function buildSocialCopy(post: AlbumSocialPost) {
  const parts = [
    post.title.trim(),
    post.hook?.trim(),
    post.body.trim(),
    post.hashtags?.map((tag) => (tag.startsWith("#") ? tag : `#${tag}`)).join(" "),
  ].filter(Boolean);
  return parts.join("\n\n");
}

async function copyToClipboard(text: string) {
  if (navigator.clipboard?.writeText && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // Fall through to the selection-based copy path for embedded browsers.
    }
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "fixed";
  textarea.style.left = "-9999px";
  textarea.style.top = "0";
  document.body.appendChild(textarea);
  textarea.focus();
  textarea.select();
  const copied = document.execCommand("copy");
  document.body.removeChild(textarea);
  if (!copied) {
    throw new Error("copy failed");
  }
}

export function ExportView({
  selectedProject,
  selectedAlbum,
  selectedImages,
  activeExport,
  recentExports,
  busy,
  exportOptions,
  githubPublishProgress,
  onExportType,
  onGoToAlbum,
}: ExportViewProps) {
  const [selectedExportType, setSelectedExportType] = useState(exportOptions[0]?.value ?? "html");
  const selectedOption = useMemo(
    () => exportOptions.find((option) => option.value === selectedExportType) ?? exportOptions[0],
    [exportOptions, selectedExportType],
  );
  const SelectedIcon = selectedOption?.icon ?? Share2;
  const isBusy = busy === `export:${selectedOption?.value}`;
  const [copiedPostKey, setCopiedPostKey] = useState("");
  const [copyFailedPostKey, setCopyFailedPostKey] = useState("");

  // Show the most recent export for the currently selected type
  const displayExport = useMemo(() => {
    // Prefer the active (just-completed) export if it matches the current type
    if (activeExport && activeExport.type === selectedExportType) {
      return activeExport;
    }
    // Otherwise, find the most recent export of this type from the project history
    const matching = recentExports.filter((exp) => exp.type === selectedExportType);
    return matching.length > 0 ? matching[matching.length - 1] : null;
  }, [activeExport, recentExports, selectedExportType]);

  const handleCopySocialPost = async (key: string, text: string) => {
    try {
      await copyToClipboard(text);
      setCopiedPostKey(key);
      setCopyFailedPostKey("");
      window.setTimeout(() => {
        setCopiedPostKey((current) => (current === key ? "" : current));
      }, 1800);
    } catch {
      setCopiedPostKey("");
      setCopyFailedPostKey(key);
      window.setTimeout(() => {
        setCopyFailedPostKey((current) => (current === key ? "" : current));
      }, 2200);
    }
  };

  if (!selectedProject) {
    return <section className="panel empty-state">先创建或选择一个项目。</section>;
  }

  if (!selectedAlbum) {
    return (
      <section className="panel action-state">
        <div>
          <div className="panel-title">还没有可导出的相册</div>
          <div className="panel-subtitle">先生成相册草稿，再导出 HTML、图片或分享链接。</div>
        </div>
        <button type="button" className="btn-primary" onClick={onGoToAlbum}>
          <BookImage size={16} style={{ marginRight: 6 }} />
          去编辑相册
        </button>
      </section>
    );
  }

  return (
    <div className="export-view">
      <section className="panel export-summary-panel">
        <div className="export-copy">
          <span className="pill" data-tone={activeExport ? "good" : "accent"}>
            {activeExport ? "最近已导出" : "等待导出"}
          </span>
          <h2>{selectedAlbum.title}</h2>
          <p>{selectedAlbum.intro}</p>
        </div>
        <div className="export-meta">
          <span className="pill" data-tone="accent">
            {selectedAlbum.pages.length} 页
          </span>
          <span className="pill" data-tone="good">
            版本 {selectedAlbum.version}
          </span>
        </div>
      </section>

      <section className="panel">
        <div className="panel-header">
          <div>
            <div className="panel-title">导出相册</div>
            <div className="panel-subtitle">只选择一种输出方式，然后生成对应结果。</div>
          </div>
          <Share2 size={18} />
        </div>

        <div className="export-mode-tabs" role="tablist" aria-label="导出方式">
          {exportOptions.map((option) => {
            const Icon = option.icon;
            return (
              <button
                type="button"
                className="export-mode-option"
                data-active={selectedOption?.value === option.value}
                key={option.value}
                onClick={() => setSelectedExportType(option.value)}
                role="tab"
                aria-selected={selectedOption?.value === option.value}
              >
                <Icon size={16} />
                <span>{option.label}</span>
              </button>
            );
          })}
        </div>

        {selectedOption ? (
          <section className="export-action-card">
            <div className="export-action-icon">
              <SelectedIcon size={22} />
            </div>
            <div className="export-action-copy">
              <strong>{selectedOption.label}</strong>
              <span>{exportDescription(selectedOption.value)}</span>
              <small>{exportHint(selectedOption.value)}</small>
            </div>
            <button
              type="button"
              className="btn-primary export-primary-action"
              onClick={() => onExportType(selectedOption.value)}
              disabled={isBusy}
            >
              {isBusy
                ? selectedOption.value === "github_pages" ? "发布中..." : "生成中..."
                : selectedOption.value === "github_pages" ? "发布到 GitHub Pages" : `生成${selectedOption.label}`}
            </button>
          </section>
        ) : null}

        {displayExport ? (
          <section className="export-result">
            <div>
              <div className="panel-title">导出结果</div>
              <div className="panel-subtitle">{displayExport.message || "导出完成"}</div>
            </div>
            <a className="btn-secondary" href={displayExport.url} target="_blank" rel="noreferrer">
              打开结果
            </a>
          </section>
        ) : null}

        {githubPublishProgress && (githubPublishProgress.active || githubPublishProgress.phase === "done" || githubPublishProgress.phase === "error") ? (
          <section className="export-progress" data-phase={githubPublishProgress.phase}>
            <div className="export-progress-head">
              <strong>
                {githubPublishProgress.phase === "done"
                  ? "发布完成"
                  : githubPublishProgress.phase === "error"
                  ? "发布失败"
                  : "正在发布到 GitHub Pages"}
              </strong>
              {githubPublishProgress.phase === "uploading_images" && githubPublishProgress.total > 0 ? (
                <span className="export-progress-count">
                  {githubPublishProgress.current} / {githubPublishProgress.total}
                </span>
              ) : null}
            </div>
            {githubPublishProgress.phase === "uploading_images" && githubPublishProgress.total > 0 ? (
              <div className="export-progress-track">
                <div
                  className="export-progress-fill"
                  style={{
                    width: `${Math.min(
                      100,
                      Math.round((githubPublishProgress.current / githubPublishProgress.total) * 100),
                    )}%`,
                  }}
                />
              </div>
            ) : (
              <div className="export-progress-track">
                <div
                  className="export-progress-fill export-progress-fill--indeterminate"
                  style={{ width: githubPublishProgress.phase === "done" ? "100%" : "40%" }}
                />
              </div>
            )}
            <p className="export-progress-message">{githubPublishProgress.message}</p>
            {githubPublishProgress.error ? (
              <p className="export-progress-error">{githubPublishProgress.error}</p>
            ) : null}
          </section>
        ) : null}
      </section>

      {selectedAlbum.socialPosts?.length ? (
        <section className="panel social-post-panel">
          <div className="panel-header">
            <div>
              <div className="panel-title">朋友圈 / 小红书图文</div>
              <div className="panel-subtitle">AI 已拆成不同平台语气，可复制文案，再按推荐配图发布。</div>
            </div>
            <Share2 size={18} />
          </div>
          <div className="social-post-grid">
            {selectedAlbum.socialPosts.map((post, index) => {
              const copyKey = `${post.platform || "post"}-${post.title}-${index}`;
              const copyText = buildSocialCopy(post);
              const platformLabel = socialPlatformLabel(post, index);
              const postImages = post.imageIds
                .map((imageId) => selectedImages.find((image) => image.id === imageId))
                .filter((image): image is ImageAsset => Boolean(image))
                .slice(0, 9);
              return (
                <article className="social-post-card" key={copyKey}>
                  <div className="social-post-head">
                    <div>
                      <span className="pill" data-tone={socialPlatformTone(post, index)}>
                        {platformLabel}
                      </span>
                      <strong>{post.title}</strong>
                    </div>
                    <button
                      type="button"
                      className="btn-secondary social-copy-button"
                      data-state={copiedPostKey === copyKey ? "copied" : copyFailedPostKey === copyKey ? "failed" : "idle"}
                      onClick={() => handleCopySocialPost(copyKey, copyText)}
                    >
                      {copiedPostKey === copyKey ? <Check size={15} /> : <Clipboard size={15} />}
                      {copiedPostKey === copyKey ? "已复制" : copyFailedPostKey === copyKey ? "复制失败" : "复制文案"}
                    </button>
                  </div>

                  <div className="social-post-copy">
                    {post.hook ? <p className="social-post-hook">{post.hook}</p> : null}
                    <p>{post.body}</p>
                  </div>

                  {postImages.length > 0 ? (
                    <div className="social-post-media">
                      <div className="social-post-media-head">
                        <span>
                          <Images size={14} />
                          推荐配图 {postImages.length} 张
                        </span>
                        <small>{platformLabel === "小红书" ? "建议按顺序发布" : "可按心情删减"}</small>
                      </div>
                      <div className="social-post-images">
                        {postImages.map((image, imageIndex) => (
                          <figure key={image.id}>
                            <img src={image.thumbnailUrl} alt={image.fileName} />
                            <figcaption>{imageIndex + 1}</figcaption>
                          </figure>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  {post.hashtags?.length ? (
                    <div className="social-tag-row">
                      {post.hashtags.map((tag, index) => (
                        <span className="social-tag" key={`${tag}-${index}`}>
                          {tag.startsWith("#") ? tag : `#${tag}`}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        </section>
      ) : null}
    </div>
  );
}
