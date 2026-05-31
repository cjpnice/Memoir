export type ProjectStatus =
  | "draft"
  | "uploading"
  | "analyzing"
  | "reviewing"
  | "generating_album"
  | "editing"
  | "exporting"
  | "done"
  | "failed";

export type ImageStatus =
  | "uploaded"
  | "analyzing"
  | "analyzed"
  | "keep"
  | "improve_then_keep"
  | "review"
  | "reject_suggested"
  | "approved"
  | "excluded";

export interface CropBox {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface CropSuggestion {
  id: string;
  aspectRatio: string;
  box: CropBox;
  reason: string;
}

export interface ImageMetrics {
  aspectRatio: number;
  brightness: number;
  contrast: number;
  sharpness: number;
  averageHash: string;
  fileSize: number;
  width: number;
  height: number;
}

export interface DetectedContent {
  peopleCount: number;
  scenes: string[];
  objects: string[];
  mood: string[];
  tags: string[];
}

export interface ImageIssue {
  type: string;
  severity: "low" | "medium" | "high" | string;
  description: string;
}

export interface ImageEditSuggestion {
  type: string;
  strength?: string;
  reason: string;
  execution?: "local" | "local_approximation" | "provider_generative" | string;
  providerBacked?: boolean;
  actionLabel?: string;
}

export interface ImageAnalysis {
  qualityScore: number;
  preservationScore: number;
  storyScore: number;
  recommendation: string;
  reasons: string[];
  detectedContent: DetectedContent;
  metrics: ImageMetrics;
  issues: ImageIssue[];
  cropSuggestions: CropSuggestion[];
  captionSeeds: string[];
  editSuggestions?: ImageEditSuggestion[];
  similarGroupId?: string;
  similarGroupLabel?: string;
  similarGroupRank?: number;
  similarGroupBest?: boolean;
  similarGroupReason?: string;
  albumRole?: string;
  socialCaption?: string;
  selectionRank?: number;
  modelVersion?: string;
  promptVersion?: string;
  completedAt?: string;
}

export interface AlbumPage {
  id: string;
  order: number;
  pageType: string;
  layoutId: string;
  imageIds: string[];
  title: string;
  body: string;
  caption: string;
}

export interface AlbumSocialPost {
  platform?: string;
  title: string;
  body: string;
  hook?: string;
  imageIds: string[];
  hashtags?: string[];
}

export interface AlbumSnapshot {
  title: string;
  intro: string;
  themeId: string;
  pages: AlbumPage[];
  version: number;
  reason: string;
  createdAt: string;
}

export interface Album {
  id: string;
  projectId: string;
  themeId: string;
  title: string;
  intro: string;
  designNotes?: string;
  status: string;
  version: number;
  modelVersion?: string;
  promptVersion?: string;
  pages: AlbumPage[];
  socialPosts?: AlbumSocialPost[];
  editHistory?: AlbumSnapshot[];
  redoStack?: AlbumSnapshot[];
  createdAt: string;
  updatedAt: string;
}

export interface AlbumExport {
  id: string;
  albumId: string;
  projectId: string;
  type: string;
  url: string;
  message?: string;
  createdAt: string;
}

export interface ImageAsset {
  id: string;
  projectId: string;
  fileName: string;
  mimeType: string;
  fileSize: number;
  width: number;
  height: number;
  originalUrl: string;
  thumbnailUrl: string;
  derivedUrl?: string;
  averageHash: string;
  status: ImageStatus;
  userDecision?: string;
  editHistory?: ImageSnapshot[];
  analysis?: ImageAnalysis;
  createdAt: string;
  updatedAt: string;
}

export interface ImageSnapshot {
  derivedUrl?: string;
  width: number;
  height: number;
  status: ImageStatus;
  userDecision?: string;
  reason?: string;
  createdAt: string;
}

export type ProjectPlaceSource = "city_catalog" | "manual";

export interface ProjectPlace {
  city: string;
  region?: string;
  country: string;
  latitude: number;
  longitude: number;
  source: ProjectPlaceSource;
  confidence?: number;
}

export interface Project {
  id: string;
  title: string;
  description: string;
  location?: string;
  place?: ProjectPlace;
  tone: string;
  themeId: string;
  status: ProjectStatus;
  analysisStatus: string;
  analysisProgress: number;
  analysisModelVersion?: string;
  analysisPromptVersion?: string;
  currentAnalysisModelVersion?: string;
  currentAnalysisPromptVersion?: string;
  pendingAnalysisCount?: number;
  staleAnalysisCount?: number;
  currentStep: string;
  lastError?: string;
  createdAt: string;
  updatedAt: string;
  images: ImageAsset[];
  album?: Album;
  activeTask?: ProjectTask;
  taskHistory?: ProjectTask[];
  exports?: AlbumExport[];
}

export interface ProjectTask {
  id: string;
  type: "analysis" | "album_generation" | "export" | string;
  status: "running" | "completed" | "failed" | "interrupted" | string;
  progress: number;
  message: string;
  error?: string;
  startedAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface AISettings {
  baseUrl: string;
  apiKey: string;
  model: string;
  imageBaseUrl: string;
  imageApiKey: string;
  imageModel: string;
}

export interface GitHubSettings {
  owner: string;
  repo: string;
  branch: string;
  token: string;
}

export interface GitHubPublishProgress {
  projectId: string;
  active: boolean;
  phase: "preparing" | "uploading_images" | "uploading_html" | "updating_listing" | "done" | "error" | "";
  current: number;
  total: number;
  message: string;
  error?: string;
}
