"use client";

import { ArrowLeft, BookOpen, CheckCircle2, Circle, Key, Rocket, Settings, Upload } from "lucide-react";

type GuideViewProps = {
  onBack: () => void;
  onOpenSettings: () => void;
};

export function GuideView({ onBack, onOpenSettings }: GuideViewProps) {
  return (
    <div className="ai-settings-page">
      {/* Header */}
      <header className="ai-settings-header">
        <button type="button" className="ai-settings-back" onClick={onBack}>
          <ArrowLeft size={18} />
          <span>返回首页</span>
        </button>
        <div className="ai-settings-title-block">
          <h1>GitHub Pages 使用指南</h1>
          <p>将相册发布到 GitHub Pages，随时随地在线查看和分享。</p>
        </div>
      </header>

      {/* Intro */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <BookOpen size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>什么是 GitHub Pages 发布？</h2>
            <p>
              GitHub Pages 是 GitHub 提供的免费静态网站托管服务。集忆可以将生成的相册直接发布到你的 GitHub 仓库，
              自动生成一个可公开访问的网址，无需服务器或域名配置。
            </p>
          </div>
        </div>
      </section>

      {/* Step 1: Create Repo */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <Circle size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>第一步：创建 GitHub 仓库</h2>
            <p>在 GitHub 上创建一个新的公开仓库来存放相册文件。</p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <ol className="guide-steps">
            <li>
              打开 <a href="https://github.com/new" target="_blank" rel="noreferrer">github.com/new</a> 创建新仓库
            </li>
            <li>仓库名称填写一个易记的名字，例如 <code>my-albums</code></li>
            <li>可见性选择 <strong>Public</strong>（公开）</li>
            <li>勾选 <strong>Add a README file</strong>（推荐）</li>
            <li>点击 <strong>Create repository</strong></li>
          </ol>
        </div>
      </section>

      {/* Step 2: Generate Token */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <Key size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>第二步：生成 Personal Access Token</h2>
            <p>Token 用于集忆代你上传文件到仓库，不会在本地保存密码。</p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <ol className="guide-steps">
            <li>
              打开 <a href="https://github.com/settings/tokens/new" target="_blank" rel="noreferrer">GitHub Token 创建页面</a>
            </li>
            <li>在 <strong>Note</strong> 中填写备注，例如 <code>memoir-albums</code></li>
            <li>设置过期时间（建议选择 <strong>90 days</strong> 或自定义更长）</li>
            <li>在 <strong>Select scopes</strong> 中勾选 <code>repo</code>（整个 repo 权限组）</li>
            <li>滚动到底部，点击 <strong>Generate token</strong></li>
            <li>
              复制生成的 Token（以 <code>ghp_</code> 开头），<strong>页面刷新后将无法再次查看</strong>
            </li>
          </ol>
        </div>
      </section>

      {/* Step 3: Configure in Memoir */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <Settings size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>第三步：在集忆中配置</h2>
            <p>将仓库信息和 Token 填入集忆的设置页面。</p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <ol className="guide-steps">
            <li>
              点击首页右上角的 <strong>⚙ 设置按钮</strong>，或者{" "}
              <button type="button" className="guide-link" onClick={onOpenSettings}>
                直接打开设置页
              </button>
            </li>
            <li>
              在 <strong>GitHub Pages</strong> 卡片中填写：
              <ul className="guide-substeps">
                <li><strong>所有者/组织</strong>：你的 GitHub 用户名</li>
                <li><strong>仓库名称</strong>：刚创建的仓库名，例如 <code>my-albums</code></li>
                <li><strong>分支</strong>：保持默认 <code>main</code> 即可</li>
                <li><strong>Token</strong>：粘贴第二步复制的 Token</li>
              </ul>
            </li>
            <li>点击 <strong>保存 GitHub 设置</strong></li>
          </ol>
        </div>
      </section>

      {/* Step 4: Enable GitHub Pages */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <Rocket size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>第四步：在 GitHub 启用 Pages</h2>
            <p>首次使用前需要在仓库设置中手动开启 GitHub Pages 功能。</p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <ol className="guide-steps">
            <li>打开你的仓库页面，进入 <strong>Settings</strong> 标签</li>
            <li>左侧菜单找到 <strong>Pages</strong></li>
            <li>
              在 <strong>Source</strong> 下拉菜单中选择 <code>Deploy from a branch</code>
            </li>
            <li>
              Branch 选择 <code>main</code>，目录选择 <code>/ (root)</code>
            </li>
            <li>点击 <strong>Save</strong></li>
            <li>
              等待 1–2 分钟后，你的相册将在以下地址可访问：
              <div className="guide-url-hint">
                <code>https://你的用户名.github.io/仓库名/</code>
              </div>
            </li>
          </ol>
        </div>
      </section>

      {/* Step 5: Publish */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon ai-settings-card-icon--github">
            <Upload size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>第五步：发布相册</h2>
            <p>配置完成后，从导出页面一键发布。</p>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <ol className="guide-steps">
            <li>进入任意已完成编辑的相册项目</li>
            <li>切换到 <strong>导出</strong> 标签</li>
            <li>选择 <strong>GitHub Pages</strong> 导出方式</li>
            <li>点击 <strong>发布到 GitHub Pages</strong></li>
            <li>发布完成后，复制返回的链接即可访问在线相册</li>
          </ol>
          <div className="guide-callout">
            <CheckCircle2 size={16} />
            <span>
              每次发布都会自动更新相册列表页。重复发布同一相册会覆盖旧版本，不会产生重复文件。
            </span>
          </div>
        </div>
      </section>

      {/* FAQ */}
      <section className="ai-settings-card">
        <div className="ai-settings-card-header">
          <div className="ai-settings-card-icon">
            <BookOpen size={22} />
          </div>
          <div className="ai-settings-card-title">
            <h2>常见问题</h2>
          </div>
        </div>
        <div className="ai-settings-card-body">
          <dl className="guide-faq">
            <dt>发布后访问页面显示 404？</dt>
            <dd>
              确认已在仓库 Settings → Pages 中启用 GitHub Pages，并选择了 <code>main</code> 分支。
              首次部署通常需要 1–2 分钟生效。
            </dd>

            <dt>Token 过期了怎么办？</dt>
            <dd>
              重新生成一个新 Token，在集忆设置页面更新后保存即可。已发布的相册不会受影响。
            </dd>

            <dt>仓库可以是私有的吗？</dt>
            <dd>
              GitHub Pages 对私有仓库有访问限制（需要 Pro 账户）。建议使用公开仓库。
            </dd>

            <dt>相册图片有大小限制吗？</dt>
            <dd>
              GitHub Contents API 支持单个文件最大约 50MB。普通照片通常远小于此限制。
            </dd>

            <dt>可以自定义域名吗？</dt>
            <dd>
              可以。在仓库 Settings → Pages 中配置自定义域名，GitHub 会自动处理 HTTPS 证书。
            </dd>
          </dl>
        </div>
      </section>
    </div>
  );
}
