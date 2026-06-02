# App 发版中心实现说明

本文是 `docs/app-release-center-spec.md` 的当前落地说明，用来区分整包发版、增量资源发版和 CI/CLI 操作入口。

## 当前总进度快照

更新时间：2026-06-02

当前 `/root/release-center` 已形成 App 发版中心的前后端主闭环：

| 模块 | 当前状态 |
|---|---|
| 后端 API | App 管理、构建登记、APK 发布、资源发布、灰度、暂停、召回、回滚、下载、事件上报、heartbeat、task-preflight、审计均已落地 |
| PostgreSQL | 一期 schema migration 覆盖 apps、builds、releases、resource versions、installations、update events、webhook events、audit events；系统管理 migration 覆盖用户、角色、权限、字典和菜单 |
| CLI / CI | `releasectl` 覆盖 artifact upload、resource pack/upload、Manifest 公钥导出和验签；Makefile 和 GitHub Actions 覆盖 test/build/resource catalog smoke |
| 后台 Web | `web/` 独立 Vite 管理台已具备登录入口、首页、系统管理页面和 App 发版中心业务模块；用户/角色/权限/数据字典/菜单编辑已从前端本地示例状态切到后端 API 数据，菜单可见性已驱动侧栏导航和首页模块入口 |
| 资源安全 | 增量资源白名单、ZIP 内容校验、禁止执行代码、SHA-256、文件大小校验和 Manifest Ed25519 签名已落地 |
| 自动保护 | `activation_failed` 会自动暂停匹配资源版本，写入 `paused_at` 并记录 `resource.auto_pause` 审计 |
| 质量观察 | overview API 输出 `quality_metrics`、`quality_policy`、`quality_alerts`；Web 统计页展示成功率、失败率、失败原因、阈值、策略建议和质量告警 |
| 本机验证 | `go test ./...`、`go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate`、`cd web && npm run build` 已通过；系统管理 API 已在 19082 无数据库 Demo fallback 模式 smoke；真实 DB smoke 覆盖迁移、发布、资源发布、检查更新和自动暂停 |
| 外部待验收 | 生产 18080 migration/重启、系统管理真实 PostgreSQL 环境 smoke、RBAC API 权限拦截、真实 APK Android 安装升级 smoke、真实资源 Android 下载/验签/激活/回滚 smoke 仍需继续推进 |

## 当前项目推进与优化点

### 已推进完成

当前项目已经从“App 发版页面”推进到轻量发布中心的一期主闭环：

- 后端闭环：App、构建、APK release、资源版本、下载、灰度、暂停、召回、回滚、事件、heartbeat、task-preflight 和审计 API 已落地。
- 数据闭环：PostgreSQL migration 已覆盖一期核心表，`cmd/migrate` 可按顺序执行迁移并写入 `schema_migrations`。
- CI / CLI 闭环：`releasectl` 已支持 APK/AAB/ZIP/Web dist 制品上传或 URL 登记、资源打包上传、Manifest 公钥导出和本地验签。
- 资源安全闭环：增量资源白名单、ZIP 内容校验、禁止执行代码、SHA-256、文件大小校验、Manifest Ed25519 签名已落地。
- 发布保护闭环：资源 `activation_failed` 会自动暂停对应资源版本，并写入 `resource.auto_pause` 审计。
- 观测闭环：overview API 和 Web 统计页已展示 APK/资源成功率、失败率、失败原因、质量阈值、策略建议和质量告警。
- 系统管理闭环：已新增用户、角色、权限、数据字典、菜单 PostgreSQL 表和 Admin API，前端系统管理页面已接入后端 overview/create/enable/disable/show/hide 接口；菜单 visible 状态驱动侧栏导航和首页模块入口；无数据库时后端提供进程内可变 Demo fallback 便于本地调试。
- 前端闭环：`web/` 已形成独立 Vite 管理台，具备登录、首页、系统管理页面和 App 发版中心业务模块。
- 本地验证闭环：Go test、Go build、Web build 和真实 DB smoke 均已跑通，覆盖迁移、发布、检查更新、资源发布和自动暂停。

### 仍需验收

这些事项不是代码完全缺失，而是需要在真实环境或客户端侧完成确认：

