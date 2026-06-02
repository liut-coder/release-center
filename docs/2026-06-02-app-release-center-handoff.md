# App 发版中心交接

更新时间：2026-06-02

## 1. 结论

当前 `/root/release-center` 已按 `docs/app-release-center-spec.md` 和 `docs/lightweight-release-center-plan.md` 推进到前后端主闭环状态：

- 整包 APK 和启动页资源增量更新已明确分流。
- 显式 App 创建、启用、停用的 service/store/handler 已补齐。
- `appreleases.Handler.Routes()` / `RoutesWithOptions()` 已补齐一期标准路由注册和分组认证挂载点。
- `BearerTokenMiddleware` 已补齐轻量 CI Token 接入能力。
- 增量资源白名单、禁止项、ZIP 校验、自动打包、Manifest 生成已落到代码。
- `releasectl` 已提供本地和 CI 可调用入口。
- `releasectl artifact-upload` 已补齐 APK/AAB/ZIP/Web dist 制品上传和 URL 登记入口。
- 客户端 heartbeat 已补齐，会刷新设备版本分布和资源版本。
- 任务执行前 preflight 已补齐，会合并 App 整包检查和资源增量检查。
- 更新事件已反写 `app_installations` 的版本、资源版本、最后升级状态和失败原因。
- 最小 `cmd/server` 已补齐，可直接启动 release center API、`/healthz`、`/readyz`。
- 最小 PostgreSQL schema migration 已补齐，覆盖一期闭环表和约束。
- 轻量 `cmd/migrate` 已补齐，可按顺序执行 `internal/migrations/*.up.sql` 并记录 `schema_migrations`。
- CI 已覆盖 `releasectl` 构建、资源白名单输出和临时资源包打包 smoke。
- 资源激活失败事件 `activation_failed` 已接入自动暂停发布保护。
- 后台 Web 已从单一发版中心页面扩展为管理控制台脚手架，支持登录入口、首页、系统管理导航、用户/角色/权限/数据字典/菜单编辑页面，并将 App 发版中心作为业务模块接入。
- App 发版中心业务页支持 App 管理、构建、发布、资源、设备、统计、事件和审计。
- 后台 Web 已突出展示 `resource.auto_pause` 自动暂停保护记录，并在资源列表展示 `paused_at` 暂停时间。
- 后台 overview API 已返回 `quality_metrics` 和 `quality_policy`，统计页已展示 APK/资源成功率、失败率、失败原因、可配置阈值和策略建议。
- 后台 overview API 已返回派生 `quality_alerts`，统计页已展示达到阈值后的质量告警卡片。
- Manifest Ed25519 签名已接入资源打包和后台上传链路，`releasectl` 已提供公钥导出和本地验签命令。
- Go server 已托管 `WEB_DIST=web/dist`，`GET /` 返回后台页面，`/api/*` 和 `/admin/api/*` 保持 API 行为。
- 系统管理已新增用户、角色、权限、数据字典、菜单 PostgreSQL migration、Admin API 和前端 API 客户端。
- 后台系统管理页面已从前端本地示例状态切换到后端 overview/create/enable/disable/show/hide 接口，操作后通过 React Query 刷新；菜单 visible 状态已驱动侧栏导航和首页模块入口。
- 后台 Admin API 已接入服务端 RBAC permission middleware，按用户账号、角色权限和路由权限码拦截；系统 overview 会按当前账号过滤菜单和系统管理数据。
- App 发版中心构建表单的 `apiBaseUrl` 已改为优先读取 `VITE_API_BASE_URL`，未配置时使用当前页面 origin，避免本地或生产同源部署时默认写入旧公网 IP。
- 无数据库模式已从只读兜底升级为进程内可变 Demo store，覆盖 App、构建、发布、资源版本、设备、事件、审计和系统管理数据；本地浏览器 smoke 可用 demo 项目真实提交构建创建、发布草稿创建和资源 ZIP 上传创建。
- App 发布页面移动端长版本号、长文件名、select 和操作按钮已补齐响应式约束，提交真实 demo 数据后仍通过页面横向溢出检查。
- 本机 PostgreSQL 临时 schema 已完成真实 DB smoke，覆盖迁移、APK URL 登记、发布、update-check、资源上传发布、resource-check、activation_failed 自动暂停和审计。

当前已落地 SHA-256、文件大小、ZIP 内容校验、Manifest Ed25519 签名和禁止执行代码增量发布。Android 客户端内置公钥验签仍需在客户端工程和真机 smoke 中确认。

