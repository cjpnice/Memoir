# Memoir / 集忆

[English](README.en.md) | [中文文档](README.md)

**Memoir / 集忆** 是一个本地优先的 AI 相册创作工具。导入旅行、家庭或日常照片，让 AI 分析和筛选，然后生成精美的叙事相册——可以导出为 HTML，也可以直接发布到 GitHub Pages。

面向那些相册里已经攒了成千上万张有意义照片的人——照片不是不重要，而是太多、太散，整理和分享的工作总是一拖再拖。

## 为什么做集忆

拍照越来越容易，翻看却越来越难。一次周末可能留下几百张图：相似连拍、模糊抓拍，还有一些当下看起来普通、几年后却会突然变珍贵的细节。

集忆的核心问题是：**如果 AI 真的理解照片，它能不能帮人保存记忆，而不只是生成更多内容？**

项目遵循三个原则：

- **本地优先** — 照片、项目状态和导出内容默认留在你的机器上
- **人在回路中** — AI 给建议、生成草稿；你决定、编辑和发布
- **叙事成册** — 输出是有封面、节奏、配文和结尾的相册，不是照片清单

## 功能特性

### 核心流程

- **智能导入** — 上传 JPG、PNG、HEIC、HEIF 文件，实时进度跟踪
- **AI 分析** — 质量分、保存价值、故事价值评分，附推荐意见、问题提示、裁剪建议和文案种子
- **智能分组** — 自动聚合相似连拍，推荐每组最值得保留的一张
- **人工审核** — 审查 AI 决策、删除图片、撤销编辑、覆盖推荐
- **叙事相册生成** — 创建可编辑的相册，包含页面、标题、正文、配图和社交发布草稿
- **相册编辑** — 页面排序、撤销/重做、精细化叙事流程

### 导出与发布

- **HTML 导出** — 独立交互式相册，支持照片放大和响应式布局
- **长图导出** — 单张 PNG，方便在聊天应用分享
- **分享链接** — 本地 Web 服务器 URL，快速预览
- **GitHub Pages 发布** — 直接发布到 GitHub Pages，包含：
  - 自动提取相册封面和元数据
  - 精美的落地页，列出所有已发布相册
  - 实时上传进度跟踪
  - 自动生成相册首页，每次发布自动更新

### 其他功能

- **集忆地图** — 使用 EXIF 位置数据在交互式地图上可视化项目
- **社交文案生成** — AI 为小红书、朋友圈等平台生成文案，附推荐配图
- **设置持久化** — AI 模型和 GitHub Pages 配置本地保存
- **项目组织** — 多个项目，支持标题、描述、地点和视觉主题

## 技术栈

- **前端**: Next.js 16, React 19, TypeScript, lucide-react, react-map-gl
- **后端**: Go 1.26, Gin Web 框架
- **AI**: Eino 框架，OpenAI 兼容多模态模型
- **存储**: 本地 JSON 文件和媒体目录（无需外部数据库）
- **媒体处理**: Go 图像库，HEIC/HEIF 转换，Chrome 渲染 PDF
- **部署**: Docker Compose，单文件 Go 二进制（内嵌前端）

## 快速开始

### 环境要求

- Go 1.26+
- Node.js 22+
- npm

### 开发环境搭建

1. **安装依赖**：
   ```bash
   make setup
   ```

2. **配置环境**：
   ```bash
   cp .env.example .env
   ```

   编辑 `.env`，设置 OpenAI 兼容 API 凭证：
   ```bash
   OPENAI_BASE_URL=https://api.openai.com/v1
   OPENAI_API_KEY=你的API密钥
   OPENAI_MODEL=gpt-4o-mini
   ```

3. **启动 API 服务**（在一个终端）：
   ```bash
   make dev-api
   ```

4. **启动 Web 前端**（在另一个终端）：
   ```bash
   make dev-web
   ```

5. **打开应用**：
   ```
   http://localhost:3000
   ```

前端默认调用 `http://127.0.0.1:8090` 的 Go API。

### Docker 部署

```bash
cp .env.example .env
# 编辑 .env 填入 API 凭证
make docker-up
```