| 方向 | 待验收内容 | 建议优先级 |
|---|---|---|
| 生产服务 | 生产 18080 使用真实 `DATABASE_URL` 跑 migration，确认 `/readyz`、后台页面和核心 API | P0 |
| APK 真机 | Android 客户端检查更新、下载 APK、校验 SHA-256、调起系统安装器、安装结果上报 | P0 |
| 资源真机 | Android 客户端 resource-check、Manifest Ed25519 验签、ZIP 下载校验、解压激活、失败回滚 | P0 |
| CI 真链路 | GitHub/Gitea 受保护环境配置真实 token，执行 artifact-upload 和 resource-upload | P0 |
| 系统管理 | 生产 PostgreSQL 执行系统管理 migration 后，验证用户/角色/权限/菜单/字典真实持久化读写 | P1 |
| 权限拦截 | 基于用户、角色、权限点接入后台路由、Admin API、CI API、Webhook API 分组鉴权和操作审计 | P1 |
| 静态资源 | 生产 `web/dist` 托管、缓存策略、刷新策略和浏览器 smoke | P1 |
| 告警通知 | 质量告警接入外部通知渠道，并明确自动执行策略边界 | P2 |

### 近期优化点

建议近期围绕“上线可用、真机可信、后台可管”推进：

1. 生产部署固化：补齐 systemd 或容器启动配置、环境变量模板、文件目录权限、日志路径和健康检查脚本。
2. 真实 DB 迁移预案：明确迁移前备份、迁移执行、失败回滚和版本核对步骤。
3. Android smoke 手册：沉淀 APK 升级、资源激活、验签失败、激活失败自动暂停的真机测试步骤。
4. 系统管理验收：在真实 PostgreSQL 上执行 `000002_system_management_schema`，验证后台新增、启停、菜单显隐和审计记录。
5. 认证授权收敛：统一 Admin Token、CI Token、Webhook 签名校验和权限拦截的挂载方式。
6. 发布前校验：发布 APK 或资源前检查 SHA-256、文件大小、Manifest 签名、公钥配置、目标渠道和灰度比例。
7. 操作体验优化：后台增加发布前确认摘要、危险操作二次确认、失败原因聚合和一键复制 smoke 命令。
8. 文档中心 MVP：整理文档目录、补 front matter、上线静态文档站，并在后台增加只读入口。

### 中期演进点

一期稳定后，可以把发布中心从 App 专用能力扩成更通用的发布平台：

- 多制品类型：Web dist、服务端二进制、Docker image、配置包和文档站部署记录。
- 多项目空间：项目、应用、环境、渠道、负责人和权限隔离。
- 发布审批：支持生产发布审批、冻结窗口、强制升级二次确认。
- 自动质量门禁：基于失败率、失败次数、设备范围和时间窗口自动暂停或建议回滚。
- 外部通知：质量告警、发布完成、回滚、自动暂停推送到 IM、邮件或 Webhook。
- Cloudflare 扩展：Pages 托管后台和文档站，Workers 做 Webhook 预处理、下载鉴权和 Manifest 边缘缓存，R2 承载大文件。
- 文档发布联动：发布详情关联版本说明、升级指南、API 变更和回滚 Runbook。

### 风险和注意事项

- 生产环境不要在未配置鉴权时暴露 Admin API 或 CI API。
- 增量资源继续禁止 Dex、JAR、动态库、脚本和其他可执行代码。
- Manifest 私钥只允许存在于受保护 CI 或服务端环境，客户端只内置公钥。
- 已登记制品应保持不可变；重新打包必须产生新的 build、artifact 或 resource version。
- 质量告警在自动回滚前需要先通过真机和业务规则验证，避免误触发影响正常发布。
- 系统管理当前已具备管理数据持久化接口，但还没有把用户角色权限真正用于 API 权限拦截；后续不能只做前端隐藏菜单，必须补服务端 RBAC middleware。

`appreleases.Handler` 已提供标准路由挂载：

```go
r.Mount("/", handler.Routes())
```

`Routes()` 覆盖后台、CI、Webhook 和客户端更新检查端点，并同时保留 `/api/v1/app/...` 与轻量方案中的 `/api/v1/apps/...` 兼容入口。