## 1.0 当前项目总进度

| 方向 | 当前状态 | 仍需外部确认 |
|---|---|---|
| 后端闭环 | App / build / release / resource / event / heartbeat / preflight / audit API 已落地 | 生产 18080 使用真实 `DATABASE_URL` 执行迁移并重启 |
| 前端闭环 | `web/` 管理台已具备登录、首页、系统管理页面、用户/角色/权限/数据字典/菜单后端数据接入、菜单/RBAC 过滤结果驱动导航和 App 发版中心业务模块；本地 Playwright 已验证桌面/移动真实表单提交和无横向溢出 | 生产静态资源浏览器 smoke |
| CLI / CI | `releasectl` 覆盖 APK URL/文件登记、资源打包上传、公钥导出、Manifest 验签；workflow 覆盖基础 smoke | 受保护 CI 环境配置真实 token 后跑发布链路 |
| APK 更新 | 发布、灰度、下载、update-check、SHA-256 信息输出已闭环 | Android 真机下载安装、校验和系统安装器 smoke |
| 资源增量 | 白名单、ZIP 校验、Manifest、下载、resource-check、自动暂停已闭环 | Android 真机下载、Ed25519 验签、解压激活、失败回滚 smoke |
| 质量策略 | `quality_metrics`、`quality_policy`、`quality_alerts` API/Web 已闭环 | 外部通知和自动执行策略尚未接入 |

## 1.1 一期闭环核对

按 `docs/lightweight-release-center-plan.md` 的 MVP 范围，当前子树已有或已推进的证据：

| MVP 项 | 当前证据 | 状态 |
|---|---|---|
| 应用管理 | `CreateApp`、`AppAction(enable/disable)`、`AdminOverview`、Web App 管理表单 | 已验证 |
| 构建记录登记 | `Service.CreateBuild`、`Handler.CreateBuild`、`Handler.CreateCIBuild` | 已有 |
| 制品上传或 URL 登记 | `Handler.CreateCIArtifact`、`releasectl artifact-upload`、真实 DB smoke | 已验证 |
| APK 发布 | `CreateRelease`、`ReleaseAction(publish/pause/recall/rollback)`、下载接口、update-check smoke | 已验证 |
| 资源增量发布 | `CreateResourceVersion`、资源包校验、Manifest、下载接口、resource-check smoke | 已验证 |
| 发布说明编辑 | `UpdateReleaseNotes`、`UpdateResourceNotes`、Web Markdown 编辑入口 | 已验证 |
| 发布、暂停、召回、回滚 | App 和资源均支持 publish/pause/recall/rollback/unpublish | 已验证 |
| 百分比灰度 | `rollout_percentage`、PostgreSQL hash 命中逻辑 | 已有 |
| 客户端检查更新 | `UpdateCheck` / `ResourceCheck` | 已验证 |
| 任务执行前兼容性校验 | `TaskPreflight` 合并 App 和资源检查，返回 `can_execute` | 已验证 |
| 资源 Manifest 和下载 | `DownloadResourceManifest` / `DownloadResourcePackage` | 已验证 |
| 更新事件上报 | `UpdateEvent`、APK/资源事件落库、`app_installations` 状态反写 | 已验证 |
| 设备版本分布 | `Heartbeat` upsert `app_installations` 当前 APK/资源版本和 `last_seen_at`、Web 设备页 | 已验证 |
| 质量指标和告警 | `quality_metrics` / `quality_policy` / `quality_alerts`、Web 统计页 | 已验证 |
| 操作审计 | `InsertAudit` 覆盖 release/resource/webhook 操作，真实 smoke 覆盖 `resource.auto_pause` | 已验证 |
| GitHub/Gitea Webhook 记录 | `Handler.Webhook(provider)`、`SaveWebhookEvent` | 已有 |
| GitHub/Gitea Actions 上传制品 | `/api/v1/ci/artifacts` + `releasectl artifact-upload` + `CIMiddleware` | 已验证 |

当前目录已包含 `appreleases` 标准路由注册、迁移、服务端 main、Go module、Makefile、CI workflow 和独立后台 Web。后台 Web 已具备基础管理台壳，系统管理页面已接入后端数据；用户、角色、权限、数据字典、菜单已有表结构和 API，Admin API 已按 `release:*`、`system:*` 等权限码接入 RBAC 拦截。无数据库模式会启用 `NewDemoStore()`，便于用 demo 项目验证系统管理和 App 发版中心写入流程。后续需要在真实 PostgreSQL 环境验收持久化读写、生产账号映射和浏览器 smoke。发版闭环已在本机 PostgreSQL 临时 schema 验证；生产 18080 仍需用真实 APK、真实资源包和 Android 客户端做真机 smoke。

