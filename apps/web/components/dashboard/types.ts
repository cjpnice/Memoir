import type { ReactNode, Dispatch, SetStateAction } from "react";

export type ProjectDraft = {
  title: string;
  description: string;
  location: string;
  tone: string;
  themeId: string;
};

export type UploadPhase = "idle" | "uploading" | "processing" | "done" | "failed";

export type UploadState = {
  phase: UploadPhase;
  percent: number;
  loadedBytes: number;
  totalBytes: number;
  fileCount: number;
  message: string;
  fileNames: string[];
};

export type WorkflowStage = {
  value: string;
  label: string;
  state: string;
  description: string;
  icon: ReactNode;
};

export type ProjectDraftSetter = Dispatch<SetStateAction<ProjectDraft>>;