完整服务端可以按端点分组挂认证中间件。CI Token 的轻量接入示例：

```go
r.Mount("/", handler.RoutesWithOptions(appreleases.RouteOptions{
  CIMiddleware: []func(http.Handler) http.Handler{
    appreleases.BearerTokenMiddleware(os.Getenv("GAME_HELPER_CI_TOKEN")),
  },
}))
```

`BearerTokenMiddleware` 仅在传入非空 token 时启用；后台登录鉴权、Webhook 签名校验也可以通过 `AdminMiddleware`、`WebhookMiddleware` 分组接入。

当前子树已提供最小服务入口：

```bash
ADDR=:18080 \
FILE_ROOT=/srv/files \
DATABASE_URL="$DATABASE_URL" \
ADMIN_TOKEN="$GAME_HELPER_ADMIN_TOKEN" \
CI_TOKEN="$GAME_HELPER_CI_TOKEN" \
go run -buildvcs=false ./cmd/server
```

如果不配置 `DATABASE_URL`，服务仍可启动并提供健康检查和配置兜底的客户端检查接口；管理后台写操作和 CI 上传需要 PostgreSQL store。

后台 Web 已拆入 `web/`，可独立构建：

```bash
cd web && npm run build
```

构建产物位于 `web/dist`。`cmd/server` 默认通过 `WEB_DIST=web/dist` 托管后台页面，`GET /` 返回管理控制台；API 路径 `/api/*` 和 `/admin/api/*` 不会被 SPA fallback 抢占。

当前前端已经从单一发版中心页面扩展为基础后台系统壳：

- 登录页：写入 `release-center-admin-token`，用于真实后台 API Bearer 鉴权。
- 首页：展示系统模块、用户/角色/权限/字典统计和快捷入口，统计来自系统管理 overview API。
- 系统管理：包含用户管理、角色管理、权限管理、数据字典和菜单编辑页面，新增、启停、显示隐藏操作通过后端 API 执行并刷新 React Query 缓存；菜单 visible 状态会影响侧栏导航和首页模块入口。
- 业务模块：App 发版中心作为菜单项接入，继续覆盖 App、构建、发布、资源、设备、统计、事件和审计。

系统管理页面已不再依赖前端本地 seed 状态。当前后端提供 `SystemManagementStore`，PostgreSQL 环境读写 `system_users`、`system_roles`、`system_permissions`、`system_dictionaries`、`system_menus`；无数据库 store 时使用进程内 Demo 状态，方便本地调试新增、启停和菜单显隐。系统管理写操作会调用 `InsertAudit` 记录操作审计。下一步需要把登录身份、角色权限和菜单可见性接入服务端 RBAC middleware。后台审计页会突出显示 `resource.auto_pause` 自动保护记录，资源列表会展示 paused 资源的 `paused_at` 暂停时间。

## 文档中心推进计划

文档中心建议作为发布中心的配套模块推进，先解决“文档可发布、可访问、可追溯”，再扩展到“多项目、多版本、权限、搜索和发布联动”。一期不自研复杂 CMS，优先复用 Git 仓库、静态站点构建、Cloudflare Pages 或现有 Go 静态托管能力。

### 建设目标

文档中心需要覆盖以下核心场景：

- 项目文档：产品说明、接入指南、运维手册、故障处理、API 文档。
- 发布文档：版本说明、变更记录、灰度公告、回滚说明、客户端升级说明。
- 操作文档：发布中心使用手册、CLI 使用说明、资源增量规范、应急预案。
- 审计追溯：文档版本、Git commit、发布人、发布时间、发布渠道可查。
- 对外访问：内部后台查看，必要时发布到公开文档站或受控访问入口。

边界约束：

- 文档源码仍以 Git 为主，避免后台直接编辑成为唯一来源。
- 文档发布必须生成不可变版本记录，不能覆盖历史发布。
- 公开文档和内部文档要能按项目、环境或可见范围隔离。
- API 密钥、账号密码、私有地址、生产密钥不得进入公开文档构建产物。

### 阶段 0：资料盘点和目录规范

目标是在不改系统的前提下，把已有文档整理成稳定结构。

交付物：

