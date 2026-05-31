"use client";

import { Book, Camera, MapPin, Palette, Sparkles } from "lucide-react";
import type { ProjectDraft, ProjectDraftSetter } from "@/components/dashboard/types";

type ThemeOption = { value: string; label: string };

type ProjectCreateFormProps = {
  draft: ProjectDraft;
  themeOptions: ThemeOption[];
  busy: boolean;
  onDraftChange: ProjectDraftSetter;
  onCreate: () => void;
};

export function ProjectCreateForm({
  draft,
  themeOptions,
  busy,
  onDraftChange,
  onCreate,
}: ProjectCreateFormProps) {
  return (
    <div className="create-project-container">
      <div className="create-project-hero">
        <div className="hero-icon">
          <Book size={48} strokeWidth={1.5} />
        </div>
        <h1>创建新的相册项目</h1>
        <p>为一次旅程、一场活动或一组照片故事创建专属相册</p>
      </div>

      <div className="create-project-form">
        <div className="form-section">
          <label className="form-label">
            <Camera size={16} />
            <span>项目名称</span>
          </label>
          <input
            type="text"
            className="form-input-large"
            value={draft.title}
            onChange={(event) =>
              onDraftChange((current) => ({ ...current, title: event.target.value }))
            }
            placeholder="例如：2024年春节家庭聚会"
            autoFocus
          />
        </div>

        <div className="form-section">
          <label className="form-label">
            <Book size={16} />
            <span>项目描述</span>
          </label>
          <textarea
            className="form-textarea"
            rows={3}
            value={draft.description}
            onChange={(event) =>
              onDraftChange((current) => ({ ...current, description: event.target.value }))
            }
            placeholder="记录这个相册的背景和故事..."
          />
        </div>

        <div className="form-row">
          <div className="form-section">
            <label className="form-label">
              <MapPin size={16} />
              <span>地点</span>
            </label>
            <input
              type="text"
              className="form-input"
              value={draft.location}
              onChange={(event) =>
                onDraftChange((current) => ({ ...current, location: event.target.value }))
              }
              placeholder="例如：北京"
            />
          </div>

          <div className="form-section">
            <label className="form-label">
              <Palette size={16} />
              <span>主题风格</span>
            </label>
            <select
              className="form-select"
              value={draft.themeId}
              onChange={(event) =>
                onDraftChange((current) => ({ ...current, themeId: event.target.value }))
              }
            >
              {themeOptions.map((theme) => (
                <option key={theme.value} value={theme.value}>
                  {theme.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <button
          type="button"
          className="create-project-button"
          onClick={onCreate}
          disabled={busy || !draft.title.trim()}
        >
          <Sparkles size={20} />
          <span>{busy ? "创建中..." : "开始创建"}</span>
        </button>
      </div>
    </div>
  );
}

