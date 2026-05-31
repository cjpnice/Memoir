"use client";

import { type Dispatch, type SetStateAction } from "react";
import { ArrowLeft, Cpu, Image, Key, Link2, RefreshCw, Save, Sparkles, User } from "lucide-react";
import type { AISettings, GitHubSettings, Project } from "@/lib/types";

function GithubIcon({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
    </svg>
  );
}

type SettingsViewProps = {
  selectedProject: Project | null;
  aiSettings: AISettings;
  githubSettings: GitHubSettings;
  busy: string;
  settingsNote: string;
  githubSettingsNote: string;
  onAISettingsChange: Dispatch<SetStateAction<AISettings>>;
  onGitHubSettingsChange: Dispatch<SetStateAction<GitHubSettings>>;
  onSaveAISettings: () => void;
  onSaveGitHubSettings: () => void;
  onRefreshListing: () => void;
  onBack: () => void;
};

export function SettingsView({
  selectedProject,
  aiSettings,
  githubSettings,
  busy,
  settingsNote,
  githubSettingsNote,
  onAISettingsChange,
  onGitHubSettingsChange,
  onSaveAISettings,
  onSaveGitHubSettings,
  onRefreshListing,
  onBack,
}: SettingsViewProps) {
  const defaultBackLabel = selectedProject ? "返回工作区" : "返回项目";
  const isSavingAI = busy === "settings_ai";
  const isSavingGH = busy === "settings_github";

  return (
    <div className="ai-settings-page">
      {/* Header */}
      <header className="ai-settings-header">
        <button type="button" className="ai-settings-back" onClick={onBack}>
          <ArrowLeft size={18} />
          <span>{defaultBackLabel}</span>
        </button>
        <div className="ai-settings-title-block">
          <h1>设置</h1>
          <p>配置 AI 模型和 GitHub Pages 发布，保存后立即生效。</p>
        </div>
      </header>

      {/* Card: Vision Model */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <Cpu size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>图片理解模型</h2>
            <p>用于照片分析、筛选、相册文案和图片理解。</p>
          </div>
        </div>

        <div className="ai-settings-card-body">
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Link2 size={14} />
              Base URL
            </span>
            <input
              value={aiSettings.baseUrl}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, baseUrl: event.target.value }))
              }
              placeholder="https://api.openai.com/v1"
            />
          </label>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Sparkles size={14} />
              模型名称
            </span>
            <input
              value={aiSettings.model}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, model: event.target.value }))
              }
              placeholder="gpt-4o-mini"
            />
          </label>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Key size={14} />
              API Key
            </span>
            <input
              type="password"
              value={aiSettings.apiKey}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, apiKey: event.target.value }))
              }
              placeholder="sk-..."
            />
          </label>
        </div>
      </section>

      {/* Card: Image Generation Model */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon ai-settings-card-icon--image">
            <Image size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>图像优化模型</h2>
            <p>用于相册编辑器里的生成式图片优化。</p>
          </div>
        </div>

        <div className="ai-settings-card-body">
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Link2 size={14} />
              Base URL
            </span>
            <input
              value={aiSettings.imageBaseUrl}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, imageBaseUrl: event.target.value }))
              }
              placeholder="https://api.openai.com/v1"
            />
          </label>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Sparkles size={14} />
              模型名称
            </span>
            <input
              value={aiSettings.imageModel}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, imageModel: event.target.value }))
              }
              placeholder="gpt-image-1.5"
            />
          </label>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Key size={14} />
              API Key
            </span>
            <input
              type="password"
              value={aiSettings.imageApiKey}
              onChange={(event) =>
                onAISettingsChange((current) => ({ ...current, imageApiKey: event.target.value }))
              }
              placeholder="sk-..."
            />
          </label>
        </div>
      </section>

      {/* AI Save bar */}
      <div className="ai-settings-footer">
        {settingsNote ? (
          <span className="ai-settings-note">{settingsNote}</span>
        ) : (
          <span className="ai-settings-note ai-settings-note--hint">修改后记得点击保存</span>
        )}
        <button
          type="button"
          className="ai-settings-save-btn"
          onClick={onSaveAISettings}
          disabled={isSavingAI}
        >
          <Save size={16} />
          <span>{isSavingAI ? "保存中..." : "保存 AI 设置"}</span>
        </button>
      </div>

      {/* Card: GitHub Pages */}
      <section className="ai-settings-card ai-settings-card--github">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon ai-settings-card-icon--github">
            <GithubIcon size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>GitHub Pages</h2>
            <p>将相册发布到 GitHub Pages，随时随地在线查看和分享。</p>
          </div>
        </div>

        <div className="ai-settings-card-body">
          <div className="ai-settings-field-row">
            <label className="ai-settings-field">
              <span className="ai-settings-field-label">
                <User size={14} />
                所有者 / 组织
              </span>
              <input
                value={githubSettings.owner}
                onChange={(event) =>
                  onGitHubSettingsChange((current) => ({ ...current, owner: event.target.value }))
                }
                placeholder="your-username"
              />
            </label>
            <label className="ai-settings-field">
              <span className="ai-settings-field-label">
                <GithubIcon size={14} />
                仓库名称
              </span>
              <input
                value={githubSettings.repo}
                onChange={(event) =>
                  onGitHubSettingsChange((current) => ({ ...current, repo: event.target.value }))
                }
                placeholder="my-albums"
              />
            </label>
          </div>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Link2 size={14} />
              分支
            </span>
            <input
              value={githubSettings.branch}
              onChange={(event) =>
                onGitHubSettingsChange((current) => ({ ...current, branch: event.target.value }))
              }
              placeholder="main"
            />
          </label>
          <label className="ai-settings-field">
            <span className="ai-settings-field-label">
              <Key size={14} />
              Personal Access Token
            </span>
            <input
              type="password"
              value={githubSettings.token}
              onChange={(event) =>
                onGitHubSettingsChange((current) => ({ ...current, token: event.target.value }))
              }
              placeholder="ghp_..."
            />
          </label>
          <p className="ai-settings-hint">
            需要 Token 具有 <code>repo</code> 权限。发布后请在仓库 Settings → Pages 中将 Source 设置为 <code>Deploy from a branch</code>，Branch 选择 <code>{githubSettings.branch || "main"}</code>。
          </p>
        </div>
      </section>

      {/* GitHub Save bar */}
      <div className="ai-settings-footer">
        {githubSettingsNote ? (
          <span className="ai-settings-note">{githubSettingsNote}</span>
        ) : (
          <span className="ai-settings-note ai-settings-note--hint">配置后即可从导出页面发布相册</span>
        )}
        <button
          type="button"
          className="ai-settings-save-btn"
          onClick={onSaveGitHubSettings}
          disabled={isSavingGH}
        >
          <Save size={16} />
          <span>{isSavingGH ? "保存中..." : "保存 GitHub 设置"}</span>
        </button>
      </div>

      {/* Refresh Listing */}
      <section className="ai-settings-card ai-settings-card--listing">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon ai-settings-card-icon--listing">
            <RefreshCw size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>相册首页</h2>
            <p>
              每次发布相册时会自动更新首页。如需手动刷新，点击下方按钮重新生成。
            </p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <div className="listing-actions">
            <button
              type="button"
              className="listing-refresh-btn"
              onClick={onRefreshListing}
              disabled={busy === "refresh_listing"}
            >
              <RefreshCw size={16} />
              <span>{busy === "refresh_listing" ? "刷新中..." : "刷新相册首页"}</span>
            </button>
            <span className="listing-hint">
              首页地址：<code>https://{githubSettings.owner || "..."}.github.io/{githubSettings.repo || "..."}/</code>
            </span>
          </div>
        </div>
      </section>
    </div>
  );
}