- 统一 `docs/` 目录分层：
  - `release-center/`：发布中心产品、部署、运维和应急文档。
  - `app-release/`：App 整包、增量资源、客户端接入和真机 smoke 文档。
  - `api/`：Admin API、CI API、客户端 API 和 Webhook 文档。
  - `runbooks/`：上线、回滚、数据库迁移、事故处理手册。
- 文档 front matter 约定：`title`、`owner`、`status`、`visibility`、`updated_at`、`related_release`。
- 文档检查清单：无敏感信息、命令可复跑、端点与代码一致、验收步骤明确。

验收标准：

- 现有发布中心文档能按主题归档。
- 每份核心文档都有负责人、状态和最后更新时间。
- 发版中心的部署、迁移、APK smoke、资源 smoke、回滚预案能在 30 分钟内被定位。

### 阶段 1：静态文档站 MVP

目标是先上线一个可访问、可搜索、可版本化的静态文档站。

推荐实现：

- 使用 VitePress、Docusaurus 或 MkDocs 生成静态站点。
- CI 执行文档构建和链接检查。
- 产物优先部署到 Cloudflare Pages；本地或内网环境可由 Go server 托管 `docs-dist`。
- 发布中心只登记文档站构建记录和访问 URL，不承载复杂编辑逻辑。

最小数据记录：

```text
doc_sites
doc_builds
doc_deployments
audit_events
```

关键字段：

```text
site_key
repo_url
git_ref
git_commit
build_status
deploy_url
visibility
created_by
created_at
```

验收标准：

- 提交文档后 CI 能生成静态站点。
- 管理后台能看到最近一次文档构建、commit、部署地址和状态。
- 文档站至少支持全文搜索、左侧导航、版本说明入口和 404 页面。
- `docs/app-release-center-implementation.md`、`docs/2026-06-02-app-release-center-handoff.md` 等核心文档能在站点中访问。

### 阶段 2：接入发布中心后台

目标是让文档中心成为后台的一个正式菜单，而不是散落在 Git 或 Pages 控制台里。

后台页面：

- 文档站列表：站点名称、项目、可见范围、最新 commit、部署状态、访问地址。
- 构建记录：触发来源、分支、commit、构建日志摘要、失败原因。
- 发布记录：环境、URL、发布时间、操作人、回滚入口。
- 文档关联：把版本说明、Runbook、API 文档关联到 App 发布或资源版本。

后端 API：

```text
GET  /admin/api/doc-sites
POST /admin/api/doc-sites
GET  /admin/api/doc-sites/{site_id}/builds
POST /admin/api/doc-sites/{site_id}/builds
POST /admin/api/doc-builds/{build_id}/deploy
POST /admin/api/doc-deployments/{deployment_id}/rollback
```

CI 接入：

```text
POST /api/v1/ci/doc-builds
POST /api/v1/ci/doc-deployments
```

验收标准：

- 外部 CI 能把文档构建和部署结果回传发布中心。
- 后台能按项目查看文档部署历史和当前线上版本。
- App 发布详情页能挂载对应发布说明、升级指南和回滚 Runbook。
- 文档发布、回滚、可见范围变更都有审计日志。

### 阶段 3：权限、质量和安全门禁

目标是把文档发布从“能上线”推进到“可控上线”。

能力项：

- 权限：公开文档、内部文档、项目私有文档分级。
- 审批：公开文档发布前支持人工确认。
- 质量：链接检查、Markdown lint、敏感词和密钥扫描。
- 预览：每次 PR/MR 或分支构建生成预览地址。
- 冻结：生产事故或发布冻结窗口内限制公开文档发布。

验收标准：

- 含疑似密钥的文档构建会失败或进入人工复核。
- 内部文档不会被部署到公开 Pages 项目。
- 文档发布记录能追溯到具体 Git commit 和操作者。
- 公开文档发布前能预览并确认。

### 阶段 4：文档和发布联动

目标是让文档成为发布流程的一部分，而不是发布之后补材料。

联动规则：

- 创建 App 发布时要求选择或生成发布说明文档。
- 强制升级版本必须关联用户升级说明和回滚预案。
- 资源版本发布必须关联资源变更说明和失败回滚说明。
- 生产发布前检查对应文档是否已构建成功。
- 回滚时自动置顶回滚公告或展示当前推荐版本说明。

