"use client";

import { useMemo, type CSSProperties } from "react";
import {
  BookImage,
  ChevronLeft,
  ChevronRight,
  ImagePlus,
  Layers3,
  Link2,
  Redo2,
  Save,
  Trash2,
  Undo2,
  X,
} from "lucide-react";
import type { Album as AlbumType, AlbumPage, ImageAsset, Project } from "@/lib/types";

type ThemeOption = {
  value: string;
  label: string;
  description?: string;
  accent?: string;
  surface?: string;
  text?: string;
};

type LayoutOption = ThemeOption;

type AlbumEditorViewProps = {
  selectedProject: Project | null;
  selectedAlbum: AlbumType | null;
  albumDraft: AlbumType | null;
  selectedImages: ImageAsset[];
  busy: string;
  albumDirty: boolean;
  albumPageId: string;
  editorImageId: string;
  dragPageId: string;
  dragImageId: string;
  themeOptions: ThemeOption[];
  layoutOptions: LayoutOption[];
  onSetAlbumPageId: (value: string) => void;
  onSetEditorImageId: (value: string) => void;
  onSetDragPageId: (value: string) => void;
  onSetDragImageId: (value: string) => void;
  onAlbumTitleChange: (value: string) => void;
  onAlbumIntroChange: (value: string) => void;
  onAlbumThemeChange: (value: string) => void;
  onMoveAlbumPage: (sourcePageId: string, targetPageId: string) => void;
  onMoveImageWithinPage: (pageId: string, sourceImageId: string, targetImageId: string) => void;
  onPatchAlbumPage: (pageId: string, patch: Partial<AlbumPage>) => void;
  onDeleteAlbumPage: (pageId: string) => void;
  onAddImageToPage: (pageId: string, imageId: string) => void;
  onRemoveImageFromPage: (pageId: string, imageId: string) => void;
  onSetCoverImage: (imageId: string) => void;
  onSaveAlbumDraft: () => void;
  onUndoAlbum: () => void;
  onRedoAlbum: () => void;
  onGenerate: () => void;
  onGoToReview: () => void;
  onGoToExport: () => void;
};

function pageTypeForLayout(layoutId: string) {
  switch (layoutId) {
    case "cover_full_bleed":
      return "cover";
    case "hero_story":
    case "full_bleed_quote":
      return "hero";
    case "single_photo_caption":
    case "diptych":
    case "triptych":
      return "detail";
    case "timeline_ribbon":
      return "sequence";
    case "mosaic":
    case "scrapbook":
      return "collage";
    case "gallery_wall":
      return "gallery";
    case "quote_break":
      return "pause";
    case "contact_sheet":
      return "index";
    case "ending_text":
      return "ending";
    default:
      return "detail";
  }
}

function normalizePageType(pageType: string, layoutId = "") {
  switch (pageType) {
    case "chapter":
    case "transition":
      return pageTypeForLayout(layoutId);
    case "cover":
    case "opener":
    case "hero":
    case "detail":
    case "sequence":
    case "collage":
    case "gallery":
    case "pause":
    case "index":
    case "ending":
      return pageType;
    default:
      return "detail";
  }
}

function pageTypeLabel(pageType: string, layoutId = "") {
  switch (normalizePageType(pageType, layoutId)) {
    case "cover":
      return "封面";
    case "opener":
      return "开场";
    case "hero":
      return "主视觉";
    case "detail":
      return "细节";
    case "sequence":
      return "时间带";
    case "collage":
      return "拼贴";
    case "gallery":
      return "画廊";
    case "pause":
      return "留白";
    case "index":
      return "索引";
    case "ending":
      return "结尾";
    default:
      return pageType || "页面";
  }
}

function imageDisplayUrl(image: ImageAsset) {
  return image.derivedUrl || image.originalUrl || image.thumbnailUrl;
}

