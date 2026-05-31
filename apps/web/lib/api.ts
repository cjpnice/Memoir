import type { AISettings, Album, AlbumExport, AlbumPage, GitHubPublishProgress, GitHubSettings, ImageAsset, Project, ProjectPlace } from "@/lib/types";

const DEFAULT_API_BASE_URL = "http://127.0.0.1:8090";
const configuredApiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
const API_BASE_URL = configuredApiBaseUrl === "" ? "" : configuredApiBaseUrl || DEFAULT_API_BASE_URL;

export type UploadProgress = {
  loaded: number;
  total: number;
  percent: number;
};

type UploadOptions = {
  onProgress?: (progress: UploadProgress) => void;
};

function resolveApiUrl(path: string) {
  if (/^https?:\/\//.test(path)) {
    return path;
  }
  if (!API_BASE_URL) {
    return path;
  }
  const base = API_BASE_URL.replace(/\/+$/, "");
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${base}${normalizedPath}`;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(resolveApiUrl(path), {
    ...init,
    headers: {
      ...(init?.headers ?? {}),
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
    },
    cache: "no-store",
  });

  if (!response.ok) {
    const payload = await response.json().catch(() => ({}));
    throw new Error(payload.error || `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return normalizeApiPayload(await response.json()) as T;
}

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string");
}

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