验收标准：

- 后台发布详情页可以一键打开本次发布文档、API 变更、Runbook。
- 缺少必要文档时，生产发布会被阻断或要求二次确认。
- 质量告警触发后，后台能快速定位到对应版本的应急文档。

### 近期优先级

建议按以下顺序推进：

1. 整理 `docs/` 目录和 front matter 规范。
2. 选定静态文档站方案并完成本地构建。
3. 增加 CI 文档构建、链接检查和产物归档。
4. 部署到 Cloudflare Pages 或 Go 静态托管目录。
5. 在后台增加“文档中心”菜单和只读列表页。
6. 补齐 `doc_sites`、`doc_builds`、`doc_deployments` 表和 CI 回传 API。
7. 将 App 发布详情与发布说明、升级指南、回滚 Runbook 建立关联。

第一阶段可以先不做在线编辑、复杂权限和内置构建 Runner。这样能最快形成可访问文档站，同时保留后续扩展成完整文档中心的空间。

Manifest Ed25519 签名已接入资源打包和后台上传链路。服务端可通过 `MANIFEST_PRIVATE_KEY` 或 `GAME_HELPER_MANIFEST_PRIVATE_KEY` 注入私钥，生成的 Manifest 会包含 `signatureAlgorithm=ed25519` 和 `signature`。后台资源列表会展示 Manifest 签名状态。

PostgreSQL 最小 schema 已提供：

```text
internal/migrations/000001_release_center_schema.up.sql
internal/migrations/000001_release_center_schema.down.sql
```

该 migration 覆盖 apps、builds、releases、resource versions、installations、update events、webhook events 和 audit events 等一期闭环表结构。

迁移执行入口：

```bash
DATABASE_URL="$DATABASE_URL" go run -buildvcs=false ./cmd/migrate
```

`cmd/migrate` 会按文件名顺序执行 `internal/migrations/*.up.sql`，并写入 `schema_migrations` 防止重复应用。

本地验证入口：

```bash
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
make build
make resource-catalog
make smoke-db
```

2026-06-02 当前落档复验已通过：

```text
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
env -u DATABASE_URL ADDR=127.0.0.1:19082 FILE_ROOT=/tmp/release-center-files WEB_DIST=web/dist ADMIN_TOKEN=dev go run -buildvcs=false ./cmd/server
curl -i http://127.0.0.1:19082/healthz
curl -H 'Authorization: Bearer dev' http://127.0.0.1:19082/admin/api/system/overview
curl -H 'Authorization: Bearer dev' -X POST http://127.0.0.1:19082/admin/api/system/menus/{menu_id}/hide
curl -H 'Authorization: Bearer dev' http://127.0.0.1:19082/admin/api/system/overview
curl -H 'Authorization: Bearer dev' -X POST http://127.0.0.1:19082/admin/api/system/menus/{menu_id}/show
```

GitHub Actions 已提供 `.github/workflows/release-center-ci.yml`，执行 test、build 和 resource catalog smoke。

本次系统管理 API smoke 结果：19082 无数据库 Demo fallback 模式下，`/healthz` 返回 204，`/admin/api/system/overview` 返回用户、角色、权限、字典、菜单数据；菜单 hide 后 action 返回 `visible=false`，再次读取 overview 仍为 `false`；菜单 show 后 action 返回 `visible=true`，再次读取 overview 仍为 `true`。Go server 同源托管 `web/dist`，`GET /` 返回构建后的后台 HTML。

浏览器自动化说明：当前机器未安装 Chromium、Playwright 或 Puppeteer，本次没有执行可视化点击验收。已完成前端 TypeScript/Vite 构建、Go server 同源 HTML 返回和系统管理 API smoke；生产或具备浏览器环境后仍需补一次页面点击 smoke。

后台 overview API 已返回 `quality_metrics`、`quality_policy` 和 `quality_alerts`，按 APK 与资源事件拆分成功率、失败率、失败原因 Top 列表、策略阈值、策略建议和派生质量告警。后台统计页会展示 APK 升级质量、资源激活质量、当前策略阈值，并给出继续观察、建议暂停资源或建议回滚 APK 的提示；达到阈值时会展示质量告警卡片。