服务：
- Web: `http://localhost:3000`
- API: `http://localhost:8090`

### 单文件发布

构建内嵌前端的独立二进制：

```bash
make package
```

发布二进制输出到 `dist/`，运行对应平台的可执行文件即可启动本地服务。

## 使用流程

### 1. 创建项目

点击首页的"新建项目"，填写标题、描述、地点（可选）、语气和视觉主题。

### 2. 导入照片

上传 JPG、PNG、HEIC 或 HEIF 文件，应用会显示实时上传进度和最近导入。

### 3. AI 分析

点击"开始分析"，让 AI 模型：
- 为每张照片的质量、保存价值和故事潜力打分
- 识别问题（模糊、曝光、构图）
- 建议裁剪和改进
- 聚合相似连拍
- 生成文案种子

### 4. 审核决策

切换到"审核"标签页：
- 查看每张图片的 AI 推荐
- 覆盖决策（保留、排除、改进后保留）
- 删除不需要的照片
- 按各种条件筛选和排序

### 5. 生成相册

点击"生成相册"，创建叙事相册：
- 封面页，包含标题和简介
- 主题页面，包含标题、正文和配图
- 结尾备注
- 社交媒体文案草稿

### 6. 编辑相册

在"相册"标签页：
- 拖拽重新排序页面
- 编辑页面标题和正文
- 调整图片位置
- 撤销/重做修改

### 7. 导出

在"导出"标签页选择导出格式：

- **HTML** — 交互式网页相册，支持照片放大
- **长图** — 单张 PNG，方便聊天分享
- **分享链接** — 本地预览 URL
- **GitHub Pages** — 在线发布（需要配置）

导出结果会持久保存——刷新页面不会丢失。

## GitHub Pages 发布

将相册发布到 GitHub Pages，随时随地在线访问和分享。

### 配置

