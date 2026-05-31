# 贡献指南

[English](CONTRIBUTING.md)

感谢你愿意帮助 Memoir / 集忆。这个项目还处于 pre-1.0，最重要的目标是把“本地优先的 AI 记忆整理工作流”打磨得可靠、可解释、可复现。

## 本地开发

需要 Go 1.26+、Node.js 22+ 和 npm。

```bash
make setup
cp .env.example .env
make dev-api
make dev-web
```

不配置 API Key 也可以开发。应用正常启动，但分析和相册生成会返回需要配置 API Key 的提示。上传、项目管理和导出流程不需要 API Key。

## 验证

提交 PR 前请运行：

```bash
make test
```

这会执行 Web 弹窗策略检查、Go 测试和前端生产构建。只改文档时，也请在 PR 里说明你做过的链接和命令一致性检查。

影响发布包时，请额外运行：

```bash
make package
```

## 代码边界

- Go API 入口在 `cmd/api`。
- 单文件应用入口在 `cmd/memoir`。
- 核心业务流程在 `internal/app`。
- AI 接口和模型适配在 `internal/ai`。
- HTTP 路由和 CORS 行为在 `internal/httpapi`。
- 上传、缩略图、图片编辑和导出媒体在 `internal/media`。
- 前端应用在 `apps/web`。

请优先做小而明确的改动。涉及 AI、导出、上传、安全边界或数据结构时，请补充测试或写清楚手动验证步骤。

## 文档规则

面向用户、贡献者和部署者的文档需要维护中英文 companion：

- 根目录文档：`README.md` 和 `README.zh-CN.md`。
- `docs/` 英文文档：`docs/*.md`。
- `docs/zh-CN/` 中文文档：`docs/zh-CN/*.md`。

中英文不需要逐字翻译，但必须表达同一事实、同一约束和同一命令。

## 隐私规则

不要提交：

- `.env` 或任何 API Key。
- `data/` 运行时状态。
- 真实照片、私人导出相册或分享页。
- `samples/demo-photos/` 或 `images/` 等本地素材目录。
- `dist/` 发布产物。
- `apps/web/node_modules/`、`.next/`、`out/`。

公开截图、视频和测试素材必须使用已获授权的图片，并在发布前移除 EXIF/GPS 元数据。

## 适合开始的方向

- 给上传、审核、相册编辑、导出流程增加前端交互测试。
- 增加更多 album theme 和导出模板。
- 增强 Docker、自托管、反向代理和单文件应用文档。
- 复核 AI 供应商兼容性、错误提示和隐私说明。

## 提交 PR

PR 描述请包含：

- 改了什么。
- 为什么需要。
- 如何验证。
- 是否影响图片隐私、AI 供应商、导出格式、单文件打包或公开部署。

如果问题涉及私人照片，请使用脱敏或可公开授权的素材复现。
