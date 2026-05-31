"use client";

import { type ChangeEvent, type DragEvent, useEffect, useMemo, useState } from "react";
import {
  Album as AlbumIcon,
  BookOpen,
  ImageDown,
  Layers3,
  Link2,
  MoreHorizontal,
  Pencil,
  Settings,
  Share2,
  Trash2,
  Upload,
  X,
} from "lucide-react";

function GithubIcon(props: React.SVGProps<SVGSVGElement> & { size?: number }) {
  const { size = 24, ...rest } = props;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" {...rest}>
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
    </svg>
  );
}
import { ConfirmDialog, type ConfirmDialogTone } from "@/components/ConfirmDialog";
import { DialogShell } from "@/components/DialogShell";
import { AlbumEditorView } from "@/components/dashboard/AlbumEditorView";
import { ProjectCreateForm } from "@/components/dashboard/ProjectCreateForm";
import { ProjectHeader } from "@/components/dashboard/ProjectHeader";
import { MemoryMapView } from "@/components/dashboard/MemoryMapView";
import { ProjectSidebar } from "@/components/dashboard/ProjectSidebar";
import { ProjectSetupPanel } from "@/components/dashboard/ProjectSetupPanel";
import { ReviewView } from "@/components/dashboard/ReviewView";
import { ExportView } from "@/components/dashboard/ExportView";
import { SettingsView } from "@/components/dashboard/SettingsView";
import { GuideView } from "@/components/dashboard/GuideView";
import { WorkflowRail } from "@/components/dashboard/WorkflowRail";
import {
  createProject,
  deleteImage,
  deleteProject,
  exportAlbumArtifact,
  generateAlbum,
  generateImageEdit,
  getAISettings,
  getGitHubPublishProgress,
  getGitHubSettings,
  getProject,
  listProjects,
  publishAlbumListing,
  redoAlbum,
  startAnalysis,
  undoAlbum,
  updateAISettings,
  updateGitHubSettings,
  updateAlbum,
  updateImageDecision,
  updateProject,
  undoImageEdit,
  uploadImagesWithProgress,
} from "@/lib/api";
import { resolveCityPlace } from "@/lib/cityCoordinates";
import type { AISettings, Album as AlbumType, AlbumExport, AlbumPage, GitHubPublishProgress, GitHubSettings, ImageAsset, Project, ProjectPlace } from "@/lib/types";

type ProjectDraft = {
  title: string;
  description: string;
  location: string;
  tone: string;
  themeId: string;
};

type ReviewFilter = "all" | "keep" | "improve_then_keep" | "review" | "reject_suggested";
type ReviewSort = "recommended" | "story" | "quality" | "newest";
type ViewId = "setup" | "review" | "album" | "export" | "settings" | "guide";
type WorkspaceMode = "project" | "memory_map";
type NextActionId = "create" | "upload" | "analyze" | "review" | "generate" | "export" | "done";

type ReviewGroup = {
  key: string;
  images: ImageAsset[];
  duplicate: boolean;
  bestImage: ImageAsset;
  label: string;
  summary: string;
  latestAt: number;
  score: number;
};

type NextAction = {
  action: NextActionId;
  label: string;
  hint: string;
};

type UploadPhase = "idle" | "uploading" | "processing" | "done" | "failed";

type UploadState = {
  phase: UploadPhase;
  percent: number;
  loadedBytes: number;
  totalBytes: number;
  fileCount: number;
  message: string;
  fileNames: string[];
};

type ConfirmDialogState = {
  open: boolean;
  title: string;
  message: string;
  details: string[];
  confirmLabel: string;
  cancelLabel: string;
  pendingLabel: string;
  tone: ConfirmDialogTone;
  onConfirm?: () => void | Promise<void>;
};

type ConfirmDialogInput = {
  title: string;
  message: string;
  details?: string[];
  confirmLabel?: string;
  cancelLabel?: string;
  pendingLabel?: string;
  tone?: ConfirmDialogTone;
  onConfirm: () => void | Promise<void>;
};

const themeOptions = [
  {
    value: "film_travel",
    label: "胶片旅行",
    description: "暖色胶片、旅途章节、底片索引感",
    accent: "#c94c2c",
    surface: "#f3eadc",
    text: "#171311",
  },
  {
    value: "warm_family",
    label: "温暖家庭",
    description: "柔和留白、亲密叙事、像一本家族手札",
    accent: "#d86f57",
    surface: "#fff1e9",
    text: "#2a1f1b",
  },
  {
    value: "editorial",
    label: "纪实杂志",
    description: "黑白标题、强网格、杂志专题节奏",
    accent: "#111111",
    surface: "#f6f6f1",
    text: "#101010",
  },
  {
    value: "minimal_gallery",
    label: "极简画廊",
    description: "大留白、细边框、适合安静观看单张作品",
    accent: "#245f73",
    surface: "#fbfbf8",
    text: "#151717",
  },
  {
    value: "nocturne",
    label: "夜色电影",
    description: "深色电影感、霓虹点缀、适合夜景和情绪片",
    accent: "#f0b15f",
    surface: "#10171f",
    text: "#f5efe3",
  },
  {
    value: "botanical",
    label: "植物手札",
    description: "植物绿、纸张肌理、像旅行中夹进来的叶片",
    accent: "#4f7f52",
    surface: "#eef4e9",
    text: "#17231a",
  },
  {
    value: "postcard",
    label: "明信片旅行",
    description: "蓝色票据、珊瑚红标题、适合城市与远方",
    accent: "#e85d4f",
    surface: "#eef7fb",
    text: "#17324a",
  },
  {
    value: "archive",
    label: "旧物档案",
    description: "档案纸、编号感、把照片整理成可保存的证物",
    accent: "#8c4b35",
    surface: "#eee6d4",
    text: "#251f18",
  },
  {
    value: "cinematic_bw",
    label: "黑白剧场",
    description: "高反差黑白、舞台光、适合人物和关键瞬间",
    accent: "#d7d7d2",
    surface: "#0e0e0f",
    text: "#f3f3ef",
  },
  {
    value: "summer_diary",
    label: "夏日清单",
    description: "清亮蓝绿、日记贴纸感、轻快但不花哨",
    accent: "#f2a33a",
    surface: "#f2fbf8",
    text: "#163235",
  },
];

const reviewFilters: Array<{ value: ReviewFilter; label: string; tone: string }> = [
  { value: "all", label: "全部", tone: "accent" },
  { value: "keep", label: "推荐保留", tone: "good" },
  { value: "improve_then_keep", label: "优化后保留", tone: "warn" },
  { value: "review", label: "待确认", tone: "accent" },
  { value: "reject_suggested", label: "建议不入册", tone: "bad" },
];

const reviewSorts: Array<{ value: ReviewSort; label: string }> = [
  { value: "recommended", label: "推荐优先" },
  { value: "story", label: "故事价值" },
  { value: "quality", label: "技术质量" },
  { value: "newest", label: "最新导入" },
];

const viewTabs: Array<{ value: ViewId; label: string }> = [
  { value: "setup", label: "准备照片" },
  { value: "review", label: "AI 审核" },
  { value: "album", label: "设计相册" },
  { value: "export", label: "导出分享" },
];

const uploadAccept = "image/*,.heic,.heif,image/heic,image/heif";
const layoutOptions = [
  { value: "cover_full_bleed", label: "封面满版" },
  { value: "hero_story", label: "主视觉叙事" },
  { value: "single_photo_caption", label: "单图配文" },
  { value: "diptych", label: "双图对照" },
  { value: "triptych", label: "三图并列" },
  { value: "mosaic", label: "拼贴叙事" },
  { value: "gallery_wall", label: "画廊墙" },
  { value: "timeline_ribbon", label: "时间带" },
  { value: "full_bleed_quote", label: "满版诗页" },
  { value: "scrapbook", label: "手账拼贴" },
  { value: "contact_sheet", label: "胶片索引" },
  { value: "quote_break", label: "文字留白" },
  { value: "ending_text", label: "结尾文字" },
];

const exportOptions = [
  { value: "html", label: "HTML", icon: Link2 },
  { value: "long_image", label: "images", icon: ImageDown },
  { value: "share_link", label: "分享链接", icon: Share2 },
  { value: "github_pages", label: "GitHub Pages", icon: GithubIcon as any },
];

const emptyAISettings: AISettings = {
  baseUrl: "",
  apiKey: "",
  model: "gpt-4o-mini",
  imageBaseUrl: "",
  imageApiKey: "",
  imageModel: "gpt-image-1.5",
};

const emptyGitHubSettings: GitHubSettings = {
  owner: "",
  repo: "",
  branch: "main",
  token: "",
};

const emptyConfirmDialog: ConfirmDialogState = {
  open: false,
  title: "",
  message: "",
  details: [],
  confirmLabel: "确认",
  cancelLabel: "取消",
  pendingLabel: "处理中...",
  tone: "default",
};

type ImagePreviewOverride = {
  imageId: string;
  title: string;
  subtitle: string;
  url: string;
};

const emptySelectedImages: ImageAsset[] = [];

const initialDraft: ProjectDraft = {
  title: "集忆 · 胶片旅行",
  description: "整理一次旅程的筛选草稿，先保留有价值的瞬间。",
  location: "上海",
  tone: "film",
  themeId: "film_travel",
};

function statusTone(status?: string) {
  switch (status) {
    case "keep":
    case "approved":
    case "done":
      return "good";
    case "improve_then_keep":
    case "analyzing":
    case "generating_album":
    case "exporting":
      return "warn";
    case "reject_suggested":
    case "excluded":
    case "failed":
      return "bad";
    default:
      return "accent";
  }
}