1. **创建 GitHub 仓库**：
   - 访问 [github.com/new](https://github.com/new)
   - 创建公开仓库（例如 `my-albums`）
   - 初始化 README

2. **生成 Personal Access Token**：
   - 访问 [GitHub Token 设置](https://github.com/settings/tokens/new)
   - 勾选 `repo` 权限
   - 生成并复制 token（以 `ghp_` 开头）

3. **在集忆中配置**：
   - 点击首页的 ⚙ 设置按钮
   - 填写 GitHub Pages 卡片：
     - **所有者**：你的 GitHub 用户名
     - **仓库名**：仓库名称（例如 `my-albums`）
     - **分支**：`main`（默认）
     - **Token**：你的 personal access token
   - 点击"保存 GitHub 设置"

4. **启用 GitHub Pages**（一次性操作）：
   - 进入仓库 Settings → Pages
   - Source: "Deploy from a branch"
   - Branch: `main`，folder: `/ (root)`
   - 保存

### 发布

1. 打开一个已生成相册的项目
2. 切换到"导出"标签页
3. 选择"GitHub Pages"导出类型
4. 点击"发布到 GitHub Pages"
5. 观看实时进度，图片和 HTML 逐步上传
6. 完成后复制发布 URL

相册地址：
```
https://{用户名}.github.io/{仓库名}/albums/{相册slug}/
```

### 相册首页

集忆会自动创建并维护仓库根目录的落地页：
```
https://{用户名}.github.io/{仓库名}/
```

首页展示：
- 相册卡片网格，包含封面图
- 相册标题和简介文字
- 发布日期
- 响应式布局，适配手机和桌面

每次发布新相册时，首页会自动更新。也可以在设置页面手动刷新。

### 手动刷新

如果删除了相册或需要重新生成首页：
1. 打开设置（首页 ⚙ 按钮）
2. 在 GitHub Pages 区域点击"刷新相册首页"

## 配置

### 环境变量

复制 `.env.example` 到 `.env` 并配置：

```bash
# 服务器
PORT=8090
DATA_DIR=./data

# CORS（本地开发留空即可）
ALLOWED_ORIGINS=

# 上传限制
MAX_UPLOAD_MB=256
MAX_UPLOAD_FILES=200

# 前端 API 地址
NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8090

# OpenAI 兼容多模态模型（AI 功能必需）
OPENAI_BASE_URL=
OPENAI_API_KEY=
OPENAI_MODEL=gpt-4o-mini

# 可选：独立的图像编辑模型
OPENAI_IMAGE_BASE_URL=
OPENAI_IMAGE_API_KEY=
OPENAI_IMAGE_MODEL=gpt-image-1.5
```

### AI 模型要求

集忆支持任何 OpenAI 兼容的多模态模型（需支持图像输入）。常用选项：

- OpenAI GPT-4o / GPT-4o-mini
- Anthropic Claude（通过兼容代理）
- 本地模型（需 OpenAI 兼容 API）

不配置 API Key 时，应用正常启动，但分析和相册生成会返回配置错误提示。

### 设置持久化

AI 模型和 GitHub Pages 设置保存在：
```
{DATA_DIR}/ai-settings.json
{DATA_DIR}/github-settings.json
```

在 Web UI 中配置的设置会覆盖环境变量。

## 项目结构

```
Memoir/
├── apps/web/              # Next.js 前端
│   ├── app/               # App router 页面
│   ├── components/        # React 组件
│   │   └── dashboard/     # 主工作区视图
│   ├── lib/               # 工具函数和 API 客户端
│   └── public/            # 静态资源
├── cmd/
│   ├── api/               # 开发环境 API 入口
│   └── memoir/            # 单文件发布入口
├── internal/
│   ├── ai/                # AI 分析器和模型适配器
│   ├── app/               # 服务层和业务逻辑
│   ├── config/            # 配置管理
│   ├── domain/            # 领域模型（Project, Album, Image）
│   ├── httpapi/           # HTTP 处理器和路由
│   ├── media/             # 图像处理和存储
│   ├── store/             # JSON 文件持久化
│   └── webassets/         # 内嵌前端资源
├── scripts/               # 构建和发布脚本
├── compose.yaml           # Docker Compose 配置
└── Makefile               # 开发命令
```

## 常用命令

```bash
make setup        # 安装前端依赖
make dev          # 同时启动 API 和 Web
make dev-api      # 启动 Go API 服务
make dev-web      # 启动 Next.js 开发服务器
make test         # 运行测试和构建检查
make build        # 构建 Go 包和 Web 应用
make package      # 创建单文件发布二进制
make fmt          # 格式化 Go 代码
make clean        # 删除 data/ 和构建产物
make docker-up    # 启动 Docker Compose 服务
make docker-down  # 停止 Docker Compose 服务
```

## AI 与隐私

- **默认本地** — 所有照片、项目数据和导出内容留在你的机器上
- **API 调用** — 只有图像分析请求会发送到你配置的 AI 提供商
- **无遥测** — 集忆不收集使用数据或分析
- **你的控制** — 你选择哪些照片分析、哪些发布

不要把 `.env`、`data/`、私人照片、API Key 或个人相册提交到版本控制。

## 贡献

集忆仍处于 pre-1.0，积极开发中。欢迎从这些方向参与：

- **测试** — 添加前端交互测试和 API 集成测试
- **导出格式** — PDF、EPUB、印刷版式
- **相册主题** — 更多视觉主题和页面布局
- **AI 提供商** — 支持更多多模态模型提供商
- **部署** — Kubernetes 配置、安全加固、自托管指南
- **文档** — 教程、视频指南和故障排查

贡献前请先阅读 [贡献指南](CONTRIBUTING.zh-CN.md) 和 [安全策略](SECURITY.zh-CN.md)。

## 许可证

MIT 许可证。详见 [LICENSE](LICENSE)。

## 致谢

- 使用 [Next.js](https://nextjs.org/)、[React](https://react.dev/) 和 [Go](https://go.dev/) 构建
- AI 集成基于 [Eino](https://github.com/cloudwego/eino) 框架
- 图标来自 [Lucide](https://lucide.dev/)
- 地图由 [MapLibre GL](https://maplibre.org/) 提供支持

---

**用 ❤️ 制作，为保存有意义的记忆而生**