CI Token 推荐在完整服务端挂载时接入：

```go
r.Mount("/", handler.RoutesWithOptions(appreleases.RouteOptions{
  CIMiddleware: []func(http.Handler) http.Handler{
    appreleases.BearerTokenMiddleware(os.Getenv("GAME_HELPER_CI_TOKEN")),
  },
}))
```

后台 Admin API 当前通过 `ADMIN_TOKEN` / `ADMIN_TOKEN_ACCOUNTS` 映射到系统账号并执行 RBAC：

```bash
ADMIN_TOKEN="$GAME_HELPER_ADMIN_TOKEN"                       # 默认映射 system.admin
ADMIN_TOKEN_ACCOUNTS="token1:system.admin,token2:release.admin"
```

`system.admin` 属于 `system_admin`，可访问系统管理和发布操作；`release.admin` 属于 `release_admin`，可访问发布中心，但系统管理写接口会返回 403。Webhook 签名校验仍可通过 `WebhookMiddleware` 单独接入，避免和后台 RBAC 混在一起。

## 2. 关键代码

资源增量规则和自动打包：

```text
internal/modules/appreleases/resource_packager.go
internal/modules/appreleases/resource_packager_test.go
internal/modules/appreleases/routes.go
internal/modules/appreleases/routes_test.go
```

后台上传 ZIP 校验和 Manifest 生成：

```text
internal/modules/appreleases/admin_service.go
```

资源激活失败自动暂停：

```text
internal/modules/appreleases/postgres.go
internal/modules/appreleases/admin_postgres.go
internal/migrations/000001_release_center_schema.up.sql
internal/migrations/000001_release_center_schema.down.sql
```

CLI 和 CI：

```text
cmd/server/main.go
cmd/migrate/main.go
cmd/releasectl/main.go
web/
Makefile
.github/workflows/release-center-ci.yml
```

实现说明：

```text
docs/app-release-center-implementation.md
docs/phase1-closure-audit.md
```

## 3. 全量和增量边界

只能整包 APK 发布：

```text
classes.dex
plugin.dex
*.jar
*.so / *.dll / *.dylib
*.apk / *.aab
shell / PowerShell / bat / cmd / JS / MJS / CJS 脚本
原生代码、字节码、未审计执行代码
```

可以增量发布的 `package_key`：

```text
scenes-core        基础场景配置
scenes-events      活动脚本配置数据
templates-common   通用识图模板
templates-bear     打熊识图模板
ocr-dictionary     OCR 关键词库
ui-assets          启动页和 UI 静态资源
feature-flags      功能开关
```

后台手工上传和 CLI 自动打包都复用同一套校验：

```text
不支持的 package_key 拒绝
空 ZIP 拒绝
隐藏文件拒绝
路径穿越拒绝
符号链接拒绝
Dex/JAR/动态库/APK/AAB/脚本拒绝
```

## 4. CLI 使用

### APK / 构建制品上传

当前 `releasectl artifact-upload` 用于 GitHub Actions、Gitea Actions 或本地脚本登记 APK/AAB/ZIP/Web dist 制品，底层走：

```text
POST /api/v1/ci/artifacts
```

上传本地 APK：

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

登记已有外部 URL：

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

### 资源包

查看白名单：

```bash
make resource-catalog
```

推荐资源目录结构：

```text
resources/
  templates-bear/
    metadata.json
    button.png
  ocr-dictionary/
    keywords.txt
```

打包：

```bash
make resource-pack \
  RESOURCE_ROOT=resources \
  RESOURCE_VERSION=20260602.1 \
  RESOURCE_TITLE="Runtime resource update"
```

或直接调用：

```bash
go run ./cmd/releasectl resource-pack \
  -version 20260602.1 \
  -channel dev \
  -title "Runtime resource update" \
  -root resources \
  -out dist/app-resources \
  -manifest-private-key "$GAME_HELPER_MANIFEST_PRIVATE_KEY"
```

产物：

```text
metadata-{resourceVersion}.json
manifest-{resourceVersion}.json
{package_key}-{resourceVersion}.zip
bundle-{resourceVersion}.json
```

导出客户端内置公钥并本地验签：