function albumPageTypeForLayout(layoutId: string) {
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

function normalizeAlbumPageType(pageType: string, layoutId = "") {
  switch (pageType) {
    case "chapter":
    case "transition":
      return albumPageTypeForLayout(layoutId);
    case "opener":
    case "hero":
    case "detail":
    case "sequence":
    case "collage":
    case "gallery":
    case "pause":
    case "index":
    case "cover":
    case "ending":
      return pageType;
    default:
      return "detail";
  }
}

function fallbackAlbumPageTitle(pageType: string, layoutId: string, order: number) {
  switch (normalizeAlbumPageType(pageType, layoutId)) {
    case "cover":
      return "封面";
    case "opener":
      return "开场白";
    case "hero":
      return "第一眼记住的画面";
    case "detail":
      return layoutId === "diptych" || layoutId === "triptych" ? "并置的回声" : "被看见的细节";
    case "sequence":
      return "时间经过这里";
    case "collage":
      return "记忆碎片";
    case "gallery":
      return "一面记忆墙";
    case "pause":
      return "留白";
    case "index":
      return "这一卷的索引";
    case "ending":
      return "结尾";
    default:
      return `页面 ${order}`;
  }
}

function normalizeAlbumPageTitle(title: string, pageType: string, layoutId: string, order: number) {
  const trimmed = title.trim();
  const forbidden = ["无题", "章节", "第一章", "第二章", "第三章", "第四章", "第五章", "第六章", "第七章", "第八章", "第九章"];
  if (!trimmed || forbidden.some((token) => trimmed.includes(token))) {
    return fallbackAlbumPageTitle(pageType, layoutId, order);
  }
  return trimmed;
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

function analysisStatusLabel(status?: string) {
  switch (status) {
    case "idle":
    case "none":
      return "待分析";
    case "done":
      return "分析完成";
    case "running":
      return "分析中";
    case "failed":
      return "分析失败";
    case "pending":
      return "待分析";
    default:
      return status || "待分析";
  }
}

function reviewBucket(image: ImageAsset): ReviewFilter {
  switch (image.status) {
    case "keep":
    case "approved":
      return "keep";
    case "improve_then_keep":
      return "improve_then_keep";
    case "reject_suggested":
    case "excluded":
      return "reject_suggested";
    default:
      return "review";
  }
}

function formatCount(value: number) {
  return new Intl.NumberFormat("zh-CN").format(value);
}

function formatBytes(value: number) {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  const amount = value / 1024 ** index;
  return `${amount >= 10 || index === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[index]}`;
}

function emptyUploadState(): UploadState {
  return {
    phase: "idle",
    percent: 0,
    loadedBytes: 0,
    totalBytes: 0,
    fileCount: 0,
    message: "",
    fileNames: [],
  };
}

function isSupportedImageFile(file: File) {
  const name = file.name.toLowerCase();
  return file.type.startsWith("image/") || name.endsWith(".heic") || name.endsWith(".heif");
}

function toErrorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback;
}

function formatDateTime(value?: string) {
  if (!value) return "未记录";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "未记录";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function timeValue(value?: string) {
  if (!value) return 0;
  const valueOf = new Date(value).getTime();
  return Number.isNaN(valueOf) ? 0 : valueOf;
}

function scoreOf(image: ImageAsset) {
  if (!image.analysis) return 0;
  return (
    image.analysis.qualityScore +
    image.analysis.preservationScore +
    image.analysis.storyScore
  );
}

function sortWeight(image: ImageAsset, sortMode: ReviewSort) {
  const analysis = image.analysis;
  const total = scoreOf(image);
  const groupBonus = analysis?.similarGroupBest ? 500000 : analysis?.similarGroupId ? 1000 : 0;
  switch (sortMode) {
    case "story":
      return groupBonus + (analysis?.storyScore ?? 0) * 1000 + total;
    case "quality":
      return groupBonus + (analysis?.qualityScore ?? 0) * 1000 + total;
    case "newest":
      return groupBonus + timeValue(image.createdAt);
    default: {
      const bucket = reviewBucket(image);
      const bucketWeight =
        bucket === "keep"
          ? 400000
          : bucket === "improve_then_keep"
            ? 300000
            : bucket === "review"
              ? 200000
              : 0;
      return groupBonus + bucketWeight + total;
    }
  }
}

function compareImages(a: ImageAsset, b: ImageAsset, sortMode: ReviewSort) {
  const sortDiff = sortWeight(b, sortMode) - sortWeight(a, sortMode);
  if (sortDiff !== 0) return sortDiff;
  const scoreDiff = scoreOf(b) - scoreOf(a);
  if (scoreDiff !== 0) return scoreDiff;
  return timeValue(b.createdAt) - timeValue(a.createdAt);
}

function searchTextForImage(image: ImageAsset) {
  const analysis = image.analysis;
  return [
    image.fileName,
    image.status,
    image.userDecision,
    analysis?.similarGroupLabel,
    analysis?.similarGroupReason,
    analysis?.albumRole,
    analysis?.socialCaption,
    ...(analysis?.reasons ?? []),
    ...(analysis?.captionSeeds ?? []),
    ...(analysis?.editSuggestions?.flatMap((suggestion) => [
      suggestion.reason,
      suggestion.actionLabel ?? "",
      suggestion.execution ?? "",
    ]) ?? []),
    ...(analysis?.detectedContent?.scenes ?? []),
    ...(analysis?.detectedContent?.objects ?? []),
    ...(analysis?.detectedContent?.mood ?? []),
    ...(analysis?.detectedContent?.tags ?? []),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

function imageDisplayUrl(image: ImageAsset) {
  return image.derivedUrl || image.originalUrl || image.thumbnailUrl;
}

function imageHasCompletedAnalysis(image: ImageAsset) {
  const completedAt = image.analysis?.completedAt;
  return Boolean(completedAt && !completedAt.startsWith("0001-01-01"));
}

function imageNeedsAnalysis(image: ImageAsset, project: Project | null) {
  if (!imageHasCompletedAnalysis(image)) {
    return true;
  }
  const currentModelVersion = project?.currentAnalysisModelVersion?.trim();
  const currentPromptVersion = project?.currentAnalysisPromptVersion?.trim();
  if (!currentModelVersion && !currentPromptVersion) {
    return false;
  }
  const analysis = image.analysis;
  if (!analysis) {
    return true;
  }
  if (currentModelVersion && analysis.modelVersion !== currentModelVersion) {
    return true;
  }
  if (currentPromptVersion && analysis.promptVersion !== currentPromptVersion) {
    return true;
  }
  return false;
}

function isAIPreselectedImage(image: ImageAsset) {
  if (image.status === "approved" || image.status === "excluded") {
    return false;
  }
  if (image.status !== "keep" && image.status !== "improve_then_keep") {
    return false;
  }
  if (image.analysis?.similarGroupId && image.analysis.similarGroupBest === false) {
    return false;
  }
  return true;
}

function imageReasonSummary(image: ImageAsset) {
  if (image.analysis?.similarGroupBest && image.analysis.similarGroupReason) {
    return `本组最佳：${image.analysis.similarGroupReason}`;
  }
  if (image.analysis?.similarGroupId && image.analysis?.similarGroupBest === false) {
    return image.analysis.similarGroupReason || "AI 判断为相似组里的替代照片";
  }
  return image.analysis?.reasons?.[0] ?? image.analysis?.captionSeeds?.[0] ?? "等待分析结果";
}

function cloneAlbumForDraft(album: AlbumType | null): AlbumType | null {
  if (!album) return null;
  return {
    ...album,
    pages: album.pages
      .map((page) => ({ ...page, imageIds: [...page.imageIds] }))
      .sort((a, b) => a.order - b.order),
    editHistory: album.editHistory ? [...album.editHistory] : undefined,
    redoStack: album.redoStack ? [...album.redoStack] : undefined,
  };
}

function normalizeDraftPages(pages: AlbumPage[]) {
  return pages.map((page, index) => ({
    ...page,
    pageType: normalizeAlbumPageType(page.pageType, page.layoutId),
    title: normalizeAlbumPageTitle(page.title, page.pageType, page.layoutId, index + 1),
    order: index + 1,
  }));
}

function groupKeyForImage(image: ImageAsset) {
  return image.analysis?.similarGroupId || image.averageHash || image.id;
}

function isAlbumExported(project: Project | null, activeExport: AlbumExport | null) {
  return Boolean(activeExport || project?.status === "done");
}

function defaultViewForProject(project: Project | null): ViewId {
  if (!project) return "setup";
  if (project.images.length === 0) return "setup";
  if (project.status === "analyzing" || project.analysisStatus !== "done") return "setup";
  if (!project.album) return "review";
  if (project.status === "done") return "export";
  return "album";
}

function viewIcon(viewId: ViewId) {
  switch (viewId) {
    case "setup":
      return <Upload size={16} />;
    case "review":
      return <Layers3 size={16} />;
    case "album":
      return <AlbumIcon size={16} />;
    case "export":
      return <Link2 size={16} />;
    case "settings":
      return <Settings size={16} />;
  }
}

export function Dashboard() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [draft, setDraft] = useState<ProjectDraft>(initialDraft);
  const [projectTitleDraft, setProjectTitleDraft] = useState<string>("");
  const [projectRenaming, setProjectRenaming] = useState(false);
  const [busy, setBusy] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [exportInfo, setExportInfo] = useState<AlbumExport | null>(null);
  const [reviewFilter, setReviewFilter] = useState<ReviewFilter>("all");
  const [reviewSort, setReviewSort] = useState<ReviewSort>("recommended");
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [selectedImageIds, setSelectedImageIds] = useState<string[]>([]);
  const [activeImageId, setActiveImageId] = useState<string>("");
  const [previewImageId, setPreviewImageId] = useState<string>("");
  const [previewOverride, setPreviewOverride] = useState<ImagePreviewOverride | null>(null);
  const [albumDraft, setAlbumDraft] = useState<AlbumType | null>(null);
  const [albumPageId, setAlbumPageId] = useState<string>("");
  const [editorImageId, setEditorImageId] = useState<string>("");
  const [dragPageId, setDragPageId] = useState<string>("");
  const [dragImageId, setDragImageId] = useState<string>("");
  const [aiSettings, setAISettings] = useState<AISettings>(emptyAISettings);
  const [settingsLoaded, setSettingsLoaded] = useState(false);
  const [settingsNote, setSettingsNote] = useState("");
  const [githubSettings, setGitHubSettings] = useState<GitHubSettings>(emptyGitHubSettings);
  const [githubSettingsLoaded, setGitHubSettingsLoaded] = useState(false);
  const [githubSettingsNote, setGitHubSettingsNote] = useState("");
  const [githubPublishProgress, setGithubPublishProgress] = useState<GitHubPublishProgress | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState>(emptyConfirmDialog);
  const [showOnboarding, setShowOnboarding] = useState(false);
  const [activeView, setActiveView] = useState<ViewId>("setup");
  const [workspaceMode, setWorkspaceMode] = useState<WorkspaceMode>("project");
  const [uploadState, setUploadState] = useState<UploadState>(() => emptyUploadState());
  const [showProjectDrawer, setShowProjectDrawer] = useState(false);
  const [isCreatingProject, setIsCreatingProject] = useState(false);
  const [editingGalleryProject, setEditingGalleryProject] = useState<Project | null>(null);
  const [galleryEditDraft, setGalleryEditDraft] = useState<ProjectDraft>(initialDraft);
  const [openMenuProjectId, setOpenMenuProjectId] = useState<string>("");

  const selectedProject = useMemo(
    () => projects.find((project) => project.id === selectedProjectId) ?? null,
    [projects, selectedProjectId],
  );

  const selectedAlbum: AlbumType | null = selectedProject?.album ?? null;
  const selectedImages = selectedProject?.images ?? emptySelectedImages;
  const activeExport = exportInfo?.projectId === selectedProject?.id ? exportInfo : null;
  const selectedSet = useMemo(() => new Set(selectedImageIds), [selectedImageIds]);
  const isUploading = uploadState.phase === "uploading" || uploadState.phase === "processing";
  const isDeletingImage = busy.startsWith("delete:");

  const openConfirmDialog = (input: ConfirmDialogInput) => {
    setConfirmDialog({
      open: true,
      title: input.title,
      message: input.message,
      details: input.details ?? [],
      confirmLabel: input.confirmLabel ?? "确认",
      cancelLabel: input.cancelLabel ?? "取消",
      pendingLabel: input.pendingLabel ?? "处理中...",
      tone: input.tone ?? "default",
      onConfirm: input.onConfirm,
    });
  };

  const closeConfirmDialog = () => {
    setConfirmDialog(emptyConfirmDialog);
  };

  const handleConfirmDialogConfirm = async () => {
    const action = confirmDialog.onConfirm;
    if (!action) return;
    await action();
  };

  const patchProject = (updated: Project) => {
    setProjects((current) =>
      current.map((project) => (project.id === updated.id ? updated : project)),
    );
  };

  const patchImage = (updatedImage: ImageAsset) => {
    setProjects((current) =>
      current.map((project) =>
        project.id !== updatedImage.projectId
          ? project
          : {
              ...project,
              images: project.images.map((image) =>
                image.id === updatedImage.id ? updatedImage : image,
              ),
            },
      ),
    );
  };

  const refresh = async () => {
    const items = await listProjects();
    setProjects(items);
    if (!selectedProjectId && items.length > 0) {
      setSelectedProjectId(items[0].id);
    }
    if (selectedProjectId) {
      const updated = await getProject(selectedProjectId).catch(() => null);
      if (updated) patchProject(updated);
    }
  };

  useEffect(() => {
    refresh().catch((err: Error) => setError(err.message));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    setShowOnboarding(window.localStorage.getItem("memoir:onboarding-dismissed") !== "true");
  }, []);

  // Close context menu when clicking outside
  useEffect(() => {
    if (!openMenuProjectId) return;
    const handleClickOutside = () => setOpenMenuProjectId("");
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpenMenuProjectId("");
    };
    document.addEventListener("click", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("click", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [openMenuProjectId]);

  useEffect(() => {
    setSelectedImageIds([]);
    setActiveImageId("");
    setPreviewImageId("");
    setAlbumDraft(null);
    setAlbumPageId("");
    setEditorImageId("");
    setReviewFilter("all");
    setReviewSort("recommended");
    setSearchTerm("");
    setExportInfo(null);
    setProjectRenaming(false);
    setUploadState(emptyUploadState());
  }, [selectedProjectId]);

  useEffect(() => {
    if (!selectedProject || selectedProject.analysisStatus !== "done") return;
    setSelectedImageIds(selectedImages.filter(isAIPreselectedImage).map((image) => image.id));
  }, [selectedProject?.id, selectedProject?.analysisStatus, selectedImages]);

  useEffect(() => {
    setProjectTitleDraft(selectedProject?.title ?? "");
    setProjectRenaming(false);
  }, [selectedProject?.id]);

  useEffect(() => {
    const imageIds = new Set(selectedImages.map((image) => image.id));
    setSelectedImageIds((current) => {
      const next = current.filter((id) => imageIds.has(id));
      if (next.length === current.length) return current;
      return next;
    });
    if (previewImageId && !imageIds.has(previewImageId)) {
      setPreviewImageId("");
      setPreviewOverride(null);
    }
  }, [previewImageId, selectedImages]);

  useEffect(() => {
    setActiveView(defaultViewForProject(selectedProject));
  }, [selectedProject?.id]);

  useEffect(() => {
    if (!selectedProject || selectedProject.status !== "analyzing") {
      return;
    }

    const timer = window.setInterval(async () => {
      const updated = await getProject(selectedProject.id).catch(() => null);
      if (!updated) return;
      patchProject(updated);
      if (updated.status !== "analyzing") {
        window.clearInterval(timer);
      }
    }, 2500);

    return () => window.clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProject?.id, selectedProject?.status]);

  useEffect(() => {
    if (!selectedProject || selectedProject.status !== "generating_album") {
      return;
    }

    const timer = window.setInterval(async () => {
      const updated = await getProject(selectedProject.id).catch(() => null);
      if (!updated) return;
      patchProject(updated);
      if (updated.status !== "generating_album") {
        window.clearInterval(timer);
        setBusy("");
        if (updated.status === "editing" && updated.album) {
          setActiveView("album");
        } else if (updated.status === "failed") {
          setError(updated.lastError || "相册生成失败");
        }
      }
    }, 2500);

    return () => window.clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedProject?.id, selectedProject?.status]);

  const filterCounts = useMemo(() => {
    const counts: Record<ReviewFilter, number> = {
      all: selectedImages.length,
      keep: 0,
      improve_then_keep: 0,
      review: 0,
      reject_suggested: 0,
    };
    selectedImages.forEach((image) => {
      counts[reviewBucket(image)] += 1;
    });
    return counts;
  }, [selectedImages]);

  const filteredImages = useMemo(() => {
    const normalizedSearch = searchTerm.trim().toLowerCase();
    return selectedImages.filter((image) => {
      if (reviewFilter !== "all" && reviewBucket(image) !== reviewFilter) {
        return false;
      }
      if (!normalizedSearch) {
        return true;
      }
      return searchTextForImage(image).includes(normalizedSearch);
    });
  }, [reviewFilter, searchTerm, selectedImages]);

  const reviewGroups = useMemo<ReviewGroup[]>(() => {
    const groups = new Map<string, ImageAsset[]>();
    filteredImages.forEach((image) => {
      const key = groupKeyForImage(image);
      const next = groups.get(key) ?? [];
      next.push(image);
      groups.set(key, next);
    });

    const items = Array.from(groups.entries()).map(([key, images]) => {
      const sorted = [...images].sort((a, b) => compareImages(a, b, reviewSort));
      const bestImage = sorted.find((image) => image.analysis?.similarGroupBest) ?? sorted[0];
      const label =
        bestImage.analysis?.similarGroupLabel || (sorted.length > 1 ? "相似照片组" : "单张照片");
      const summary =
        bestImage.analysis?.similarGroupReason ||
        (sorted.length > 1
          ? "AI 已挑出本组最值得保留的一张，其余作为替代图"
          : imageReasonSummary(bestImage));
      return {
        key,
        images: sorted,
        duplicate: sorted.length > 1,
        bestImage,
        label,
        summary,
        latestAt: Math.max(...sorted.map((image) => timeValue(image.createdAt))),
        score: scoreOf(bestImage),
      };
    });

    return items.sort((a, b) => {
      const sortDiff = sortWeight(b.bestImage, reviewSort) - sortWeight(a.bestImage, reviewSort);
      if (sortDiff !== 0) return sortDiff;
      if (a.duplicate !== b.duplicate) return a.duplicate ? -1 : 1;
      const scoreDiff = b.score - a.score;
      if (scoreDiff !== 0) return scoreDiff;
      return b.latestAt - a.latestAt;
    });
  }, [filteredImages, reviewSort]);

  const visibleImages = useMemo(
    () => reviewGroups.flatMap((group) => group.images),
    [reviewGroups],
  );
  const visibleImageIds = useMemo(
    () => visibleImages.map((image) => image.id),
    [visibleImages],
  );

  const visibleIdSet = useMemo(() => new Set(visibleImageIds), [visibleImageIds]);
  const selectedVisibleCount = selectedImageIds.filter((id) => visibleIdSet.has(id)).length;
  const duplicateGroupCount = reviewGroups.filter((group) => group.duplicate).length;

  useEffect(() => {
    if (visibleImageIds.length === 0) {
      setActiveImageId("");
      return;
    }
    if (!visibleImageIds.includes(activeImageId)) {
      setActiveImageId(visibleImageIds[0]);
    }
  }, [activeImageId, visibleImageIds]);

  useEffect(() => {
    const nextDraft = cloneAlbumForDraft(selectedAlbum);
    setAlbumDraft(nextDraft);
    if (!nextDraft?.pages.length) {
      setAlbumPageId("");
      setEditorImageId("");
      return;
    }
    setAlbumPageId((current) =>
      current && nextDraft.pages.some((page) => page.id === current)
        ? current
        : nextDraft.pages[0].id,
    );
    setEditorImageId((current) => {
      if (current && selectedImages.some((image) => image.id === current)) return current;
      return nextDraft.pages.find((page) => page.imageIds.length > 0)?.imageIds[0] ?? "";
    });
  }, [selectedAlbum, selectedImages]);

  const activeAlbumPage = useMemo(
    () => albumDraft?.pages.find((page) => page.id === albumPageId) ?? albumDraft?.pages[0] ?? null,
    [albumDraft, albumPageId],
  );
  const albumDirty = useMemo(() => {
    if (!selectedAlbum || !albumDraft) return false;
    return (
      JSON.stringify({
        title: selectedAlbum.title,
        intro: selectedAlbum.intro,
        themeId: selectedAlbum.themeId,
        pages: normalizeDraftPages(selectedAlbum.pages),
      }) !==
      JSON.stringify({
        title: albumDraft.title,
        intro: albumDraft.intro,
        themeId: albumDraft.themeId,
        pages: normalizeDraftPages(albumDraft.pages),
      })
    );
  }, [albumDraft, selectedAlbum]);

  useEffect(() => {
    if (!activeAlbumPage) return;
    if (editorImageId && activeAlbumPage.imageIds.includes(editorImageId)) return;
    setEditorImageId(activeAlbumPage.imageIds[0] ?? editorImageId);
  }, [activeAlbumPage, editorImageId]);

  useEffect(() => {
    if (settingsLoaded) return;
    getAISettings()
      .then((settings) => {
        setAISettings({
          ...emptyAISettings,
          ...settings,
          model: settings.model || emptyAISettings.model,
          imageBaseUrl: settings.imageBaseUrl ?? emptyAISettings.imageBaseUrl,
          imageApiKey: settings.imageApiKey ?? emptyAISettings.imageApiKey,
          imageModel: settings.imageModel || emptyAISettings.imageModel,
        });
        setSettingsLoaded(true);
      })
      .catch((err: Error) => {
        setSettingsNote(err.message || "AI 设置读取失败");
        setSettingsLoaded(true);
      });
  }, [settingsLoaded]);

  useEffect(() => {
    if (githubSettingsLoaded) return;
    getGitHubSettings()
      .then((settings) => {
        setGitHubSettings({
          ...emptyGitHubSettings,
          ...settings,
          branch: settings.branch || emptyGitHubSettings.branch,
        });
        setGitHubSettingsLoaded(true);
      })
      .catch((err: Error) => {
        setGitHubSettingsNote(err.message || "GitHub 设置读取失败");
        setGitHubSettingsLoaded(true);
      });
  }, [githubSettingsLoaded]);

  const activeImage = useMemo(
    () => selectedImages.find((image) => image.id === activeImageId) ?? null,
    [activeImageId, selectedImages],
  );
  const activeTask = selectedProject?.activeTask ?? null;

  const activeImageIndex = useMemo(
    () => visibleImageIds.indexOf(activeImageId),
    [activeImageId, visibleImageIds],
  );
  const previewImage = useMemo(
    () => selectedImages.find((image) => image.id === previewImageId) ?? null,
    [previewImageId, selectedImages],
  );
  const pendingAnalysisCount = useMemo(
    () =>
      selectedProject?.pendingAnalysisCount ??
      selectedImages.filter((image) => imageNeedsAnalysis(image, selectedProject)).length,
    [selectedImages, selectedProject],
  );
  const staleAnalysisCount = selectedProject?.staleAnalysisCount ?? 0;

  const stats = useMemo(
    () => ({
      count: selectedImages.length,
      keep: filterCounts.keep,
      improve: filterCounts.improve_then_keep,
      review: filterCounts.review,
      reject: filterCounts.reject_suggested,
    }),
    [filterCounts, selectedImages.length],
  );

  const manuallyHandledCount = useMemo(
    () =>
      selectedImages.filter(
        (image) =>
          image.status === "approved" ||
          image.status === "excluded" ||
          image.userDecision === "keep" ||
          image.userDecision === "exclude" ||
          image.userDecision === "crop_applied",
      ).length,
    [selectedImages],
  );

  const nextAction = useMemo<NextAction>(() => {
    if (!selectedProject) {
      return {
        action: "create",
        label: "创建项目",
        hint: "先建立一个相册工作区",
      };
    }
    if (selectedImages.length === 0) {
      return {
        action: "upload",
        label: "导入照片",
        hint: "添加照片后再交给 AI 分析",
      };
    }
    if (selectedProject.status === "analyzing") {
      return {
        action: "done",
        label: "等待分析完成",
        hint: `当前进度 ${selectedProject.analysisProgress}%`,
      };
    }
    if (pendingAnalysisCount > 0) {
      return {
        action: "analyze",
        label:
          staleAnalysisCount > 0
            ? `重新分析 ${formatCount(pendingAnalysisCount)} 张旧结果`
            : `分析 ${formatCount(pendingAnalysisCount)} 张新增照片`,
        hint:
          staleAnalysisCount > 0
            ? `AI 配置或提示词已更新，${formatCount(staleAnalysisCount)} 张照片需要刷新结果。`
            : "只分析本次新导入的照片，并行处理会更快完成。",
      };
    }
    if (selectedProject.analysisStatus !== "done") {
      return {
        action: "analyze",
        label: "开始 AI 分析",
        hint: "让大模型先完成筛选、评分和优化建议",
      };
    }
    if (!selectedAlbum && manuallyHandledCount === 0) {
      return {
        action: "review",
        label: "进入照片审核",
        hint: "确认保留、不入册和需要优化的照片",
      };
    }
    if (!selectedAlbum) {
      return {
        action: "generate",
        label: "生成相册",
        hint: "使用已确认照片生成版式和配文",
      };
    }
    if (!isAlbumExported(selectedProject, activeExport)) {
      return {
        action: "export",
        label: "导出分享",
        hint: "生成 HTML、图片或分享链接",
      };
    }
    return {
      action: "done",
      label: "相册已完成",
      hint: "可以继续调整主题后重新生成",
    };
  }, [
    activeExport,
    manuallyHandledCount,
    pendingAnalysisCount,
    selectedAlbum,
    selectedImages.length,
    selectedProject,
    staleAnalysisCount,
  ]);

  const workflowStages = useMemo(
    () =>
      viewTabs.map((tab) => {
        const state =
          tab.value === activeView
            ? "active"
            : tab.value === "setup" && selectedImages.length > 0
              ? "done"
              : tab.value === "review" && selectedProject?.analysisStatus === "done"
                ? "done"
                : tab.value === "album" && selectedAlbum
                  ? "done"
                  : tab.value === "export" && isAlbumExported(selectedProject, activeExport)
                    ? "done"
                    : "idle";

        const description =
          tab.value === "setup"
            ? selectedImages.length > 0
              ? `${formatCount(selectedImages.length)} 张已导入`
              : "导入 JPG、PNG、HEIC"
            : tab.value === "review"
              ? selectedProject?.analysisStatus === "done"
                ? `${formatCount(stats.keep + stats.improve)} 张推荐入册`
                : analysisStatusLabel(selectedProject?.analysisStatus)
                : tab.value === "album"
                  ? selectedAlbum
                    ? `${formatCount(selectedAlbum.pages.length)} 页草稿`
                    : "选择主题并生成"
                : activeExport || selectedProject?.status === "done"
                  ? "导出结果已就绪"
                  : "导出 HTML、图片或分享链接";

        return {
          ...tab,
          icon: viewIcon(tab.value),
          state,
          description,
        };
      }),
    [activeExport, activeView, selectedAlbum, selectedImages.length, selectedProject, stats.improve, stats.keep],
  );

  const handleCreateProject = async () => {
    setBusy("create");
    setError("");
    try {
      const created = await createProject({
        ...draft,
        place: resolveCityPlace(draft.location) ?? undefined,
      });
      setProjects((current) => [created, ...current.filter((item) => item.id !== created.id)]);
      setSelectedProjectId(created.id);
      setActiveView("setup");
      setWorkspaceMode("project");
      setIsCreatingProject(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败");
    } finally {
      setBusy("");
    }
  };

  const handleUpload = async (files: FileList | File[] | null) => {
    if (!selectedProject || !files?.length) return;
    const pickedFiles = Array.from(files).filter((file) => isSupportedImageFile(file));
    if (pickedFiles.length === 0) {
      setError("请选择 JPG、PNG、HEIC 或 HEIF images。");
      return;
    }

    const rejectedCount = Array.from(files).length - pickedFiles.length;
    setBusy("upload");
    setError("");
    setUploadState({
      phase: "uploading",
      percent: 0,
      loadedBytes: 0,
      totalBytes: pickedFiles.reduce((total, file) => total + file.size, 0),
      fileCount: pickedFiles.length,
      message:
        rejectedCount > 0
          ? `已忽略 ${rejectedCount} 个非图片文件，正在上传 ${pickedFiles.length} 张照片`
          : `正在上传 ${pickedFiles.length} 张照片`,
      fileNames: pickedFiles.slice(0, 4).map((file) => file.name),
    });
    try {
      await uploadImagesWithProgress(selectedProject.id, pickedFiles, {
        onProgress: (progress) => {
          setUploadState((current) => ({
            ...current,
            phase: progress.percent >= 100 ? "processing" : "uploading",
            percent: progress.percent || current.percent,
            loadedBytes: progress.loaded,
            totalBytes: progress.total || current.totalBytes,
            message:
              progress.percent >= 100
                ? "上传已完成，正在转换 HEIC 并生成缩略图"
                : `正在上传 ${progress.percent || 0}%`,
          }));
        },
      });
      setUploadState((current) => ({
        ...current,
        phase: "processing",
        percent: 100,
        message: "上传完成，正在同步项目状态",
      }));
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
      setUploadState({
        phase: "done",
        percent: 100,
        loadedBytes: pickedFiles.reduce((total, file) => total + file.size, 0),
        totalBytes: pickedFiles.reduce((total, file) => total + file.size, 0),
        fileCount: pickedFiles.length,
        message: "导入完成，可以继续发起 AI 分析",
        fileNames: pickedFiles.slice(0, 4).map((file) => file.name),
      });
      setActiveView("setup");
    } catch (err) {
      const message = toErrorMessage(err, "上传失败");
      setUploadState((current) => ({
        ...current,
        phase: "failed",
        message,
      }));
      setError(message);
    } finally {
      setBusy("");
    }
  };

  const handleUploadInputChange = (event: ChangeEvent<HTMLInputElement>) => {
    const files = event.currentTarget.files ? Array.from(event.currentTarget.files) : [];
    event.currentTarget.value = "";
    void handleUpload(files);
  };

  const handleDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    const files = event.dataTransfer.files ? Array.from(event.dataTransfer.files) : [];
    if (files.length > 0) {
      void handleUpload(files);
    }
  };

  const refreshProject = async () => {
    if (!selectedProject) return;
    setBusy("refresh");
    setError("");
    try {
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "刷新失败");
    } finally {
      setBusy("");
    }
  };

  const handleThemeChange = async (themeId: string) => {
    if (!selectedProject || selectedProject.themeId === themeId) return;
    setBusy("theme");
    setError("");
    try {
      const updated = await updateProject(selectedProject.id, { themeId });
      patchProject(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "主题更新失败");
    } finally {
      setBusy("");
    }
  };

  const handleAnalyze = async () => {
    if (!selectedProject) return;
    setBusy("analyze");
    setError("");
    try {
      const updated = await startAnalysis(selectedProject.id);
      patchProject(updated);
      setActiveView("setup");
    } catch (err) {
      setError(err instanceof Error ? err.message : "分析失败");
    } finally {
      setBusy("");
    }
  };

  const performGenerate = async () => {
    if (!selectedProject) return;
    setBusy("generate");
    setError("");
    try {
      const updated = await generateAlbum(selectedProject.id);
      patchProject(updated);
      setSelectedProjectId(updated.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "生成相册失败");
      setBusy("");
    }
  };

  const handleGenerate = () => {
    if (!selectedProject) return;
    if (!selectedAlbum) {
      void performGenerate();
      return;
    }

    openConfirmDialog({
      title: "重新生成相册？",
      message:
        "AI 会基于当前保留照片重新生成相册结构、配文和版式，现有相册草稿会被新的结果替换。",
      details: [
        `当前相册：${selectedAlbum.title}`,
        `已有页面：${selectedAlbum.pages.length} 页`,
        albumDirty ? "当前有未保存编辑，重新生成后不会保留这些微调" : "当前编辑已保存",
      ],
      confirmLabel: "重新生成",
      cancelLabel: "保留当前草稿",
      pendingLabel: "生成中...",
      tone: "warning",
      onConfirm: performGenerate,
    });
  };

  const applyDecisionToImage = async (imageId: string, decision: string) => {
    const updatedImage = await updateImageDecision(imageId, decision);
    patchImage(updatedImage);
    return updatedImage;
  };

  const handleDecision = async (imageId: string, decision: string) => {
    setError("");
    try {
      await applyDecisionToImage(imageId, decision);
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新失败");
    }
  };

  const goToAdjacentImage = (direction: -1 | 1) => {
    if (visibleImageIds.length === 0) return;
    const currentIndex = activeImageIndex >= 0 ? activeImageIndex : 0;
    const nextIndex = Math.min(
      Math.max(currentIndex + direction, 0),
      visibleImageIds.length - 1,
    );
    setActiveImageId(visibleImageIds[nextIndex]);
  };

  const toggleImageSelection = (imageId: string) => {
    setSelectedImageIds((current) =>
      current.includes(imageId)
        ? current.filter((id) => id !== imageId)
        : [...current, imageId],
    );
  };

  const selectVisibleImages = () => {
    setSelectedImageIds((current) => Array.from(new Set([...current, ...visibleImageIds])));
  };

  const clearSelection = () => {
    setSelectedImageIds([]);
  };

  const performDeleteImage = async (imageId: string, projectId: string) => {
    setBusy(`delete:${imageId}`);
    setError("");
    try {
      await deleteImage(imageId);
      setSelectedImageIds((current) => current.filter((id) => id !== imageId));
      if (activeImageId === imageId) {
        setActiveImageId("");
      }
      if (previewImageId === imageId) {
        setPreviewImageId("");
      }
      const updated = await getProject(projectId);
      patchProject(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "删除失败");
      throw err;
    } finally {
      setBusy("");
    }
  };

  const handleDeleteImage = (imageId: string) => {
    if (!selectedProject || selectedProject.status === "analyzing") return;
    const image = selectedProject.images.find((item) => item.id === imageId);
    if (!image) return;

    openConfirmDialog({
      title: "删除这张照片？",
      message: `确定删除「${image.fileName}」吗？这个操作会同时移除预览和相册里的引用，删除后不可恢复。`,
      details: [
        `文件：${image.fileName}`,
        `分辨率：${image.width} × ${image.height}`,
        `大小：${formatBytes(image.fileSize)}`,
      ],
      confirmLabel: "删除照片",
      cancelLabel: "再想想",
      pendingLabel: "删除中...",
      tone: "danger",
      onConfirm: () => performDeleteImage(image.id, selectedProject.id),
    });
  };

  const performBatchDecision = async (
    decision: "keep" | "exclude",
    imageIds: string[],
  ) => {
    setBusy(decision === "keep" ? "batch_keep" : "batch_exclude");
    setError("");
    try {
      for (const imageId of imageIds) {
        await applyDecisionToImage(imageId, decision);
      }
      setSelectedImageIds([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : "批量更新失败");
    } finally {
      setBusy("");
    }
  };

  const handleBatchDecision = (decision: "keep" | "exclude") => {
    if (!selectedProject || selectedImageIds.length === 0) return;
    const validIds = selectedImageIds.filter((id) =>
      selectedProject.images.some((image) => image.id === id),
    );
    if (validIds.length === 0) return;

    if (decision === "exclude") {
      openConfirmDialog({
        title: "批量标记为不入册？",
        message:
          "这些照片会从 AI 推荐入册列表中移出，后续生成相册时默认不会使用。你仍然可以在审核台里重新保留它们。",
        details: [
          `本次选择：${formatCount(validIds.length)} 张`,
          `当前项目：${selectedProject.title}`,
          "不会删除原始照片文件",
        ],
        confirmLabel: "批量不入册",
        cancelLabel: "继续审核",
        pendingLabel: "更新中...",
        tone: "warning",
        onConfirm: () => performBatchDecision("exclude", validIds),
      });
      return;
    }

    void performBatchDecision("keep", validIds);
  };

  const patchAlbumInProject = (album: AlbumType) => {
    setProjects((current) =>
      current.map((project) =>
        project.id === album.projectId
          ? { ...project, themeId: album.themeId, album }
          : project,
      ),
    );
  };

  const patchAlbumPage = (pageId: string, patch: Partial<AlbumPage>) => {
    setAlbumDraft((current) =>
      current
        ? {
            ...current,
            pages: current.pages.map((page) =>
              page.id === pageId ? { ...page, ...patch } : page,
            ),
          }
        : current,
    );
  };

  const moveAlbumPage = (sourcePageId: string, targetPageId: string) => {
    if (!sourcePageId || !targetPageId || sourcePageId === targetPageId) return;
    setAlbumDraft((current) => {
      if (!current) return current;
      const pages = [...current.pages];
      const sourceIndex = pages.findIndex((page) => page.id === sourcePageId);
      const targetIndex = pages.findIndex((page) => page.id === targetPageId);
      if (sourceIndex < 0 || targetIndex < 0) return current;
      const [moving] = pages.splice(sourceIndex, 1);
      pages.splice(targetIndex, 0, moving);
      return { ...current, pages: normalizeDraftPages(pages) };
    });
    setAlbumPageId(sourcePageId);
    setDragPageId("");
  };

  const handleDeleteAlbumPage = (pageId: string) => {
    if (!pageId) return;
    setAlbumDraft((current) => {
      if (!current || current.pages.length <= 1) return current;
      const pageIndex = current.pages.findIndex((page) => page.id === pageId);
      if (pageIndex < 0) return current;
      const nextPages = normalizeDraftPages(current.pages.filter((page) => page.id !== pageId));
      const nextSelectedPage =
        nextPages[Math.min(pageIndex, nextPages.length - 1)] ?? nextPages[0] ?? null;
      setAlbumPageId((currentPageId) =>
        currentPageId === pageId ? nextSelectedPage?.id ?? "" : currentPageId,
      );
      setEditorImageId((currentImageId) => {
        if (!currentImageId) return currentImageId;
        const stillUsed = nextPages.some((page) => page.imageIds.includes(currentImageId));
        return stillUsed ? currentImageId : nextSelectedPage?.imageIds[0] ?? "";
      });
      return { ...current, pages: nextPages };
    });
  };

  const moveImageWithinPage = (pageId: string, sourceImageId: string, targetImageId: string) => {
    if (!sourceImageId || !targetImageId || sourceImageId === targetImageId) return;
    setAlbumDraft((current) => {
      if (!current) return current;
      return {
        ...current,
        pages: current.pages.map((page) => {
          if (page.id !== pageId) return page;
          const imageIds = [...page.imageIds];
          const sourceIndex = imageIds.indexOf(sourceImageId);
          const targetIndex = imageIds.indexOf(targetImageId);
          if (sourceIndex < 0 || targetIndex < 0) return page;
          const [moving] = imageIds.splice(sourceIndex, 1);
          imageIds.splice(targetIndex, 0, moving);
          return { ...page, imageIds };
        }),
      };
    });
    setEditorImageId(sourceImageId);
    setDragImageId("");
  };

  const handleAddImageToPage = (pageId: string, imageId: string) => {
    if (!imageId) return;
    setAlbumDraft((current) =>
      current
        ? {
            ...current,
            pages: current.pages.map((page) =>
              page.id === pageId
                ? {
                    ...page,
                    imageIds: page.imageIds.includes(imageId)
                      ? page.imageIds
                      : [...page.imageIds, imageId],
                  }
                : page,
            ),
          }
        : current,
    );
    setEditorImageId(imageId);
  };

  const handleRemoveImageFromPage = (pageId: string, imageId: string) => {
    setAlbumDraft((current) =>
      current
        ? {
            ...current,
            pages: current.pages.map((page) =>
              page.id === pageId
                ? { ...page, imageIds: page.imageIds.filter((id) => id !== imageId) }
                : page,
            ),
          }
        : current,
    );
    if (editorImageId === imageId) {
      setEditorImageId("");
    }
  };

  const handleSetCoverImage = (imageId: string) => {
    if (!albumDraft || !imageId) return;
    const coverPage = albumDraft.pages.find((page) => page.pageType === "cover") ?? albumDraft.pages[0];
    if (!coverPage) return;
    patchAlbumPage(coverPage.id, {
      imageIds: [imageId],
      layoutId: "cover_full_bleed",
    });
    setAlbumPageId(coverPage.id);
    setEditorImageId(imageId);
  };

  const handleSaveAlbumDraft = async () => {
    if (!selectedProject || !albumDraft) return;
    setBusy("album_save");
    setError("");
    try {
      const saved = await updateAlbum(selectedProject.id, {
        title: albumDraft.title,
        intro: albumDraft.intro,
        themeId: albumDraft.themeId,
        pages: normalizeDraftPages(albumDraft.pages),
        reason: "手动编辑相册",
      });
      patchAlbumInProject(saved);
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
      setAlbumDraft(cloneAlbumForDraft(updated.album ?? saved));
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存相册失败");
    } finally {
      setBusy("");
    }
  };

  const handleUndoAlbum = async () => {
    if (!selectedProject || !selectedAlbum) return;
    setBusy("album_undo");
    setError("");
    try {
      const album = await undoAlbum(selectedProject.id);
      patchAlbumInProject(album);
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
      setAlbumDraft(cloneAlbumForDraft(updated.album ?? album));
    } catch (err) {
      setError(err instanceof Error ? err.message : "没有可撤销的编辑");
    } finally {
      setBusy("");
    }
  };

  const handleRedoAlbum = async () => {
    if (!selectedProject || !selectedAlbum) return;
    setBusy("album_redo");
    setError("");
    try {
      const album = await redoAlbum(selectedProject.id);
      patchAlbumInProject(album);
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
      setAlbumDraft(cloneAlbumForDraft(updated.album ?? album));
    } catch (err) {
      setError(err instanceof Error ? err.message : "没有可重做的编辑");
    } finally {
      setBusy("");
    }
  };

  const handleGenerateImageEdit = async (imageId: string, prompt: string) => {
    if (!imageId) return;
    setBusy(`generate_edit:${imageId}`);
    setError("");
    try {
      const updated = await generateImageEdit(imageId, { prompt });
      patchImage(updated);
      setActiveImageId(updated.id);
      setEditorImageId(updated.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "生成式图片优化失败");
    } finally {
      setBusy("");
    }
  };

  const handleUndoImageEdit = async (imageId: string) => {
    if (!imageId) return;
    setBusy(`undo_edit:${imageId}`);
    setError("");
    try {
      const updated = await undoImageEdit(imageId);
      patchImage(updated);
      setActiveImageId(updated.id);
      setEditorImageId(updated.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "没有可撤销的图片优化");
    } finally {
      setBusy("");
    }
  };

  const handleExportType = async (type: string) => {
    if (!selectedProject) return;
    setBusy(`export:${type}`);
    setError("");

    let progressTimer: ReturnType<typeof setInterval> | null = null;
    if (type === "github_pages") {
      setGithubPublishProgress({
        projectId: selectedProject.id,
        active: true,
        phase: "preparing",
        current: 0,
        total: 0,
        message: "正在准备发布...",
      });
      progressTimer = setInterval(async () => {
        try {
          const p = await getGitHubPublishProgress(selectedProject.id);
          setGithubPublishProgress(p);
        } catch {
          // ignore transient polling errors
        }
      }, 700);
    }

    try {
      const result = await exportAlbumArtifact(selectedProject.id, type);
      setExportInfo(result);
      const updated = await getProject(selectedProject.id);
      patchProject(updated);
      setActiveView("export");
    } catch (err) {
      setError(err instanceof Error ? err.message : "导出失败");
    } finally {
      if (progressTimer) {
        clearInterval(progressTimer);
        // Do one final poll to pick up the terminal message
        try {
          const p = await getGitHubPublishProgress(selectedProject.id);
          setGithubPublishProgress(p);
        } catch {
          // ignore
        }
        // Keep the terminal progress visible briefly, then clear
        window.setTimeout(() => setGithubPublishProgress(null), 4000);
      }
      setBusy("");
    }
  };

  const handleSaveAISettings = async () => {
    setBusy("settings_ai");
    setError("");
    setSettingsNote("");
    try {
      const saved = await updateAISettings(aiSettings);
      setAISettings({
        ...emptyAISettings,
        ...saved,
        model: saved.model || emptyAISettings.model,
        imageBaseUrl: saved.imageBaseUrl ?? emptyAISettings.imageBaseUrl,
        imageApiKey: saved.imageApiKey ?? emptyAISettings.imageApiKey,
        imageModel: saved.imageModel || emptyAISettings.imageModel,
      });
      setSettingsNote("AI 配置已保存，后续分析和图像优化会使用新的模型配置。");
    } catch (err) {
      const message = err instanceof Error ? err.message : "AI 配置保存失败";
      setSettingsNote(message);
      setError(message);
    } finally {
      setBusy("");
    }
  };

  const handleSaveGitHubSettings = async () => {
    setBusy("settings_github");
    setError("");
    setGitHubSettingsNote("");
    try {
      const saved = await updateGitHubSettings(githubSettings);
      setGitHubSettings({
        ...emptyGitHubSettings,
        ...saved,
        branch: saved.branch || emptyGitHubSettings.branch,
      });
      setGitHubSettingsNote("GitHub 配置已保存，可从导出页面发布相册到 GitHub Pages。");
    } catch (err) {
      const message = err instanceof Error ? err.message : "GitHub 配置保存失败";
      setGitHubSettingsNote(message);
      setError(message);
    } finally {
      setBusy("");
    }
  };

  const handleRefreshListing = async () => {
    setBusy("refresh_listing");
    setError("");
    setGitHubSettingsNote("");
    try {
      await publishAlbumListing();
      setGitHubSettingsNote("相册首页已刷新");
    } catch (err) {
      const message = err instanceof Error ? err.message : "刷新首页失败";
      setGitHubSettingsNote(message);
      setError(message);
    } finally {
      setBusy("");
    }
  };

  const startProjectRename = () => {
    if (!selectedProject) return;
    setProjectTitleDraft(selectedProject.title);
    setProjectRenaming(true);
  };

  const cancelProjectRename = () => {
    setProjectTitleDraft(selectedProject?.title ?? "");
    setProjectRenaming(false);
  };

  const handleSaveProjectRename = async () => {
    if (!selectedProject) return;
    const title = projectTitleDraft.trim();
    if (!title) {
      setError("项目名称不能为空。");
      return;
    }
    if (title === selectedProject.title) {
      setProjectRenaming(false);
      return;
    }

    setBusy("project_rename");
    setError("");
    try {
      const updated = await updateProject(selectedProject.id, { title });
      patchProject(updated);
      setProjectTitleDraft(updated.title);
      setProjectRenaming(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "项目重命名失败");
    } finally {
      setBusy("");
    }
  };

  const performDeleteProject = async (projectId: string) => {
    setBusy("project_delete");
    setError("");
    try {
      await deleteProject(projectId);
      const remaining = await listProjects();
      setProjects(remaining);
      setSelectedProjectId((current) => (current === projectId ? remaining[0]?.id ?? "" : current));
      setProjectRenaming(false);
      if (remaining.length === 0) {
        setActiveView("setup");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "项目删除失败");
      throw err;
    } finally {
      setBusy("");
    }
  };

  const handleDeleteProject = () => {
    if (!selectedProject || selectedProject.status === "analyzing") return;

    openConfirmDialog({
      title: "删除这个项目？",
      message: `确定删除项目「${selectedProject.title}」吗？项目里的照片、相册和导出记录都会一起移除，删除后不可恢复。`,
      details: [
        `照片：${selectedProject.images.length} 张`,
        selectedProject.album ? "已生成相册会一并删除" : "当前还没有生成相册",
        selectedProject.activeTask ? `当前任务：${selectedProject.activeTask.type}` : "没有运行中的任务",
      ],
      confirmLabel: "删除项目",
      cancelLabel: "保留项目",
      pendingLabel: "删除中...",
      tone: "danger",
      onConfirm: () => performDeleteProject(selectedProject.id),
    });
  };

  const dismissOnboarding = () => {
    window.localStorage.setItem("memoir:onboarding-dismissed", "true");
    setShowOnboarding(false);
  };

  // --- Gallery Edit/Delete Handlers ---

  const openGalleryEditDialog = (project: Project) => {
    setEditingGalleryProject(project);
    setGalleryEditDraft({
      title: project.title,
      description: project.description ?? "",
      location: project.location ?? "",
      tone: project.tone ?? "",
      themeId: project.themeId ?? themeOptions[0]?.value ?? "",
    });
    setOpenMenuProjectId("");
  };

  const closeGalleryEditDialog = () => {
    if (busy === "gallery_edit") return;
    setEditingGalleryProject(null);
  };

  const saveGalleryEdit = async () => {
    if (!editingGalleryProject) return;
    const title = galleryEditDraft.title.trim();
    if (!title) return;
    setBusy("gallery_edit");
    setError("");
    try {
      await updateProject(editingGalleryProject.id, {
        title,
        description: galleryEditDraft.description.trim() || undefined,
        location: galleryEditDraft.location.trim() || undefined,
        tone: galleryEditDraft.tone.trim() || undefined,
        themeId: galleryEditDraft.themeId,
      });
      const updated = await listProjects();
      setProjects(updated);
      setEditingGalleryProject(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "项目更新失败");
    } finally {
      setBusy("");
    }
  };

  const handleGalleryDelete = (project: Project) => {
    setOpenMenuProjectId("");
    if (project.status === "analyzing") return;

    openConfirmDialog({
      title: "删除这个项目？",
      message: `确定删除项目「${project.title}」吗？项目里的照片、相册和导出记录都会一起移除，删除后不可恢复。`,
      details: [
        `照片：${project.images.length} 张`,
        project.album ? "已生成相册会一并删除" : "当前还没有生成相册",
        project.activeTask ? `当前任务：${project.activeTask.type}` : "没有运行中的任务",
      ],
      confirmLabel: "删除项目",
      cancelLabel: "保留项目",
      pendingLabel: "删除中...",
      tone: "danger",
      onConfirm: () => performDeleteProject(project.id),
    });
  };

  const statusLabel = (status: Project["status"]): string => {
    switch (status) {
      case "draft": return "草稿";
      case "uploading": return "上传中";
      case "analyzing": return "分析中";
      case "reviewing": return "审核中";
      case "generating_album": return "生成中";
      case "editing": return "编辑中";
      case "exporting": return "导出中";
      case "done": return "已完成";
      case "failed": return "失败";
      default: return status;
    }
  };

  const openDiscardAlbumEditsDialog = (onConfirm: () => void, targetLabel: string) => {
    if (!albumDirty || !selectedAlbum) {
      onConfirm();
      return;
    }

    openConfirmDialog({
      title: "放弃未保存的相册编辑？",
      message: `你当前对「${selectedAlbum.title}」有未保存的版式、配文或照片顺序调整。继续${targetLabel}会丢弃这些本地编辑。`,
      details: [
        `当前相册：${selectedAlbum.title}`,
        `页面：${selectedAlbum.pages.length} 页`,
        "已保存版本不会受影响",
      ],
      confirmLabel: `放弃并${targetLabel}`,
      cancelLabel: "继续编辑",
      pendingLabel: "处理中...",
      tone: "warning",
      onConfirm,
    });
  };

  const handleSelectProject = (projectId: string) => {
    if (projectId === selectedProjectId && workspaceMode === "project") return;
    openDiscardAlbumEditsDialog(() => {
      setSelectedProjectId(projectId);
      setProjectRenaming(false);
      setWorkspaceMode("project");
      setIsCreatingProject(false);
    }, "切换项目");
  };

  const handleNewProject = () => {
    openDiscardAlbumEditsDialog(() => {
      setSelectedProjectId("");
      setProjectRenaming(false);
      setProjectTitleDraft("");
      setActiveView("setup");
      setWorkspaceMode("project");
      setIsCreatingProject(true);
    }, "创建新项目");
  };

  const handleOpenMemoryMap = () => {
    openDiscardAlbumEditsDialog(() => {
      setWorkspaceMode("memory_map");
      setProjectRenaming(false);
    }, "打开集忆地图");
  };

  const renderCreateForm = () => (
    <ProjectCreateForm
      draft={draft}
      themeOptions={themeOptions}
      busy={busy === "create"}
      onDraftChange={setDraft}
      onCreate={handleCreateProject}
    />
  );

  const renderUploadTrigger = (className = "btn-primary") => (
    <label className={`${className} file-trigger${isUploading ? " is-busy" : ""}`}>
      <Upload size={16} style={{ marginRight: 6 }} />
      导入照片
      <input
        type="file"
        accept={uploadAccept}
        multiple
        onChange={handleUploadInputChange}
        disabled={isUploading}
        style={{ display: "none" }}
      />
    </label>
  );

  const renderWorkflowRail = () => {
    if (!selectedProject) return null;

    return (
      <WorkflowRail stages={workflowStages} onSelectStage={(stage) => setActiveView(stage as ViewId)} />
    );
  };

  const renderSetupView = () => {
    if (!selectedProject) {
      return (
        <div className="setup-view">
          {showOnboarding ? (
            <section className="first-run-note onboarding-panel">
              <div className="onboarding-copy">
                <span className="pill" data-tone="good">
                  首次使用
                </span>
                <strong>先创建一个工作区；导入后页面会进入“准备照片”，上传和处理状态会独立显示。</strong>
              </div>
              <button
                type="button"
                className="icon-button"
                onClick={dismissOnboarding}
                aria-label="关闭首次使用提示"
              >
                <X size={16} />
              </button>
            </section>
          ) : null}
          {renderCreateForm()}
        </div>
      );
    }

    const recentImages = selectedImages.slice(0, 8);
    const canAnalyze = pendingAnalysisCount > 0 && selectedProject.status !== "analyzing";

    return (
      <ProjectSetupPanel
        project={selectedProject}
        themeOptions={themeOptions}
        busy={busy}
        uploadAccept={uploadAccept}
        isUploading={isUploading}
        uploadState={uploadState}
        uploadHint={nextAction.hint}
        analysisStatusLabel={analysisStatusLabel(selectedProject.analysisStatus)}
        analysisProgress={selectedProject.analysisProgress}
        staleAnalysisCount={staleAnalysisCount}
        stats={stats}
        pendingAnalysisCount={pendingAnalysisCount}
        canAnalyze={canAnalyze}
        recentImages={recentImages}
        allImages={selectedImages}
        isDeletingImage={isDeletingImage}
        onDrop={handleDrop}
        onUploadInputChange={handleUploadInputChange}
        onThemeChange={handleThemeChange}
        onAnalyze={handleAnalyze}
        onGoReview={() => setActiveView("review")}
        onPreviewImage={(imageId) => setPreviewImageId(imageId)}
        onDeleteImage={handleDeleteImage}
      />
    );
  };

  const renderReviewView = () => {
    if (!selectedProject) {
      return <section className="panel empty-state">先创建或选择一个项目。</section>;
    }
    return (
      <ReviewView
        selectedProject={selectedProject}
        selectedAlbum={selectedAlbum}
        selectedImages={selectedImages}
        busy={busy}
        visibleImages={visibleImages}
        visibleImageIds={visibleImageIds}
        reviewGroups={reviewGroups}
        activeImage={activeImage}
        activeImageIndex={activeImageIndex}
        selectedImageIds={selectedImageIds}
        formatCount={formatCount}
        imageDisplayUrl={imageDisplayUrl}
        onSetActiveImageId={setActiveImageId}
        onGoToAdjacentImage={goToAdjacentImage}
        onDecision={handleDecision}
        onGenerate={handleGenerate}
        onAnalyze={handleAnalyze}
        onGenerateImageEdit={handleGenerateImageEdit}
        onUndoImageEdit={handleUndoImageEdit}
      />
    );
  };

  const renderAlbumView = () => {
    return (
      <AlbumEditorView
        selectedProject={selectedProject}
        selectedAlbum={selectedAlbum}
        albumDraft={albumDraft}
        selectedImages={selectedImages}
        busy={busy}
        albumDirty={albumDirty}
        albumPageId={albumPageId}
        editorImageId={editorImageId}
        dragPageId={dragPageId}
        dragImageId={dragImageId}
        themeOptions={themeOptions}
        layoutOptions={layoutOptions}
        onSetAlbumPageId={setAlbumPageId}
        onSetEditorImageId={setEditorImageId}
        onSetDragPageId={setDragPageId}
        onSetDragImageId={setDragImageId}
        onAlbumTitleChange={(title) =>
          setAlbumDraft((current) => (current ? { ...current, title } : current))
        }
        onAlbumIntroChange={(intro) =>
          setAlbumDraft((current) => (current ? { ...current, intro } : current))
        }
        onAlbumThemeChange={(themeId) =>
          setAlbumDraft((current) => (current ? { ...current, themeId } : current))
        }
        onMoveAlbumPage={moveAlbumPage}
        onMoveImageWithinPage={moveImageWithinPage}
        onPatchAlbumPage={patchAlbumPage}
        onDeleteAlbumPage={handleDeleteAlbumPage}
        onAddImageToPage={handleAddImageToPage}
        onRemoveImageFromPage={handleRemoveImageFromPage}
        onSetCoverImage={handleSetCoverImage}
        onSaveAlbumDraft={handleSaveAlbumDraft}
        onUndoAlbum={handleUndoAlbum}
        onRedoAlbum={handleRedoAlbum}
        onGenerate={handleGenerate}
        onGoToReview={() => setActiveView("review")}
        onGoToExport={() => setActiveView("export")}
      />
    );
  };

  const renderExportView = () => {
    const recentExports = selectedProject?.exports ?? [];
    return (
      <ExportView
        selectedProject={selectedProject}
        selectedAlbum={selectedAlbum}
        selectedImages={selectedImages}
        activeExport={activeExport}
        recentExports={recentExports}
        busy={busy}
        exportOptions={exportOptions}
        githubPublishProgress={githubPublishProgress}
        onExportType={handleExportType}
        onGoToAlbum={() => setActiveView("album")}
      />
    );
  };

  const renderSettingsView = () => {
    return (
      <SettingsView
        selectedProject={selectedProject}
        aiSettings={aiSettings}
        githubSettings={githubSettings}
        busy={busy}
        settingsNote={settingsNote}
        githubSettingsNote={githubSettingsNote}
        onAISettingsChange={setAISettings}
        onGitHubSettingsChange={setGitHubSettings}
        onSaveAISettings={handleSaveAISettings}
        onSaveGitHubSettings={handleSaveGitHubSettings}
        onRefreshListing={handleRefreshListing}
        onBack={() => setActiveView(defaultViewForProject(selectedProject))}
      />
    );
  };

  const renderGuideView = () => {
    return (
      <GuideView
        onBack={() => setActiveView(defaultViewForProject(selectedProject))}
        onOpenSettings={() => setActiveView("settings")}
      />
    );
  };

  const handleSaveProjectPlace = async (
    projectId: string,
    input: {
      location?: string;
      place?: ProjectPlace;
      clearPlace?: boolean;
    },
  ) => {
    setBusy(`place:${projectId}`);
    setError("");
    try {
      const updated = await updateProject(projectId, input);
      patchProject(updated);
    } catch (err) {
      const message = err instanceof Error ? err.message : "地点保存失败";
      setError(message);
      throw err;
    } finally {
      setBusy("");
    }
  };

  const renderActiveView = () => {
    switch (activeView) {
      case "review":
        return renderReviewView();
      case "album":
        return renderAlbumView();
      case "export":
        return renderExportView();
      case "settings":
        return renderSettingsView();
      case "guide":
        return renderGuideView();
      default:
        return renderSetupView();
    }
  };

  return (
    <div className="app-shell" data-mode={
      workspaceMode === "memory_map" ? "memory_map" :
      selectedProject || isCreatingProject ? "workspace" : "gallery"
    }>
      {/* Memory Map - full page overlay */}
      {workspaceMode === "memory_map" && (
        <MemoryMapView
          projects={projects}
          busy={busy}
          onSelectProject={handleSelectProject}
          onSaveProjectPlace={handleSaveProjectPlace}
          onBack={() => setWorkspaceMode("project")}
        />
      )}

      {/* Project Gallery Mode */}
      {!selectedProject && !isCreatingProject && workspaceMode !== "memory_map" && (
        <div className="project-gallery">
          {/* Top-right toolbar */}
          <div className="gallery-toolbar">
            <button
              type="button"
              className="gallery-icon-btn"
              onClick={() => setActiveView('guide')}
              aria-label="使用指南"
              title="GitHub Pages 使用指南"
            >
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10"/>
                <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
                <line x1="12" y1="17" x2="12.01" y2="17"/>
              </svg>
            </button>
            <button
              type="button"
              className="gallery-icon-btn"
              onClick={() => setActiveView('settings')}
              aria-label="设置"
              title="设置"
            >
              <Settings size={18} />
            </button>
          </div>
          {activeView === "settings" || activeView === "guide" ? (
            <section className="view-shell">{renderActiveView()}</section>
          ) : (
            <>
          <div className="gallery-header">
            <h1 className="gallery-title">Memoir</h1>
            <div className="gallery-divider">
              <span className="gallery-divider-dot" />
            </div>
            <p className="gallery-subtitle">你的照片故事集</p>
            <div className="gallery-actions">
              <button
                type="button"
                className="gallery-btn gallery-btn--outline"
                onClick={handleOpenMemoryMap}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polygon points="1 6 1 22 8 18 16 22 23 18 23 2 16 6 8 2 1 6"/>
                  <line x1="8" y1="2" x2="8" y2="18"/>
                  <line x1="16" y1="6" x2="16" y2="22"/>
                </svg>
                <span>集忆地图</span>
              </button>
              <button
                type="button"
                className="gallery-btn gallery-btn--primary"
                onClick={handleNewProject}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>新建项目</span>
              </button>
            </div>
            {projects.length > 0 && (
              <div className="gallery-stats">
                <span className="stat">
                  <span className="stat-value">{projects.length}</span> 个项目
                </span>
                <span className="stat-divider">·</span>
                <span className="stat">
                  <span className="stat-value">{projects.reduce((sum, p) => sum + p.images.length, 0)}</span> 张照片
                </span>
                <span className="stat-divider">·</span>
                <span className="stat">
                  <span className="stat-value">{projects.filter((p) => p.album).length}</span> 本相册
                </span>
              </div>
            )}
          </div>
          {projects.length === 0 ? (
            <div className="gallery-empty">
              <div className="gallery-empty-icon">
                <BookOpen size={36} strokeWidth={1.5} />
              </div>
              <h2>还没有项目</h2>
              <p>创建你的第一个相册项目，开始整理和讲述你的照片故事。</p>
              <button
                type="button"
                className="gallery-btn gallery-btn--primary"
                onClick={handleNewProject}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                <span>创建第一个项目</span>
              </button>
            </div>
          ) : (
            <div className="gallery-grid">
              {projects.map((project) => {
                const coverImage = project.images[0];
                const isMenuOpen = openMenuProjectId === project.id;
                const createdAt = project.createdAt
                  ? new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "short", day: "numeric" }).format(new Date(project.createdAt))
                  : "";
                return (
                  <div
                    key={project.id}
                    className="project-card"
                    onClick={() => handleSelectProject(project.id)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        handleSelectProject(project.id);
                      }
                    }}
                    role="button"
                    tabIndex={0}
                  >
                    <div className="project-card-image">
                      {coverImage && (
                        <img src={coverImage.thumbnailUrl} alt="" />
                      )}
                      <div className="project-card-status" data-status={project.status}>
                        <span className="project-card-status-dot" />
                        <span>{statusLabel(project.status)}</span>
                      </div>
                      <button
                        type="button"
                        className="project-card-menu-btn"
                        data-open={isMenuOpen}
                        onClick={(e) => {
                          e.stopPropagation();
                          setOpenMenuProjectId(isMenuOpen ? "" : project.id);
                        }}
                        aria-label="项目操作"
                      >
                        <MoreHorizontal size={16} />
                      </button>
                      {isMenuOpen && (
                        <div
                          className="project-card-menu"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            type="button"
                            onClick={() => openGalleryEditDialog(project)}
                          >
                            <Pencil size={14} />
                            <span>编辑</span>
                          </button>
                          <button
                            type="button"
                            className="menu-danger"
                            onClick={() => handleGalleryDelete(project)}
                            disabled={project.status === "analyzing"}
                          >
                            <Trash2 size={14} />
                            <span>删除</span>
                          </button>
                        </div>
                      )}
                      <div className="project-card-overlay">
                        <span className="project-card-overlay-text">打开</span>
                      </div>
                    </div>
                    <div className="project-card-content">
                      <h3 className="project-card-title">{project.title}</h3>
                      {project.description && (
                        <p className="project-card-description">{project.description}</p>
                      )}
                      <p className="project-card-meta">
                        {project.images.length} 张照片
                        {project.album && ` · ${project.album.pages.length} 页`}
                        {createdAt && ` · ${createdAt}`}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
            </>
          )}
        </div>
      )}

      {/* Workspace Mode */}
      {(selectedProject || isCreatingProject) && workspaceMode !== "memory_map" && (
        <>
          {/* Floating Navigation */}
          <nav className="floating-nav">
            <button
              type="button"
              className="nav-back"
              onClick={() => {
                setSelectedProjectId("");
                setIsCreatingProject(false);
              }}
            >
              ← 返回
            </button>
            <div className="nav-project-name">{selectedProject?.title || "新建项目"}</div>
            {selectedProject && (
            <div className="nav-tabs">
              <button
                type="button"
                className={`nav-tab ${activeView === 'setup' ? 'active' : ''}`}
                onClick={() => setActiveView('setup')}
              >
                导入
              </button>
              <button
                type="button"
                className={`nav-tab ${activeView === 'review' ? 'active' : ''}`}
                onClick={() => setActiveView('review')}
                disabled={selectedProject.images.length === 0}
              >
                审核
              </button>
              <button
                type="button"
                className={`nav-tab ${activeView === 'album' ? 'active' : ''}`}
                onClick={() => setActiveView('album')}
                disabled={!selectedProject.album}
              >
                相册
              </button>
              <button
                type="button"
                className={`nav-tab ${activeView === 'export' ? 'active' : ''}`}
                onClick={() => setActiveView('export')}
                disabled={!selectedProject.album}
              >
                导出
              </button>
            </div>
            )}
            <button
              type="button"
              className="nav-settings"
              onClick={() => setActiveView('settings')}
            >
              ⚙
            </button>
          </nav>

          {/* Scene Container */}
          <main className="scene-container">
            {error ? (
              <section className="error-banner">
                <span>{error}</span>
                <button type="button" className="icon-button" onClick={() => setError("")}>
                  <X size={16} />
                </button>
              </section>
            ) : null}

            <section className="view-shell">{renderActiveView()}</section>
          </main>
        </>
      )}

      {previewImage ? (
        <DialogShell
          open={Boolean(previewImage)}
          onClose={() => {
            setPreviewImageId("");
            setPreviewOverride(null);
          }}
          rootClassName="image-modal"
          backdropClassName="image-modal-backdrop"
          panelClassName="image-modal-content panel"
          ariaLabel={previewOverride?.title || previewImage.fileName}
          zIndex={60}
        >
          <div className="image-modal-head">
            <div>
              <div className="panel-title">{previewOverride?.title || previewImage.fileName}</div>
              <div className="panel-subtitle">
                {previewOverride?.subtitle ||
                  `${previewImage.width} x ${previewImage.height} · ${imageStatusLabel(previewImage.status)}`}
              </div>
            </div>
            <div className="btn-row">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => handleDeleteImage(previewImage.id)}
                disabled={isDeletingImage || selectedProject?.status === "analyzing"}
              >
                <Trash2 size={16} style={{ marginRight: 6 }} />
                删除
              </button>
              <button
                type="button"
                className="icon-button"
                onClick={() => {
                  setPreviewImageId("");
                  setPreviewOverride(null);
                }}
                aria-label="关闭预览"
              >
                <X size={16} />
              </button>
            </div>
          </div>
          <div className="image-modal-frame">
            <img src={previewOverride?.url || imageDisplayUrl(previewImage)} alt={previewOverride?.title || previewImage.fileName} />
          </div>
        </DialogShell>
      ) : null}

      {/* Gallery Edit Project Dialog */}
      <DialogShell
        open={Boolean(editingGalleryProject)}
        onClose={closeGalleryEditDialog}
        rootClassName="project-edit-dialog"
        backdropClassName="image-modal-backdrop"
        panelClassName="panel"
        ariaLabel="编辑项目信息"
        zIndex={60}
      >
        <div className="dialog-header">
          <div>
            <h3>编辑项目信息</h3>
            <p>{editingGalleryProject?.title}</p>
          </div>
          <button type="button" className="dialog-close" onClick={closeGalleryEditDialog} aria-label="关闭">
            <X size={20} />
          </button>
        </div>

        <div className="dialog-body">
          <label className="dialog-field">
            <span>项目名称</span>
            <input
              type="text"
              value={galleryEditDraft.title}
              onChange={(e) => setGalleryEditDraft((d) => ({ ...d, title: e.target.value }))}
              placeholder="例如：2024年春节家庭聚会"
              autoFocus
            />
          </label>

          <label className="dialog-field">
            <span>项目描述</span>
            <textarea
              rows={3}
              value={galleryEditDraft.description}
              onChange={(e) => setGalleryEditDraft((d) => ({ ...d, description: e.target.value }))}
              placeholder="记录这个相册的背景和故事..."
            />
          </label>

          <div className="dialog-row">
            <label className="dialog-field">
              <span>地点</span>
              <input
                type="text"
                value={galleryEditDraft.location}
                onChange={(e) => setGalleryEditDraft((d) => ({ ...d, location: e.target.value }))}
                placeholder="例如：北京"
              />
            </label>

            <label className="dialog-field">
              <span>主题风格</span>
              <select
                value={galleryEditDraft.themeId}
                onChange={(e) => setGalleryEditDraft((d) => ({ ...d, themeId: e.target.value }))}
              >
                {themeOptions.map((theme) => (
                  <option key={theme.value} value={theme.value}>
                    {theme.label}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>

        <div className="dialog-footer">
          <button
            type="button"
            className="btn-secondary"
            onClick={closeGalleryEditDialog}
            disabled={busy === "gallery_edit"}
          >
            取消
          </button>
          <button
            type="button"
            className="btn-primary"
            onClick={saveGalleryEdit}
            disabled={busy === "gallery_edit" || !galleryEditDraft.title.trim()}
          >
            {busy === "gallery_edit" ? "保存中..." : "保存"}
          </button>
        </div>
      </DialogShell>

      <ConfirmDialog
        open={confirmDialog.open}
        title={confirmDialog.title}
        message={confirmDialog.message}
        details={confirmDialog.details}
        confirmLabel={confirmDialog.confirmLabel}
        cancelLabel={confirmDialog.cancelLabel}
        pendingLabel={confirmDialog.pendingLabel}
        tone={confirmDialog.tone}
        onCancel={closeConfirmDialog}
        onConfirm={handleConfirmDialogConfirm}
      />
    </div>
  );
}
