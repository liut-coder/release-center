# App 发版中心实现说明

本文是 `docs/app-release-center-spec.md` 的当前落地说明，用来区分整包发版、增量资源发版和 CI/CLI 操作入口。

## 当前总进度快照

更新时间：2026-06-02

当前 `/root/release-center` 已形成 App 发版中心的前后端主闭环：

| 模块 | 当前状态 |
|---|---|
| 后端 API | App 管理、构建登记、APK 发布、资源发布、灰度、暂停、召回、回滚、下载、事件上报、heartbeat、task-preflight、审计均已落地 |
| PostgreSQL | 一期 schema migration 覆盖 apps、builds、releases、resource versions、installations、update events、webhook events、audit events |
| CLI / CI | `releasectl` 覆盖 artifact upload、resource pack/upload、Manifest 公钥导出和验签；Makefile 和 GitHub Actions 覆盖 test/build/resource catalog smoke |
| 后台 Web | `web/` 独立 Vite 管理台已具备登录入口、首页、系统管理脚手架、用户/角色/权限/数据字典/菜单编辑页面，并接入 App 发版中心业务模块；Go server 可托管 `web/dist` |
| 资源安全 | 增量资源白名单、ZIP 内容校验、禁止执行代码、SHA-256、文件大小校验和 Manifest Ed25519 签名已落地 |
| 自动保护 | `activation_failed` 会自动暂停匹配资源版本，写入 `paused_at` 并记录 `resource.auto_pause` 审计 |
| 质量观察 | overview API 输出 `quality_metrics`、`quality_policy`、`quality_alerts`；Web 统计页展示成功率、失败率、失败原因、阈值、策略建议和质量告警 |
| 本机验证 | `go test ./...`、`go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate`、`cd web && npm run build` 已通过；真实 DB smoke 覆盖迁移、发布、资源发布、检查更新和自动暂停 |
| 外部待验收 | 系统管理真实 API、生产 18080 migration/重启、真实 APK Android 安装升级 smoke、真实资源 Android 下载/验签/激活/回滚 smoke 仍需继续推进 |

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
- 首页：展示系统模块、用户/角色/权限/字典统计和快捷入口。
- 系统管理：包含用户管理、角色管理、权限管理、数据字典和菜单编辑脚手架。
- 业务模块：App 发版中心作为菜单项接入，继续覆盖 App、构建、发布、资源、设备、统计、事件和审计。

系统管理页面当前先使用前端本地状态和示例数据，已具备新增、启停、菜单排序、字典分组等交互骨架；后续需要补齐用户、角色、权限、数据字典和菜单的数据库表、后台 API、审计和权限拦截，再按相同数据模型接入真实系统管理 API。后台审计页会突出显示 `resource.auto_pause` 自动保护记录，资源列表会展示 paused 资源的 `paused_at` 暂停时间。

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
```

GitHub Actions 已提供 `.github/workflows/release-center-ci.yml`，执行 test、build 和 resource catalog smoke。

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