质量策略阈值可通过环境变量覆盖：

```text
QUALITY_RESOURCE_ACTIVATION_FAILED_COUNT  默认 3
QUALITY_RESOURCE_FAILURE_RATE             默认 5
QUALITY_APK_CHECKSUM_FAILED_COUNT         默认 3
QUALITY_APK_INSTALL_FAILED_COUNT          默认 10
QUALITY_APK_INSTALL_FAILURE_RATE          默认 10
```

本机 PostgreSQL 真实 DB smoke 已验证：

```text
cmd/migrate applied 000001_release_center_schema
releasectl artifact-upload 登记 APK URL
POST /admin/api/app-releases 创建发布
POST /admin/api/app-releases/{id}/publish 发布成功
POST /api/v1/app/update-check 返回 version_code=100
releasectl resource-pack 生成 ZIP/Manifest/bundle
releasectl resource-upload -publish 上传资源
POST /api/v1/app/resource-check 返回 latestResourceVersion=20260602.1
POST /api/v1/app/update-event activation_failed
GET /admin/api/app-releases 显示资源 paused，audit_logs 含 resource.auto_pause
```

可复跑入口：

```bash
make smoke-db
```

默认使用 `game_helper_codex.release_center_mvp_smoke` 临时 schema，结束时清理。可用 `SMOKE_DB_NAME`、`SMOKE_DB_SCHEMA`、`SMOKE_ADDR`、`SMOKE_DATABASE_URL` 覆盖。

## 全量 APK

以下内容只能走整包 APK 发版：

- `classes.dex`、`plugin.dex`
- `*.jar`
- `*.so`、`*.dll`、`*.dylib`
- `*.apk`、`*.aab`
- shell、PowerShell、bat/cmd、JS/MJS/CJS 等可执行脚本
- 原生代码、字节码、未经过安全审计的执行代码

后台 API 已支持 APK 上传、构建登记、发布创建、发布/暂停/召回/回滚和下载：

- `POST /admin/api/apps`
- `POST /admin/api/apps/{app_id}/enable`
- `POST /admin/api/apps/{app_id}/disable`
- `POST /admin/api/app-releases/build`
- `POST /admin/api/app-releases`
- `POST /admin/api/app-releases/{release_id}/publish`
- `GET /api/v1/app/releases/check`

CI 或本地脚本可以用 `releasectl` 上传 APK/AAB/ZIP 制品，底层走一期轻量 API `POST /api/v1/ci/artifacts`：

```bash
go run ./cmd/releasectl artifact-upload \
  -base-url "$GAME_HELPER_RELEASE_BASE_URL" \
  -token "$GAME_HELPER_ADMIN_TOKEN" \
  -file app-release.apk \
  -git-ref main \
  -git-commit "$GITHUB_SHA" \
  -version-name 0.2.0 \
  -version-code 200 \
  -build-number 37 \
  -channel dev \
  -artifact-type apk
```

如果制品已经上传到外部对象存储或 CI 下载地址，也可以只登记 URL：

```bash
go run ./cmd/releasectl artifact-upload \
  -base-url "$GAME_HELPER_RELEASE_BASE_URL" \
  -token "$GAME_HELPER_ADMIN_TOKEN" \
  -artifact-url "https://ci.example.com/app-release.apk" \
  -file-name app-release.apk \
  -size-bytes 42800000 \
  -sha256 "$APK_SHA256" \
  -git-ref main \
  -version-name 0.2.0 \
  -version-code 200
```

## 增量资源

增量只允许更新数据和静态资源。当前可自动打包的 `package_key` 如下：

| package_key | 内容 |
|---|---|
| `scenes-core` | 基础场景配置 |
| `scenes-events` | 活动脚本配置数据 |
| `templates-common` | 通用识图模板 |
| `templates-bear` | 打熊识图模板 |
| `ocr-dictionary` | OCR 关键词库 |
| `ui-assets` | 启动页和 UI 静态资源 |
| `feature-flags` | 功能开关 |

可以用命令查看当前白名单和允许扩展名：

```bash
make resource-catalog
```

后台手工上传资源 ZIP 和 CLI 自动打包产物都走同一套校验：不支持的 `package_key`、空 ZIP、隐藏文件、路径穿越、符号链接、Dex/JAR/动态库/脚本都会被拒绝。

