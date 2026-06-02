# 轻量独立发布中心方案

更新时间：2026-06-02

## 1. 结论

建议先做轻量版发布中心：

```text
不自研 CI Runner
不自研多语言构建平台
不自研 Docker / Kubernetes 部署编排
```

发布中心只负责：

```text
制品登记
版本管理
发布说明
审批
灰度
回滚
审计
客户端下载入口
发布观测
```

构建和部署尽量交给现成工具：

```text
GitHub Actions
Gitea Actions
Jenkins
本地脚本
Cloudflare Wrangler
```

这样一期可以快速落地，后面再决定是否扩展成完整 CI/CD 平台。

## 2. GitHub CLI 能不能直接用

可以直接用，但边界要清楚。

GitHub CLI `gh` 是 GitHub 官方命令行工具，可以在终端或脚本里使用。官方文档说明它支持用 `GITHUB_TOKEN` 做自动化认证，也支持 GitHub Enterprise Server。参考：GitHub CLI manual。

适合直接使用的能力：

```text
gh auth login
gh repo clone
gh workflow run
gh run watch
gh run view
gh run download
gh release create
gh release upload
gh api
```

也就是说，发布中心不需要自己实现 GitHub Actions 调度，可以通过：

```bash
gh workflow run build.yml --ref main
gh run watch
gh run download
```

触发构建、等待构建、下载构建产物。

官方 GitHub Actions 文档也明确：如果 workflow 配置了 `workflow_dispatch`，可以通过 GitHub CLI 运行 workflow。

限制：

- `gh` 主要服务 GitHub 和 GitHub Enterprise。
- 自建 Gitea 不能直接当成 GitHub 用。
- 对 Gitea 建议走 Gitea Actions、Webhook、Git 拉取、Gitea API，或者使用 Gitea 自己的 CLI/API。
- 发布中心不要把 `gh` 做成唯一依赖；应该把它作为 GitHub Provider 的一种实现。

## 3. Cloudflare Workers 和 Pages 能不能用

可以，而且很适合轻量版。

推荐用法：

```text
Cloudflare Pages
  管理后台前端
  文档站
  静态资源页面

Cloudflare Workers
  Webhook 接收
  客户端 update-check API
  资源 manifest API
  简单 Admin API
  下载鉴权/短链跳转

Cloudflare R2
  APK
  资源 ZIP
  Manifest
  Web dist 包

Cloudflare D1 或外部 PostgreSQL
  发布元数据
  构建记录
  审批记录
  审计日志
```

Cloudflare Pages 支持直接上传预构建静态资源，并且官方文档给出了：

```bash
CLOUDFLARE_ACCOUNT_ID=<ACCOUNT_ID> npx wrangler pages deploy <DIRECTORY> --project-name=<PROJECT_NAME>
```

Cloudflare Workers 也可以用 Wrangler 直接部署：

```bash
npx wrangler deploy
```

所以轻量版可以不用自己建设部署平台，直接让 GitHub Actions / Gitea Actions 调 Wrangler。

## 4. 轻量版总体架构

推荐架构结论：

```text
一期先做 Go 后端发布中心
  ↓
接入 GitHub/Gitea 的外部 CI 上传制品
  ↓
管理后台可以后续放到 Cloudflare Pages
  ↓
Webhook、下载鉴权、边缘缓存后续再扩展 Cloudflare Workers
```

不建议一期直接用纯 Cloudflare Workers + D1 重写发布中心。原因是当前仓库已经有 Go 后端、PostgreSQL 风格的数据模型、文件存储和 App 发布逻辑，先复用这部分更快，也更容易处理 APK/ZIP 上传、资源校验、审计和后续迁移。

```text
GitHub / Gitea
  ↓ push / tag / manual
GitHub Actions / Gitea Actions
  ↓ build
APK / ZIP / dist / Docker image
  ↓ upload metadata
Release Center API
  ↓
审批 / 灰度 / 发布 / 回滚 / 审计
  ↓
客户端 update-check / resource-check
```

如果使用 Cloudflare：

```text
┌──────────────────────────────┐
│ Cloudflare Pages              │
│ 发布中心管理后台              │
└───────────────┬──────────────┘
                │
┌───────────────▼──────────────┐
│ Cloudflare Workers            │
│ API / Webhook / 下载鉴权       │
└───────┬──────────────┬───────┘
        │              │
┌───────▼──────┐ ┌─────▼───────┐
│ D1/Postgres   │ │ R2/File Store │
│ 元数据         │ │ APK/ZIP/dist  │
└──────────────┘ └─────────────┘
```

也可以继续用现有 Go 后端：