function layoutDescription(layoutId: string) {
  switch (layoutId) {
    case "cover_full_bleed":
      return "适合相册第一眼，用一张强记忆照片做满版封面。";
    case "hero_story":
      return "一张主图配一段文字，适合开场主视觉或关键回忆。";
    case "single_photo_caption":
      return "单张照片和短配文，适合需要被安静看完的画面。";
    case "diptych":
      return "两张照片并置，适合对照场景、人物或情绪变化。";
    case "triptych":
      return "三张照片形成节奏，适合动作、细节或时间推进。";
    case "mosaic":
      return "多张照片拼贴，适合把零散片段整理成一页故事。";
    case "gallery_wall":
      return "画廊墙式排布，适合把多张照片做成一页有节奏的展览。";
    case "timeline_ribbon":
      return "横向时间带，适合表达一天、一段路或一组动作的推进。";
    case "full_bleed_quote":
      return "满版照片叠加短句，适合相册里的情绪高潮。";
    case "scrapbook":
      return "手账拼贴，适合票据、细节、人物和场景混合成一页记忆板。";
    case "contact_sheet":
      return "胶片索引感，适合收纳更多照片但保持秩序。";
    case "quote_break":
      return "文字留白页，用来给相册呼吸和转场。";
    case "ending_text":
      return "收束页，适合一句结尾和最后的回望。";
    default:
      return "根据当前照片和文字生成页面版式。";
  }
}

function pageImageLimitHint(layoutId: string) {
  switch (layoutId) {
    case "cover_full_bleed":
    case "hero_story":
    case "single_photo_caption":
      return "建议 1 张";
    case "diptych":
      return "建议 2 张";
    case "triptych":
      return "建议 3 张";
    case "mosaic":
      return "建议 4-6 张";
    case "gallery_wall":
      return "建议 5-9 张";
    case "timeline_ribbon":
      return "建议 3-7 张";
    case "full_bleed_quote":
      return "建议 1 张";
    case "scrapbook":
      return "建议 4-8 张";
    case "contact_sheet":
      return "适合 6 张以上";
    default:
      return "可不放照片";
  }
}

function AlbumPagePreview({
  page,
  images,
  themeId,
  onSelectImage,
}: {
  page: AlbumPage;
  images: ImageAsset[];
  themeId: string;
  onSelectImage: (imageId: string) => void;
}) {
  return (
    <article
      className="album-spread-preview"
      data-page-type={normalizePageType(page.pageType, page.layoutId)}
      data-layout={page.layoutId}
      data-theme={themeId || "film_travel"}
    >
      {normalizePageType(page.pageType, page.layoutId) === "cover" ? (
        <>
          <div className="spread-cover-media">
            {images[0] ? <img src={imageDisplayUrl(images[0])} alt={images[0].fileName} /> : null}
          </div>
          <div className="spread-cover-copy">
            <span>Memoir / 集忆</span>
            <h2>{page.title || "未命名相册"}</h2>
            {page.body ? <p>{page.body}</p> : null}
          </div>
        </>
      ) : (
        <>
          <div className="spread-copy">
            <span>{pageTypeLabel(page.pageType, page.layoutId)}</span>
            <h2>{page.title || pageTypeLabel(page.pageType, page.layoutId)}</h2>
            {page.body ? <p>{page.body}</p> : null}
            {page.caption ? <small>{page.caption}</small> : null}
          </div>
          {images.length > 0 ? (
            <div className="spread-photo-grid">
              {images.map((image, index) => (
                <button
                  type="button"
                  key={`${image.id}-${index}`}
                  className="spread-photo"
                  onClick={() => onSelectImage(image.id)}
                  aria-label={`选择 ${image.fileName}`}
                >
                  <img src={imageDisplayUrl(image)} alt={image.fileName} />
                </button>
              ))}
            </div>
          ) : (
            <div className="spread-empty-copy">
              <strong>{page.caption || page.body || "留白"}</strong>
            </div>
          )}
        </>
      )}
    </article>
  );
}