function fallbackAlbumPageTitle(pageType: string, layoutId: string, order: number) {
  switch (normalizePageType(pageType, layoutId)) {
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

function normalizeImage(image: ImageAsset): ImageAsset {
  if (!image.analysis) {
    return {
      ...image,
      editHistory: Array.isArray(image.editHistory) ? image.editHistory : [],
    };
  }
  const analysis = image.analysis;
  return {
    ...image,
    editHistory: Array.isArray(image.editHistory) ? image.editHistory : [],
    analysis: {
      ...analysis,
      reasons: asStringArray(analysis.reasons),
      issues: Array.isArray(analysis.issues) ? analysis.issues : [],
      cropSuggestions: Array.isArray(analysis.cropSuggestions) ? analysis.cropSuggestions : [],
      captionSeeds: asStringArray(analysis.captionSeeds),
      editSuggestions: Array.isArray(analysis.editSuggestions) ? analysis.editSuggestions : [],
      detectedContent: {
        peopleCount: analysis.detectedContent?.peopleCount ?? 0,
        scenes: asStringArray(analysis.detectedContent?.scenes),
        objects: asStringArray(analysis.detectedContent?.objects),
        mood: asStringArray(analysis.detectedContent?.mood),
        tags: asStringArray(analysis.detectedContent?.tags),
      },
    },
  };
}

function normalizeAlbumPage(page: AlbumPage): AlbumPage {
  const pageType = normalizePageType(page.pageType, page.layoutId);
  return {
    ...page,
    pageType,
    title: normalizeAlbumPageTitle(page.title ?? "", pageType, page.layoutId, page.order),
    imageIds: asStringArray(page.imageIds),
  };
}

function normalizeAlbum(album: Album): Album {
  return {
    ...album,
    pages: Array.isArray(album.pages) ? album.pages.map(normalizeAlbumPage) : [],
    socialPosts: Array.isArray(album.socialPosts)
      ? album.socialPosts.map((post) => ({
          ...post,
          platform: typeof post.platform === "string" ? post.platform : "",
          hook: typeof post.hook === "string" ? post.hook : "",
          imageIds: asStringArray(post.imageIds),
          hashtags: asStringArray(post.hashtags),
        }))
      : [],
    editHistory: Array.isArray(album.editHistory)
      ? album.editHistory.map((snapshot) => ({
          ...snapshot,
          pages: Array.isArray(snapshot.pages) ? snapshot.pages.map(normalizeAlbumPage) : [],
        }))
      : [],
    redoStack: Array.isArray(album.redoStack)
      ? album.redoStack.map((snapshot) => ({
          ...snapshot,
          pages: Array.isArray(snapshot.pages) ? snapshot.pages.map(normalizeAlbumPage) : [],
        }))
      : [],
  };
}

function normalizeProject(project: Project): Project {
  return {
    ...project,
    images: Array.isArray(project.images) ? project.images.map(normalizeImage) : [],
    album: project.album ? normalizeAlbum(project.album) : undefined,
    taskHistory: Array.isArray(project.taskHistory) ? project.taskHistory : [],
    exports: Array.isArray(project.exports) ? project.exports : [],
  };
}

function normalizeApiPayload(payload: unknown): unknown {
  if (Array.isArray(payload)) {
    return payload.map((item) => normalizeApiPayload(item));
  }
  if (!payload || typeof payload !== "object") {
    return payload;
  }
  const record = payload as Record<string, unknown>;
  if (Array.isArray(record.images) && typeof record.id === "string" && typeof record.title === "string") {
    return normalizeProject(payload as Project);
  }
  if (Array.isArray(record.pages) && typeof record.projectId === "string" && typeof record.title === "string") {
    return normalizeAlbum(payload as Album);
  }
  if (typeof record.fileName === "string" && typeof record.projectId === "string") {
    return normalizeImage(payload as ImageAsset);
  }
  return payload;
}

export async function listProjects(): Promise<Project[]> {
  return request<Project[]>("/api/v1/projects");
}

export async function createProject(input: {
  title: string;
  description: string;
  location: string;
  place?: ProjectPlace;
  tone: string;
  themeId: string;
}): Promise<Project> {
  return request<Project>("/api/v1/projects", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function getProject(projectId: string): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}`);
}

export async function updateProject(
  projectId: string,
  input: {
    title?: string;
    description?: string;
    location?: string;
    place?: ProjectPlace;
    clearPlace?: boolean;
    tone?: string;
    themeId?: string;
  },
): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function deleteProject(projectId: string): Promise<void> {
  await request<void>(`/api/v1/projects/${projectId}`, {
    method: "DELETE",
  });
}

export async function deleteImage(imageId: string): Promise<void> {
  await request<void>(`/api/v1/images/${imageId}`, {
    method: "DELETE",
  });
}

export async function uploadImages(projectId: string, files: FileList | File[]): Promise<ImageAsset[]> {
  const formData = new FormData();
  Array.from(files).forEach((file) => formData.append("files", file));
  return uploadForm<ImageAsset[]>(`/api/v1/projects/${projectId}/images`, formData);
}

export async function uploadImagesWithProgress(
  projectId: string,
  files: FileList | File[],
  options: UploadOptions = {},
): Promise<ImageAsset[]> {
  const formData = new FormData();
  Array.from(files).forEach((file) => formData.append("files", file));
  return uploadForm<ImageAsset[]>(`/api/v1/projects/${projectId}/images`, formData, options);
}

function uploadForm<T>(
  path: string,
  formData: FormData,
  options: UploadOptions = {},
): Promise<T> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", resolveApiUrl(path));
    xhr.responseType = "text";

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) {
        options.onProgress?.({ loaded: event.loaded, total: 0, percent: 0 });
        return;
      }
      const percent = Math.round((event.loaded / event.total) * 100);
      options.onProgress?.({ loaded: event.loaded, total: event.total, percent });
    };

    xhr.onerror = () => reject(new Error("上传连接失败，请确认 Go 后端已启动。"));
    xhr.ontimeout = () => reject(new Error("上传超时，请减少单次导入数量后重试。"));
    xhr.onload = () => {
      if (xhr.status < 200 || xhr.status >= 300) {
        reject(new Error(readUploadError(xhr)));
        return;
      }
      try {
        resolve(JSON.parse(xhr.responseText) as T);
      } catch {
        reject(new Error("上传完成，但服务器返回数据无法解析。"));
      }
    };

    xhr.send(formData);
  });
}

function readUploadError(xhr: XMLHttpRequest) {
  const raw = xhr.responseText || "";
  try {
    const payload = JSON.parse(raw) as { error?: string };
    if (payload.error) return payload.error;
  } catch {
    // Fall through to plain-text handling below.
  }

  if (raw.includes("request body exceeded") || raw.includes("middlewareClientMaxBodySize")) {
    return "上传请求被 Next.js 代理限制了大小，请让前端直连 Go 后端或减少单次导入数量。";
  }
  return raw.trim() || `Upload failed: ${xhr.status}`;
}

export async function startAnalysis(projectId: string): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}/analyze`, {
    method: "POST",
  });
}

export async function updateImageDecision(
  imageId: string,
  decision: string,
): Promise<ImageAsset> {
  return request<ImageAsset>(`/api/v1/images/${imageId}/decision`, {
    method: "PATCH",
    body: JSON.stringify({ decision }),
  });
}

export async function applyCrop(imageId: string, cropId: string): Promise<ImageAsset> {
  return request<ImageAsset>(`/api/v1/images/${imageId}/crop`, {
    method: "POST",
    body: JSON.stringify({ cropId }),
  });
}

export async function processImage(
  imageId: string,
  input: {
    operation: string;
    angle?: number;
    brightness?: number;
    contrast?: number;
    saturation?: number;
    warmth?: number;
    expandRatio?: number;
    prompt?: string;
  },
): Promise<ImageAsset> {
  return request<ImageAsset>(`/api/v1/images/${imageId}/process`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function generateImageEdit(
  imageId: string,
  input: {
    prompt?: string;
  },
): Promise<ImageAsset> {
  return request<ImageAsset>(`/api/v1/images/${imageId}/generate-edit`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function undoImageEdit(imageId: string): Promise<ImageAsset> {
  return request<ImageAsset>(`/api/v1/images/${imageId}/undo-edit`, {
    method: "POST",
  });
}

export async function generateAlbum(projectId: string): Promise<Project> {
  return request<Project>(`/api/v1/projects/${projectId}/albums/generate`, {
    method: "POST",
  });
}

export async function updateAlbum(
  projectId: string,
  input: {
    title?: string;
    intro?: string;
    themeId?: string;
    pages?: AlbumPage[];
    reason?: string;
  },
): Promise<Album> {
  return request<Album>(`/api/v1/projects/${projectId}/albums`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function undoAlbum(projectId: string): Promise<Album> {
  return request<Album>(`/api/v1/projects/${projectId}/albums/undo`, {
    method: "POST",
  });
}

export async function redoAlbum(projectId: string): Promise<Album> {
  return request<Album>(`/api/v1/projects/${projectId}/albums/redo`, {
    method: "POST",
  });
}

export async function exportAlbum(projectId: string): Promise<AlbumExport> {
  return request<AlbumExport>(`/api/v1/projects/${projectId}/albums/export`, {
    method: "POST",
  });
}

export async function exportAlbumArtifact(projectId: string, type: string): Promise<AlbumExport> {
  return request<AlbumExport>(`/api/v1/projects/${projectId}/exports`, {
    method: "POST",
    body: JSON.stringify({ type }),
  });
}

export async function getAISettings(): Promise<AISettings> {
  return request<AISettings>("/api/v1/settings/ai");
}

export async function updateAISettings(input: AISettings): Promise<AISettings> {
  return request<AISettings>("/api/v1/settings/ai", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function getGitHubSettings(): Promise<GitHubSettings> {
  return request<GitHubSettings>("/api/v1/settings/github");
}

export async function updateGitHubSettings(input: GitHubSettings): Promise<GitHubSettings> {
  return request<GitHubSettings>("/api/v1/settings/github", {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function publishAlbumListing(): Promise<{ status: string }> {
  return request<{ status: string }>("/api/v1/settings/github/publish-listing", {
    method: "POST",
  });
}

export async function getGitHubPublishProgress(projectId: string): Promise<GitHubPublishProgress> {
  return request<GitHubPublishProgress>(`/api/v1/projects/${projectId}/exports/github-progress`);
}