```text
Cloudflare Pages 只放管理后台
Go 后端继续提供发布 API
/srv/files 或 R2 存文件
PostgreSQL 存元数据
```

这一版更稳，因为当前仓库已经有 Go 后端模块。

## 4.1 三种架构对比

| 架构 | 做法 | 优点 | 缺点 | 建议 |
|---|---|---|---|---|
| Go 后端优先 | Go API + PostgreSQL + `/srv/files`，CI 用 `curl/releasectl` 上传 | 复用现有代码；适合 APK/ZIP；数据模型清晰；后续可扩展 | 前端和边缘能力要后补 | 最推荐一期 |
| Cloudflare 优先 | Workers + Pages + R2 + D1 | 部署轻；边缘访问快；前端上线方便 | 大文件上传、复杂后台、审计和迁移会更麻烦 | 适合很轻的元数据服务 |
| 混合架构 | Go 做核心 API，Pages 做后台，Workers 做 Webhook/下载鉴权，R2 存文件 | 兼顾稳定和轻量；扩展自然 | 模块稍多，需要边界清楚 | 推荐二期演进 |

最终推荐：

```text
MVP：Go 后端 + PostgreSQL + 本地文件/对象存储 + 外部 CI
二期：Cloudflare Pages 承载管理后台
三期：Cloudflare Workers 承载 Webhook、下载鉴权、边缘 Manifest
四期：R2 替换或补充 /srv/files
```

## 4.2 推荐一期架构图

```text
GitHub Actions / Gitea Actions / 本地脚本
  ↓
curl 或 releasectl 上传 APK/ZIP/元数据
  ↓
Go Release Center API
  ↓
PostgreSQL
  ↓
/srv/files 或 R2
  ↓
客户端 update-check / resource-check
```

管理后台一期可以先走普通 Web 部署，或者后置到 Cloudflare Pages：

```text
Admin Web
  ↓
Go Release Center Admin API
```

## 4.3 后期接入 Cloudflare 的位置

Cloudflare 不需要一开始承载全部发布中心。更合理的位置是：

```text
Cloudflare Pages
  管理后台静态前端

Cloudflare Workers
  GitHub/Gitea Webhook 预处理
  下载 URL 鉴权
  Manifest 边缘缓存
  简单健康检查 API

Cloudflare R2
  APK
  资源 ZIP
  Manifest
  Web dist 包
```

Go 后端仍然保留核心职责：

```text
版本决策
灰度命中
审批状态
审计记录
资源安全校验
客户端更新策略
```

## 5. MVP 范围

一期只做这些：

1. 应用管理。
2. 构建记录登记。
3. 制品上传或制品 URL 登记。
4. APK 发布。
5. 资源增量发布。
6. 发布说明编辑。
7. 发布、暂停、召回、回滚。
8. 百分比灰度。
9. 客户端检查更新。
10. 资源 Manifest 和下载。
11. 更新事件上报。
12. 操作审计。
13. GitHub/Gitea Webhook 记录。
14. GitHub Actions/Gitea Actions 上传制品。

不做：

```text
内置 Runner
多语言流水线引擎
构建缓存系统
复杂 Secret 管理
Kubernetes 编排
Docker 镜像构建服务
iOS macOS Runner
复杂审批流
复杂发布日历
```

## 6. GitHub Actions 推荐流程

Android APK：

```text
push / tag
  ↓
GitHub Actions 构建 APK
  ↓
计算 SHA-256
  ↓
上传 APK 到发布中心
  ↓
发布中心创建 build + artifact
  ↓
人工在后台创建发布并审批
```

示例命令：

```bash
curl -X POST "$RELEASE_CENTER_BASE_URL/api/v1/ci/artifacts" \
  -H "Authorization: Bearer $RELEASE_CENTER_TOKEN" \
  -F "metadata=@artifact.json" \
  -F "file=@app-release.apk"
```

如果要用 GitHub CLI 触发：

```bash
gh workflow run android-build.yml --ref main
gh run watch
```

## 7. Gitea Actions 推荐流程

Gitea 不建议硬套 `gh`。

推荐：

```text
Gitea Webhook
  ↓
Gitea Actions 构建
  ↓
curl 上传制品到发布中心
```

发布中心只需要支持：

```text
POST /api/v1/webhooks/gitea
POST /api/v1/ci/builds
POST /api/v1/ci/artifacts
```

Gitea Actions 里的上传方式仍然用 `curl` 或 `releasectl`。

## 8. Cloudflare Pages 发布

适合：

```text
管理后台前端
官网
文档站
纯静态 Web 应用
```

GitHub Actions / Gitea Actions 先执行项目自己的构建命令：

