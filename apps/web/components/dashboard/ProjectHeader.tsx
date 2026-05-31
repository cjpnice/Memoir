"use client";

import { Pencil, Save, Trash2 } from "lucide-react";
import type { Project, ProjectTask } from "@/lib/types";

type NextAction = {
  label: string;
};

type ProjectHeaderProps = {
  project: Project | null;
  nextAction: NextAction;
  projectRenaming: boolean;
  projectTitleDraft: string;
  busy: string;
  activeTask: ProjectTask | null;
  onStartRename: () => void;
  onDeleteProject: () => void;
  onSaveRename: () => void;
  onCancelRename: () => void;
  onProjectTitleChange: (value: string) => void;
  onProjectTitleKeyDown: (event: React.KeyboardEvent<HTMLInputElement>) => void;
};

function projectStatusLabel(status?: string) {
  switch (status) {
    case "draft":
      return "草稿";
    case "uploading":
      return "上传中";
    case "analyzing":
      return "AI 分析中";
    case "reviewing":
      return "待审核";
    case "generating_album":
      return "生成相册中";
    case "editing":
      return "编辑中";
    case "exporting":
      return "导出中";
    case "done":
      return "已完成";
    case "failed":
      return "失败";
    default:
      return status || "未开始";
  }
}

function projectTaskLabel(type?: string) {
  switch (type) {
    case "analysis":
      return "AI 分析";
    case "album_generation":
      return "相册生成";
    case "export":
      return "导出任务";
    default:
      return type || "任务";
  }
}

function projectTaskStatusLabel(status?: string) {
  switch (status) {
    case "running":
      return "进行中";
    case "completed":
      return "已完成";
    case "failed":
      return "失败";
    case "interrupted":
      return "中断";
    default:
      return status || "待处理";
  }
}

export function ProjectHeader({
  project,
  nextAction,
  projectRenaming,
  projectTitleDraft,
  busy,
  activeTask,
  onStartRename,
  onDeleteProject,
  onSaveRename,
  onCancelRename,
  onProjectTitleChange,
  onProjectTitleKeyDown,
}: ProjectHeaderProps) {
  const statusLabel = project ? projectStatusLabel(project.status) : "新项目";
  const stepLabel = project?.currentStep || "";
  const subtitleParts = [statusLabel, stepLabel, `下一步：${nextAction.label}`].filter(Boolean);

  return (
    <section className="project-header">
      <div className="project-heading">
        <div className="project-title-row">
          <h1>{project?.title || "创建新的相册项目"}</h1>
          {project ? (
            <div className="project-header-actions">
              <button
                type="button"
                className="btn-secondary"
                onClick={onStartRename}
                disabled={projectRenaming || busy === "project_rename"}
              >
                <Pencil size={14} style={{ marginRight: 4 }} />
                重命名
              </button>
              <button
                type="button"
                className="btn-danger"
                onClick={onDeleteProject}
                disabled={busy === "project_delete" || project.status === "analyzing"}
              >
                <Trash2 size={14} style={{ marginRight: 4 }} />
                删除
              </button>
            </div>
          ) : null}
        </div>
        {project && projectRenaming ? (
          <div className="project-rename-row">
            <input
              value={projectTitleDraft}
              onChange={(event) => onProjectTitleChange(event.target.value)}
              onKeyDown={onProjectTitleKeyDown}
              autoFocus
              placeholder="新的项目名称"
            />
            <button
              type="button"
              className="btn-primary"
              onClick={onSaveRename}
              disabled={busy === "project_rename"}
            >
              <Save size={16} style={{ marginRight: 6 }} />
              保存
            </button>
            <button type="button" className="btn-secondary" onClick={onCancelRename}>
              取消
            </button>
          </div>
        ) : (
          <p>{subtitleParts.join(" · ")}</p>
        )}
        {project ? (
          <p className="project-description">
            {project.description || "暂无描述"}
            {project.location ? ` — ${project.location}` : ""}
          </p>
        ) : (
          <p>
            从一个清晰的项目开始，后续导入、审核、生成和导出会分阶段展开。
          </p>
        )}
        {activeTask && activeTask.status !== "completed" ? (
          <div className="task-progress-card" data-tone={activeTask.status === "running" ? "warn" : activeTask.status === "failed" ? "bad" : "good"}>
            <div className="task-progress-head">
              <div>
                <div className="panel-title">
                  {projectTaskLabel(activeTask.type)} · {projectTaskStatusLabel(activeTask.status)}
                </div>
                <div className="panel-subtitle">
                  {activeTask.message || project?.currentStep || "任务正在更新进度"}
                </div>
              </div>
              <span className="pill" data-tone={activeTask.status === "running" ? "warn" : activeTask.status === "failed" ? "bad" : "good"}>
                {activeTask.progress}%
              </span>
            </div>
            <div className="progress-block task-progress">
              <div className="progress-track">
                <span style={{ width: `${Math.min(activeTask.progress, 100)}%` }} />
              </div>
              {activeTask.error ? (
                <div className="status-note" data-tone="bad">
                  {activeTask.error}
                </div>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
    </section>
  );
}
