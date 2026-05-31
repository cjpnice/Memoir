"use client";

import { CirclePlus } from "lucide-react";
import type { Project } from "@/lib/types";

type ProjectSidebarProps = {
  projects: Project[];
  selectedProjectId: string;
  onSelectProject: (projectId: string) => void;
  onNewProject: () => void;
};

function projectStatusLabel(status?: string) {
  switch (status) {
    case "draft":
      return "草稿";
    case "uploading":
      return "上传中";
    case "analyzing":
      return "分析中";
    case "reviewing":
      return "待审核";
    case "generating_album":
      return "生成中";
    case "editing":
      return "编辑中";
    case "exporting":
      return "导出中";
    case "done":
      return "已完成";
    case "failed":
      return "失败";
    default:
      return status || "";
  }
}

export function ProjectSidebar({
  projects,
  selectedProjectId,
  onSelectProject,
  onNewProject,
}: ProjectSidebarProps) {
  return (
    <aside className="sidebar">
      <section className="sidebar-head">
        <div>
          <div className="panel-title">项目目录</div>
          <div className="panel-subtitle">{projects.length} 个</div>
        </div>
        <button type="button" className="icon-button" onClick={onNewProject} aria-label="新建项目">
          <CirclePlus size={18} />
        </button>
      </section>

      <section className="project-list">
        {projects.length === 0 ? (
          <div className="empty-state">还没有项目</div>
        ) : (
          projects.map((project) => (
            <button
              key={project.id}
              type="button"
              className="project-item"
              data-active={project.id === selectedProjectId}
              onClick={() => onSelectProject(project.id)}
            >
              <strong>{project.title}</strong>
              <span>
                {project.images.length} 张 · {projectStatusLabel(project.status)}
              </span>
            </button>
          ))
        )}
      </section>
    </aside>
  );
}