```bash
npm ci
npm run build
```

然后部署到 Pages：

```bash
CLOUDFLARE_ACCOUNT_ID="$CLOUDFLARE_ACCOUNT_ID" \
npx wrangler pages deploy dist --project-name="$CLOUDFLARE_PAGES_PROJECT"
```

发布中心只记录：

```text
project_name
branch
commit
pages_url
deployment_id
status
```

不需要自己上传静态文件。

## 9. Cloudflare Workers 发布

适合：

```text
轻量 API
Webhook 接收
边缘鉴权
下载短链
Manifest API
```

部署命令：

```bash
npx wrangler deploy
```

如果 Workers 作为发布中心 API，需要注意：

- 大文件不要进 Worker 请求体，APK/ZIP 应直接传 R2 或现有文件服务。
- Worker 适合处理元数据、鉴权和跳转。
- 长任务构建不放 Worker 里做。
- 数据量较小时用 D1；如果已有 PostgreSQL，优先保留 PostgreSQL。

## 10. 轻量数据模型

只保留最小表：

```text
apps
repositories
builds
artifacts
releases
release_rules
installations
update_events
audit_logs
webhook_events
```

不建：

```text
pipelines
pipeline_runs
pipeline_steps
runner_agents
deployment_targets
deployments
```

如果后面真要自研 CI/CD，再补这些表。

## 11. 轻量 API

管理后台：

```text
GET    /admin/api/apps
POST   /admin/api/apps
GET    /admin/api/builds
GET    /admin/api/artifacts
POST   /admin/api/releases
POST   /admin/api/releases/{release_id}/publish
POST   /admin/api/releases/{release_id}/pause
POST   /admin/api/releases/{release_id}/rollback
GET    /admin/api/audit-logs
```

CI 接入：

```text
POST /api/v1/ci/builds
POST /api/v1/ci/artifacts
POST /api/v1/ci/releases
```

Webhook：

```text
POST /api/v1/webhooks/github
POST /api/v1/webhooks/gitea
```

客户端：

```text
POST /api/v1/apps/update-check
POST /api/v1/apps/resource-check
GET  /api/v1/apps/resources/{resource_id}/manifest
GET  /api/v1/apps/resources/packages/{package_id}/download
POST /api/v1/apps/update-events
```

## 12. 轻量版落地顺序

### M1：保留现有 Go 后端，独立发布 API

- 复用现有 `appreleases`。
- 增加 GitHub/Gitea webhook 事件记录。
- 增加 CI Token。
- 增加制品上传和 URL 登记。

### M2：管理后台前端上 Cloudflare Pages

- 后台前端构建为静态资源。
- 使用 Wrangler 部署到 Cloudflare Pages。
- 后台调用 Go 后端或 Worker API。

### M3：文件存储切到 R2 或保留 `/srv/files`

- APK 和资源 ZIP 仍走文件存储。
- Worker/Go API 只返回下载 URL。
- 客户端继续校验 SHA-256。

### M4：GitHub/Gitea CI 模板

- 提供 Android APK 上传模板。
- 提供资源 ZIP 上传模板。
- 提供 Web dist 部署 Pages 模板。
- 提供 Worker deploy 模板。

### M5：再评估是否自研 Runner

只有当外部 CI 不够用，再做：

```text
pipeline_templates
pipeline_runs
runner_agents
内置 Docker Runner
```

## 13. 推荐取舍

最推荐的一期组合：

```text
构建：GitHub Actions / Gitea Actions
触发：Git Webhook + 手工按钮
上传：curl / releasectl
发布 API：现有 Go 后端
管理后台：Cloudflare Pages
轻量边缘 API：Cloudflare Workers，可选
文件：/srv/files 起步，后续 R2
数据库：PostgreSQL 起步，轻量纯 Cloudflare 版本可用 D1
```

这个组合的好处：

- 不重。
- 能快速上线。
- 不重复造 GitHub Actions。
- Gitea 自建也能接。
- Cloudflare Pages/Workers 可以承接前端和轻量 API。
- 后续仍然能扩展到完整 CI/CD。

## 14. 参考资料

- GitHub CLI manual: https://cli.github.com/manual/
- GitHub Actions 手动运行 Workflow: https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow
- Cloudflare Workers Wrangler deploy: https://developers.cloudflare.com/workers/wrangler/commands/workers/
- Cloudflare Pages Direct Upload: https://developers.cloudflare.com/pages/get-started/direct-upload/
- Cloudflare Pages 使用 Wrangler 做 CI 部署: https://developers.cloudflare.com/pages/how-to/use-direct-upload-with-continuous-integration/