export function AlbumEditorView({
  selectedProject,
  selectedAlbum,
  albumDraft,
  selectedImages,
  busy,
  albumDirty,
  albumPageId,
  editorImageId,
  dragPageId,
  dragImageId,
  themeOptions,
  layoutOptions,
  onSetAlbumPageId,
  onSetEditorImageId,
  onSetDragPageId,
  onSetDragImageId,
  onAlbumTitleChange,
  onAlbumIntroChange,
  onAlbumThemeChange,
  onMoveAlbumPage,
  onMoveImageWithinPage,
  onPatchAlbumPage,
  onDeleteAlbumPage,
  onAddImageToPage,
  onRemoveImageFromPage,
  onSetCoverImage,
  onSaveAlbumDraft,
  onUndoAlbum,
  onRedoAlbum,
  onGenerate,
  onGoToReview,
  onGoToExport,
}: AlbumEditorViewProps) {
  const activeAlbumPage = useMemo(
    () => albumDraft?.pages.find((page) => page.id === albumPageId) ?? albumDraft?.pages[0] ?? null,
    [albumDraft, albumPageId],
  );
  const currentPageIndex = useMemo(() => {
    if (!albumDraft || !activeAlbumPage) return 0;
    const index = albumDraft.pages.findIndex((page) => page.id === activeAlbumPage.id);
    return Math.max(0, index);
  }, [activeAlbumPage, albumDraft]);
  const pageImages = useMemo(
    () =>
      activeAlbumPage
        ? activeAlbumPage.imageIds
            .map((imageId) => selectedImages.find((image) => image.id === imageId))
            .filter((image): image is ImageAsset => Boolean(image))
        : [],
    [activeAlbumPage, selectedImages],
  );
  const availableImages = useMemo(
    () =>
      selectedImages.filter((image) => !activeAlbumPage?.imageIds.includes(image.id)),
    [activeAlbumPage, selectedImages],
  );
  const coverPage = useMemo(
    () => albumDraft?.pages.find((page) => page.pageType === "cover") ?? albumDraft?.pages[0] ?? null,
    [albumDraft],
  );
  const selectedTheme = useMemo(
    () => themeOptions.find((theme) => theme.value === albumDraft?.themeId) ?? themeOptions[0] ?? null,
    [albumDraft?.themeId, themeOptions],
  );
  const pageLayoutDescription = activeAlbumPage ? layoutDescription(activeAlbumPage.layoutId) : "";
  const selectedEditorImage = useMemo(
    () => selectedImages.find((image) => image.id === editorImageId) ?? pageImages[0] ?? null,
    [editorImageId, pageImages, selectedImages],
  );

  if (!selectedProject) {
    return <section className="panel empty-state">先创建或选择一个项目。</section>;
  }

  if (selectedProject.status === "generating_album") {
    return (
      <div className="album-editor">
        <section className="panel action-state">
          <div>
            <div className="panel-title">正在生成相册</div>
            <div className="panel-subtitle">AI 正在为你的照片设计相册结构、版式和配文，请稍候…</div>
          </div>
          <div className="progress-bar">
            <div className="progress-bar-track">
              <div className="progress-bar-fill progress-bar-indeterminate" />
            </div>
          </div>
          <div className="btn-row">
            <button
              type="button"
              className="btn-secondary"
              onClick={onGoToReview}
            >
              <Layers3 size={16} style={{ marginRight: 6 }} />
              回到审核
            </button>
          </div>
        </section>
      </div>
    );
  }

  if (!selectedAlbum || !albumDraft) {
    return (
      <div className="album-editor">
        <section className="panel action-state">
          <div>
            <div className="panel-title">还没有可编辑的相册</div>
            <div className="panel-subtitle">先在审核页确认照片，再生成相册草稿。</div>
          </div>
          <div className="btn-row">
            <button type="button" className="btn-primary" onClick={onGenerate}>
              <BookImage size={16} style={{ marginRight: 6 }} />
              生成相册
            </button>
            <button
              type="button"
              className="btn-secondary"
              onClick={onGoToReview}
              disabled={selectedImages.length === 0}
            >
              <Layers3 size={16} style={{ marginRight: 6 }} />
              回到审核
            </button>
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="album-editor">
      <section className="panel album-editor-toolbar">
        <div className="album-toolbar-copy">
          <div className="panel-title">设计相册</div>
          <div className="panel-subtitle">
            AI 已生成相册结构、版式和配文。这里主要做少量确认和微调。
          </div>
        </div>
        <div className="album-toolbar-actions">
          <span className="pill" data-tone={albumDirty ? "warn" : "good"}>
            {albumDirty ? "未保存" : "已保存"}
          </span>
          <button
            type="button"
            className="btn-secondary"
            onClick={onUndoAlbum}
            disabled={busy === "album_undo" || !selectedAlbum?.editHistory?.length}
          >
            <Undo2 size={16} style={{ marginRight: 6 }} />
            撤销
          </button>
          <button
            type="button"
            className="btn-secondary"
            onClick={onRedoAlbum}
            disabled={busy === "album_redo" || !selectedAlbum?.redoStack?.length}
          >
            <Redo2 size={16} style={{ marginRight: 6 }} />
            重做
          </button>
          <button
            type="button"
            className="btn-primary"
            onClick={onSaveAlbumDraft}
            disabled={!albumDirty || busy === "album_save"}
          >
            <Save size={16} style={{ marginRight: 6 }} />
            保存编辑
          </button>
          <button type="button" className="btn-secondary" onClick={onGenerate} disabled={busy === "generate"}>
            <BookImage size={16} style={{ marginRight: 6 }} />
            重新生成
          </button>
          <button type="button" className="btn-secondary" onClick={onGoToExport}>
            <Link2 size={16} style={{ marginRight: 6 }} />
            导出
          </button>
        </div>
      </section>

      <div className="album-editor-grid">
        <aside className="panel album-outline">
          <div className="panel-header">
            <div>
              <div className="panel-title">相册结构</div>
              <div className="panel-subtitle">{albumDraft.title} · {albumDraft.pages.length} 页 · 版本 {albumDraft.version}</div>
            </div>
            <Layers3 size={18} />
          </div>

          <div className="album-meta-form">
            <label className="setup-field">
              <span>相册标题</span>
              <input
                value={albumDraft.title}
                onChange={(event) => onAlbumTitleChange(event.target.value)}
                placeholder="相册标题"
              />
            </label>
          </div>

          <section className="theme-picker" aria-label="相册主题">
            <div className="section-title-row">
              <div>
                <div className="panel-title">主题风格</div>
                <div className="panel-subtitle">
                  {selectedTheme?.description || "切换后会立即改变预览和导出相册的视觉风格。"}
                </div>
              </div>
              {selectedTheme ? (
                <span className="pill" data-tone="accent">
                  {selectedTheme.label}
                </span>
              ) : null}
            </div>
            <div className="theme-card-grid">
              {themeOptions.map((theme) => (
                <button
                  type="button"
                  key={theme.value}
                  className="theme-choice"
                  data-active={theme.value === albumDraft.themeId}
                  data-theme={theme.value}
                  style={
                    {
                      "--theme-accent": theme.accent,
                      "--theme-surface": theme.surface,
                      "--theme-text": theme.text,
                    } as CSSProperties
                  }
                  onClick={() => onAlbumThemeChange(theme.value)}
                >
                  <span className="theme-swatch" aria-hidden="true">
                    <span />
                    <span />
                    <span />
                  </span>
                  <strong>{theme.label}</strong>
                  <small>{theme.description}</small>
                </button>
              ))}
            </div>
          </section>

          <label className="setup-field">
            <span>相册前言</span>
            <textarea
              rows={5}
              value={albumDraft.intro}
              onChange={(event) => onAlbumIntroChange(event.target.value)}
              placeholder="为这本相册写一段开场白"
            />
          </label>

          {albumDraft.designNotes ? (
            <div className="design-note">
              <strong>AI 设计意图</strong>
              <span>{albumDraft.designNotes}</span>
            </div>
          ) : null}

          <div className="outline-pages">
            {albumDraft.pages.map((page) => (
              <article
                key={page.id}
                className="outline-page"
                data-active={page.id === albumPageId}
                draggable
                onDragStart={() => onSetDragPageId(page.id)}
                onDragOver={(event) => event.preventDefault()}
                onDrop={() => onMoveAlbumPage(dragPageId, page.id)}
                onClick={() => onSetAlbumPageId(page.id)}
              >
                <button
                  type="button"
                  className="outline-page-main"
                  onClick={(event) => {
                    event.stopPropagation();
                    onSetAlbumPageId(page.id);
                  }}
                >
                  <strong>
                    {page.order}. {page.title || pageTypeLabel(page.pageType, page.layoutId)}
                  </strong>
                  <span>
                    {pageTypeLabel(page.pageType, page.layoutId)} · {page.layoutId}
                  </span>
                </button>
                <button
                  type="button"
                  className="outline-page-delete"
                  onClick={(event) => {
                    event.stopPropagation();
                    onDeleteAlbumPage(page.id);
                  }}
                  disabled={albumDraft.pages.length <= 1}
                  aria-label={`删除页面 ${page.title || pageTypeLabel(page.pageType, page.layoutId)}`}
                >
                  <Trash2 size={15} />
                </button>
              </article>
            ))}
          </div>
        </aside>

        <section className="panel album-canvas">
          <div className="panel-header">
            <div>
              <div className="panel-title">页面预览</div>
              <div className="panel-subtitle">
                {activeAlbumPage
                  ? `${pageTypeLabel(activeAlbumPage.pageType, activeAlbumPage.layoutId)} · ${activeAlbumPage.layoutId} · 第 ${currentPageIndex + 1} 页`
                  : "选择一页开始编辑"}
              </div>
            </div>
            <div className="btn-row">
              <button
                type="button"
                className="icon-button"
                onClick={() =>
                  onSetAlbumPageId(albumDraft.pages[Math.max(currentPageIndex - 1, 0)]?.id ?? albumPageId)
                }
                disabled={currentPageIndex <= 0}
                aria-label="上一页"
              >
                <ChevronLeft size={16} />
              </button>
              <button
                type="button"
                className="icon-button"
                onClick={() =>
                  onSetAlbumPageId(
                    albumDraft.pages[Math.min(currentPageIndex + 1, albumDraft.pages.length - 1)]
                      ?.id ?? albumPageId,
                  )
                }
                disabled={currentPageIndex >= albumDraft.pages.length - 1}
                aria-label="下一页"
              >
                <ChevronRight size={16} />
              </button>
            </div>
          </div>

          {activeAlbumPage ? (
            <AlbumPagePreview
              page={activeAlbumPage}
              images={pageImages}
              themeId={albumDraft.themeId}
              onSelectImage={onSetEditorImageId}
            />
          ) : (
            <div className="empty-state">选择一页开始编辑。</div>
          )}
        </section>

        <aside className="panel album-page-sidebar">
          {activeAlbumPage ? (
            <div className="album-page-editor">
              <section className="album-editor-section">
                <div className="section-title-row">
                  <div>
                    <div className="panel-title">页面内容</div>
                    <div className="panel-subtitle">{pageLayoutDescription}</div>
                  </div>
                  <button
                    type="button"
                    className="icon-button"
                    onClick={() => onDeleteAlbumPage(activeAlbumPage.id)}
                    disabled={albumDraft.pages.length <= 1}
                    aria-label="删除当前页面"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
                <label className="setup-field">
                  <span>页面标题</span>
                  <input
                    value={activeAlbumPage.title}
                    onChange={(event) => onPatchAlbumPage(activeAlbumPage.id, { title: event.target.value })}
                  />
                </label>
                <label className="setup-field">
                  <span>页面正文</span>
                  <textarea
                    rows={5}
                    value={activeAlbumPage.body}
                    onChange={(event) => onPatchAlbumPage(activeAlbumPage.id, { body: event.target.value })}
                    placeholder="为这一页写一段说明"
                  />
                </label>
                <label className="setup-field">
                  <span>配文</span>
                  <textarea
                    rows={3}
                    value={activeAlbumPage.caption}
                    onChange={(event) => onPatchAlbumPage(activeAlbumPage.id, { caption: event.target.value })}
                    placeholder="直接编辑图片配文"
                  />
                </label>
              </section>

              <section className="album-editor-section">
                <div className="section-title-row">
                  <div>
                    <div className="panel-title">版式</div>
                    <div className="panel-subtitle">{pageImageLimitHint(activeAlbumPage.layoutId)}</div>
                  </div>
                  <span className="pill" data-tone="accent">
                    {pageImages.length} 张
                  </span>
                </div>
                <div className="layout-picker">
                  {layoutOptions.map((layout) => (
                    <button
                      type="button"
                      key={layout.value}
                      className="layout-choice"
                      data-active={layout.value === activeAlbumPage.layoutId}
                      onClick={() => onPatchAlbumPage(activeAlbumPage.id, { layoutId: layout.value })}
                    >
                      <span>{layout.label}</span>
                    </button>
                  ))}
                </div>
              </section>

              <section className="album-editor-section">
                <div className="section-title-row">
                  <div>
                    <div className="panel-title">页面照片</div>
                    <div className="panel-subtitle">拖动照片调整顺序，预览会实时更新。</div>
                  </div>
                  <ImagePlus size={18} />
                </div>

                <div className="album-image-strip">
                  {pageImages.length > 0 ? (
                    pageImages.map((image, index) => (
                      <article
                        key={image.id}
                        className="album-image-chip"
                        data-active={editorImageId === image.id}
                        draggable
                        onDragStart={() => onSetDragImageId(image.id)}
                        onDragOver={(event) => event.preventDefault()}
                        onDrop={() => onMoveImageWithinPage(activeAlbumPage.id, dragImageId, image.id)}
                        onClick={() => onSetEditorImageId(image.id)}
                      >
                        <img src={image.thumbnailUrl} alt={image.fileName} />
                        <div className="album-image-chip-copy">
                          <strong>{index + 1}</strong>
                          <span>{image.fileName}</span>
                        </div>
                        <button
                          type="button"
                          className="album-image-chip-remove"
                          onClick={(event) => {
                            event.stopPropagation();
                            onRemoveImageFromPage(activeAlbumPage.id, image.id);
                          }}
                          aria-label={`从页面移除 ${image.fileName}`}
                        >
                          <X size={14} />
                        </button>
                      </article>
                    ))
                  ) : (
                    <div className="empty-state">这一页还没有照片。</div>
                  )}
                </div>

                <label className="setup-field">
                  <span>添加照片</span>
                  <select
                    value=""
                    onChange={(event) => {
                      if (event.target.value) {
                        onAddImageToPage(activeAlbumPage.id, event.target.value);
                      }
                    }}
                  >
                    <option value="">选择一张照片</option>
                    {availableImages.map((image) => (
                      <option key={image.id} value={image.id}>
                        {image.fileName}
                      </option>
                    ))}
                  </select>
                </label>
              </section>

              {selectedEditorImage ? (
                <section className="album-editor-section selected-photo-card">
                  <img src={imageDisplayUrl(selectedEditorImage)} alt={selectedEditorImage.fileName} />
                  <div>
                    <strong>{selectedEditorImage.fileName}</strong>
                    <span>
                      {selectedEditorImage.width} x {selectedEditorImage.height}
                    </span>
                  </div>
                </section>
              ) : null}

              {normalizePageType(activeAlbumPage.pageType, activeAlbumPage.layoutId) === "cover" && coverPage ? (
                <section className="album-editor-section cover-picker">
                  <div>
                    <div className="panel-title">封面照片</div>
                    <div className="panel-subtitle">从 AI 筛好的照片中换一张更有记忆锚点的封面。</div>
                  </div>
                  <div className="cover-choice-grid">
                    {selectedImages.slice(0, 12).map((image) => (
                      <button
                        type="button"
                        key={image.id}
                        className="cover-image-button"
                        onClick={() => onSetCoverImage(image.id)}
                        aria-label={`设为封面 ${image.fileName}`}
                      >
                        <img src={image.thumbnailUrl} alt={image.fileName} />
                      </button>
                    ))}
                  </div>
                </section>
              ) : null}
            </div>
          ) : (
            <div className="empty-state">选择一页开始编辑。</div>
          )}
        </aside>

      </div>
    </div>
  );
}