```bash
go run ./cmd/releasectl manifest-public-key \
  -private-key "$GAME_HELPER_MANIFEST_PRIVATE_KEY"

go run ./cmd/releasectl manifest-verify \
  -manifest dist/app-resources/manifest-20260602.1.json \
  -public-key "$GAME_HELPER_MANIFEST_PUBLIC_KEY"
```

上传：

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

## 5. API 行为

App 整包更新：

```text
POST /admin/api/apps
POST /admin/api/apps/{app_id}/enable
POST /admin/api/apps/{app_id}/disable
POST /admin/api/app-releases/build
POST /admin/api/app-releases
POST /admin/api/app-releases/{release_id}/publish
GET  /api/v1/app/releases/check
```

资源增量：

```text
POST /admin/api/app-resources
POST /admin/api/app-resources/{resource_id}/publish
POST /admin/api/app-resources/{resource_id}/pause
POST /admin/api/app-resources/{resource_id}/recall
POST /api/v1/app/resources/check
GET  /api/v1/app/resources/{resource_id}/manifest
GET  /api/v1/app/resources/packages/{package_id}/download
```

资源事件：

```text
POST /api/v1/app/heartbeat
POST /api/v1/app/task-preflight
POST /api/v1/app/update-event
```

`POST /api/v1/apps/heartbeat`、`POST /api/v1/apps/task-preflight` 也作为轻量方案兼容路径保留。

当客户端上报：

```json
{
  "eventType": "activation_failed",
  "toVersion": "20260602.1"
}
```

后台会自动暂停匹配的资源版本：

```text
匹配 app_id + resource_version
只暂停 released / rolling_out / testing
写入 paused_at
写入 audit_events: resource.auto_pause
```

更新事件同时会同步设备状态：

```text
install_success      更新 installed_version / installed_code / last_upgrade_status
install_failed       更新 last_upgrade_status / last_error
download_failed      更新 last_upgrade_status / last_error
activation_success   更新 resource_version / last_upgrade_status
activation_failed    更新 last_upgrade_status / last_error，并触发 resource.auto_pause
```

## 6. 上线步骤

1. 确认代码已合入并构建新服务端。
2. 执行数据库迁移：`DATABASE_URL="$DATABASE_URL" go run -buildvcs=false ./cmd/migrate`。
3. 重启 18080 服务。
4. 确认 `/readyz` 正常。
5. 用 `resource-pack` 打一个测试资源包。
6. 用 `resource-upload` 上传到后台。
7. 在后台或 CLI 发布资源版本。
8. 用 `POST /api/v1/app/resources/check` 验证客户端可拿到 Manifest 和 packages。
9. 上报一次测试 `activation_failed`，确认资源版本自动变为 `paused`，审计出现 `resource.auto_pause`。

## 7. 验证结果

当前验证入口：

```bash
gofmt -w ...
go mod tidy &&
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
make test
make build
make resource-catalog
make smoke-db
ADDR=127.0.0.1:18082 FILE_ROOT=/tmp/release-center-files WEB_DIST=web/dist go run -buildvcs=false ./cmd/server
curl -i -s http://127.0.0.1:18082/readyz
curl -i -s http://127.0.0.1:18082/
curl -i -s http://127.0.0.1:18082/api/v1/app/update-check \
  -X POST -H 'Content-Type: application/json' \
  -d '{"deviceId":"device-1","versionCode":1}'
```

2026-06-02 当前落档复验已通过：

```text
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
```

2026-06-02 最新修复复验：

```text
App 构建表单 apiBaseUrl 默认值已改为 VITE_API_BASE_URL 或当前页面 origin
Admin API RBAC 已接入路由权限码，system overview 会按账号过滤菜单和系统管理数据
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
ADDR=127.0.0.1:18083 ADMIN_TOKEN=admin-token ADMIN_TOKEN_ACCOUNTS='release-token:release.admin' WEB_DIST=web/dist go run -buildvcs=false ./cmd/server
cd web && npm run smoke:browser
```

本地 RBAC server smoke 结果：`release.admin` token 可读取发布中心和初始化后台 overview，overview 只返回首页和发布中心菜单，不返回系统管理菜单和系统管理数据；`release.admin` 写 `/admin/api/system/users` 返回 403；`system.admin` 写 `/admin/api/system/users` 返回 200。