`activation_failed` 资源事件会自动触发对应 `resource_version` 挂起：

- 仅匹配 `app_id + resource_version`
- 只暂停 `released / rolling_out / testing` 状态
- 写入 `paused_at`
- 记录审计 `resource.auto_pause`

这项行为的目标是让“资源激活失败”不继续影响后续在线用户，避免继续放量扩散。

资源版本操作已与 App 发布动作对齐：

- `publish` / `rollback`：切回 `released`
- `pause`：切到 `paused`
- `recall` / `unpublish`：切到 `recalled`

更新事件也会反写设备安装记录：

- `install_success` 更新 `installed_version` / `installed_code` / `last_upgrade_status`。
- `install_failed`、`download_failed` 写入 `last_upgrade_status` 和 `last_error`。
- `activation_success` 更新 `resource_version` / `last_upgrade_status`。
- `activation_failed` 写入 `last_upgrade_status` 和 `last_error`，并继续触发资源自动暂停保护。

客户端心跳已支持：

- `POST /api/v1/app/heartbeat`
- `POST /api/v1/apps/heartbeat`

心跳会 upsert `app_installations`，刷新设备当前 APK 版本、`versionCode`、`buildNumber`、`resourceVersion`、系统版本、机型和 `last_seen_at`，用于后台查看版本分布。

任务执行前校验已支持：

- `POST /api/v1/app/task-preflight`
- `POST /api/v1/apps/task-preflight`

Preflight 会同时执行 App 整包检查和资源增量检查，返回 `can_execute`、`block_reason`、`app_update` 和 `resource_update`。当 App 强制升级、当前版本不可用，或资源更新要求阻断任务时，`can_execute=false`。

## 自动打包

推荐资源目录结构：

```text
resources/
  templates-bear/
    metadata.json
    button.png
  ocr-dictionary/
    keywords.txt
```

本地打包：

```bash
make resource-pack \
  RESOURCE_ROOT=resources \
  RESOURCE_VERSION=20260602.1 \
  RESOURCE_TITLE="Runtime resource update"
```

或直接使用 CLI：

```bash
go run ./cmd/releasectl resource-pack \
  -version 20260602.1 \
  -channel dev \
  -title "Runtime resource update" \
  -root resources \
  -out dist/app-resources \
  -manifest-private-key "$GAME_HELPER_MANIFEST_PRIVATE_KEY"
```

导出给客户端内置的公钥：

```bash
go run ./cmd/releasectl manifest-public-key \
  -private-key "$GAME_HELPER_MANIFEST_PRIVATE_KEY"
```

本地验签：

```bash
go run ./cmd/releasectl manifest-verify \
  -manifest dist/app-resources/manifest-20260602.1.json \
  -public-key "$GAME_HELPER_MANIFEST_PUBLIC_KEY"
```

签名 payload 为移除 `signatureAlgorithm` 和 `signature` 后的 Manifest 紧凑 JSON，避免缩进和换行影响验签。

输出内容：

- `metadata-{resourceVersion}.json`：可上传给后台的资源版本元数据
- `manifest-{resourceVersion}.json`：客户端增量更新 manifest
- `{package_key}-{resourceVersion}.zip`：按模块自动打包的资源 ZIP
- `bundle-{resourceVersion}.json`：CLI 上传部署用索引文件

## 上传部署

上传资源版本到后台：

```bash
go run ./cmd/releasectl resource-upload \
  -base-url http://127.0.0.1:18080 \
  -bundle dist/app-resources/bundle-20260602.1.json \
  -username admin \
  -password "$GAME_HELPER_ADMIN_PASSWORD"
```

上传后直接发布：

```bash
go run ./cmd/releasectl resource-upload \
  -base-url "$GAME_HELPER_RELEASE_BASE_URL" \
  -bundle dist/app-resources/bundle-20260602.1.json \
  -token "$GAME_HELPER_ADMIN_TOKEN" \
  -publish
```

CI 推荐先执行 `resource-pack` 生成产物；只有受保护环境才配置 `GAME_HELPER_ADMIN_TOKEN` 并执行 `resource-upload -publish`。