本地浏览器 smoke 结果：已安装 Playwright Chromium，`npm run smoke:browser` 通过 4 个用例，覆盖桌面和移动视口、`release.admin` / `system.admin` 登录、RBAC 菜单裁剪、系统管理页、App 发版中心概览 / 构建记录 / App 发布 / 资源增量 / 设备版本 / 升级统计 / 升级事件 / 操作审计、构建创建、发布草稿创建、合法资源 ZIP 上传创建、控制台错误、页面错误、Admin/API 5xx 监听和页面横向溢出检查。生产环境仍需对真实域名、真实 PostgreSQL 和生产 token 映射复跑。

结果：当前 `/root/release-center` Go module 验证通过：

```text
cmd/releasectl                         编译通过
internal/modules/appreleases            测试通过
  - 包含 BearerTokenMiddleware 空 token 放行、错误 token 拒绝、正确 token 放行
  - 包含 Routes 客户端 update-check / heartbeat / task-preflight smoke
  - 包含 RoutesWithOptions 对 CI endpoint 的 token 保护 smoke
  - 包含 RoutesWithOptions 对 Admin API 的 RBAC 拦截，覆盖 release.admin 可读发布、不可写系统管理、system.admin 可写系统管理
  - 包含 system overview 按账号过滤菜单和系统管理数据，覆盖 release.admin 不展示系统管理菜单
  - 包含无数据库 Demo store 写入闭环，覆盖构建创建、发布草稿创建、资源版本创建和 RBAC 拦截
  - 包含安装/资源生命周期事件状态映射
  - 包含 heartbeat snake_case 字段兼容和 deviceId 必填校验
  - 包含 task-preflight 当前版本放行、低于最低支持版本阻断
  - 包含资源 recall 动作切到 recalled 且不触发 publish
internal/platform/httpx                 测试通过
internal/platform/storage               测试通过
cmd/server                              编译通过
cmd/releasectl                          编译通过
cmd/migrate                             编译通过
web                                    `npm run build` 通过，dist 由 Go server 托管
web browser smoke                      `npm run smoke:browser` 通过，Chromium desktop/mobile 共 4 个用例，含 demo 构建/发布/资源表单提交
server smoke                            /readyz 204，/ 200，/admin/api/app-releases 200，update-check 200
real DB smoke                           migration、artifact-upload、publish、update-check、resource-upload、resource-check、activation_failed auto_pause 通过
scripts/smoke_release_center_db.sh       已固化真实 DB smoke，`make smoke-db` 通过
migration files                         覆盖当前 PostgresStore 使用的一期表、唯一约束和索引
Makefile targets                        test / build / resource-catalog 通过
GitHub Actions workflow                 release-center-ci 已补齐 test/build/resource-catalog smoke
```

当前 `/root/release-center` 目录已补齐 `go.mod` / `go.sum`、`Makefile`、GitHub Actions workflow、最小 `cmd/server`、`cmd/migrate`、基础 migration、进程内 Demo store 和后台 Web，可以直接运行 `make build`，也可以启动无数据库模式或带 PostgreSQL 模式做本地 smoke。生产级 Android 安装/资源激活仍需要在真实 18080 和客户端工程上确认。

注意：`cmd/migrate` 当前只执行 up migrations，不做 down 回滚；生产回滚仍建议走数据库备份和发布回滚流程。

## 8. 剩余事项

建议后续按优先级处理：

```text
P0 在生产 18080 真实环境执行 `cmd/migrate` 并重启服务。
P0 用真实 APK 和 Android 客户端跑安装升级、SHA-256 校验和系统安装器 smoke。
P0 用真实资源包和 Android 客户端跑 ZIP 下载、校验、激活、失败回滚 smoke。
P1 Android 客户端内置 Manifest 公钥并完成真机验签 smoke。
P1 在真实 PostgreSQL 上验收系统管理用户、角色、权限、数据字典、菜单持久化读写。
P1 在真实 PostgreSQL 和生产 token 映射下验收 RBAC：API 403、菜单裁剪、系统管理数据过滤和审计记录。
P1 在生产环境复跑前端浏览器 smoke，覆盖真实域名、真实 PostgreSQL、生产 token 映射和缓存刷新策略。
P2 将质量告警接入外部通知和自动执行策略。
```

## 9. 注意事项

- 不要把 APK、Dex、JAR、动态库或脚本塞进增量资源包。
- CI 默认只做打包 smoke，不在公共 CI 中发布资源版本。
- `resource-upload -publish` 只应在受保护环境使用 `GAME_HELPER_ADMIN_TOKEN`。
- 生产资源包版本号不要复用，建议按日期递增，例如 `20260602.1`、`20260602.2`。
