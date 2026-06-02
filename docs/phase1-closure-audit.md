# 一期闭环审计

更新时间：2026-06-02

本文按 `docs/lightweight-release-center-plan.md` 和 `docs/app-release-center-spec.md` 核对当前 `/root/release-center` 的一期闭环状态。

## 1. 结论

当前项目已经具备轻量发布中心一期所需的核心闭环：

```text
外部 CI / 本地脚本
  ↓
releasectl / CI API 上传构建制品或资源包
  ↓
Go Release Center API
  ↓
PostgreSQL schema + 本地文件存储
  ↓
发布、灰度、暂停、召回、回滚、审计
  ↓
客户端 update-check / resource-check / heartbeat / task-preflight
  ↓
独立后台 Web 管理台提供登录、首页、系统管理脚手架和 App 发版中心业务模块
```

后台 overview API 现已输出 `quality_metrics`、`quality_policy` 和 `quality_alerts`，统计页可查看 APK 与资源增量的成功率、失败率、失败原因、可配置阈值、策略建议和达到阈值后的质量告警。Manifest Ed25519 签名已接入资源打包和后台上传链路，`releasectl` 可导出公钥并本地验签；Android 客户端内置公钥验签仍需在客户端工程和真机 smoke 中确认。

本仓库内已通过：

```bash
go test ./...
make build
make resource-catalog
make smoke-db
```

本地 server smoke 已通过：

```text
/readyz                         204
/                                200，返回 web/dist 静态后台
/admin/api/app-releases          200，返回 JSON
/api/v1/app/update-check         200
```

本机 PostgreSQL 已用 `game_helper_codex.release_center_mvp_smoke` 临时 schema 完成真实 DB smoke：

```text
cmd/migrate                                  applied 000001_release_center_schema
POST /admin/api/apps                         200，创建 App
releasectl artifact-upload                   200，登记 APK URL 构建制品
POST /admin/api/app-releases                 200，创建发布
POST /admin/api/app-releases/{id}/publish    200，发布版本
POST /api/v1/app/update-check                200，返回 version_code=100 更新
releasectl resource-pack                     生成 ZIP / Manifest / bundle
releasectl resource-upload -publish          200，上传并发布资源
POST /api/v1/app/resource-check              200，返回 latestResourceVersion=20260602.1
POST /api/v1/app/update-event                activation_failed 200
GET /admin/api/app-releases                  资源状态 paused，审计含 resource.auto_pause
```

生产 18080 实例仍需用真实域名、真实文件目录权限和 Android 客户端执行一次真机下载安装/资源激活 smoke。

## 2. 轻量 MVP 核对

| MVP 项 | 当前证据 | 状态 |
|---|---|---|
| 应用管理 | `CreateApp`、`AppAction(enable/disable)`、`POST /admin/api/apps`、Web App 管理表单 | 本仓库完成 |
| 构建记录登记 | `CreateBuild`、`CreateCIBuild`、`POST /api/v1/ci/builds`、Web 构建页 | 本仓库完成 |
| 制品上传或 URL 登记 | `CreateCIArtifact`、`releasectl artifact-upload`、真实 DB smoke | 本仓库完成 |
| APK 发布 | `CreateRelease`、`ReleaseAction`、release 下载、update-check smoke | 本仓库完成 |
| 资源增量发布 | `CreateResourceVersion`、ZIP 校验、Manifest、资源下载、resource-check smoke | 本仓库完成 |
| 发布说明编辑 | `UpdateReleaseNotes`、`UpdateResourceNotes`、Web Markdown 编辑入口 | 本仓库完成 |
| 发布、暂停、召回、回滚 | App 和资源均支持 publish/pause/recall/rollback/unpublish | 本仓库完成 |
| 百分比灰度 | `rollout_percentage` + PostgreSQL hash 命中 + Web 灰度调整入口 | 本仓库完成 |
| 客户端检查更新 | update-check、resource-check、task-preflight | 本仓库完成 |
| 资源 Manifest 和下载 | manifest/package download handlers | 本仓库完成 |
| 更新事件上报 | `UpdateEvent` 落库并反写 installation 状态 | 本仓库完成 |
| 质量指标和告警 | `quality_metrics` / `quality_policy` / `quality_alerts`、Web 统计页 | 本仓库完成 |
| 操作审计 | app/release/resource/webhook audit events，真实 smoke 覆盖 `resource.auto_pause` | 本仓库完成 |
| GitHub/Gitea Webhook 记录 | `/api/v1/webhooks/github`、`/api/v1/webhooks/gitea` | 本仓库完成 |
| GitHub/Gitea Actions 上传制品 | CI artifact API、CI token middleware、workflow smoke | 本仓库完成 |
| 后台前端页面 | `web/` 独立 Vite 管理台，包含登录、首页、用户/角色/权限/数据字典/菜单编辑脚手架和 App 发版中心业务页，Go server 托管 `web/dist` | 前端脚手架完成，系统管理持久化 API 待接 |

## 3. App 规范核心目标核对

| 目标 | 当前证据 | 状态 |
|---|---|---|
| 区分开发、测试、正式发布 | `channel`、`build_type`、release/resource channel | 本仓库完成 |
| APK 整包更新 | build artifact、release、download、update-check | 本仓库完成 |
| 启动页资源增量更新 | resource package catalog、ZIP 校验、resource-check、manifest | 本仓库完成 |
| 后台编辑版本更新内容 | notes update APIs + Web Markdown 编辑入口 | 本仓库完成 |
| 普通、推荐、强制更新 | `update_level`、`ForceUpdate`、min supported code | 本仓库完成 |
| 指定用户、设备、测试组、百分比灰度 | target rules、target metadata、rollout percentage | 本仓库完成 |
| 暂停、撤回、回滚、自动保护 | release/resource actions、`activation_failed` auto pause | 本仓库完成 |
| 设备版本分布和失败原因 | heartbeat、update event installation backfill、admin overview、Web 设备页 | 本仓库完成 |
| 发布质量观察和告警 | overview quality metrics/policy/alerts、Web 统计页质量告警 | 本仓库完成 |
| 任务前版本/资源校验 | `TaskPreflight` | 本仓库完成 |
| 完整操作记录 | audit events + webhook events + update events | 本仓库完成 |

## 4. 关键交付物

```text
cmd/server/main.go
cmd/migrate/main.go
cmd/releasectl/main.go
web/*
internal/modules/appreleases/*
internal/platform/httpx/*
internal/platform/storage/*
internal/migrations/000001_release_center_schema.up.sql
internal/migrations/000001_release_center_schema.down.sql
Makefile
.github/workflows/release-center-ci.yml
go.mod
go.sum
```

## 5. 验证命令

```bash
go test ./...
make build
make resource-catalog
```

本地 server smoke：

```bash
ADDR=127.0.0.1:18082 \
FILE_ROOT=/tmp/release-center-files \
WEB_DIST=web/dist \
go run -buildvcs=false ./cmd/server

curl -i -s http://127.0.0.1:18082/readyz
curl -i -s http://127.0.0.1:18082/
curl -i -s http://127.0.0.1:18082/admin/api/app-releases

curl -i -s http://127.0.0.1:18082/api/v1/app/update-check \
  -X POST \
  -H 'Content-Type: application/json' \
  -d '{"deviceId":"device-1","versionCode":1}'
```

真实 DB smoke 使用本机 `game_helper_codex` 临时 schema：

```bash
make smoke-db
```

## 6. 外部环境验收项

当前本机 PostgreSQL 已证明服务端和后台 API 闭环。以下仍需在生产 18080 和 Android 客户端工程验证：

```text
1. DATABASE_URL 指向生产 PostgreSQL 后执行 cmd/migrate。
2. 启动生产 18080 服务并确认 /readyz、后台页面和 /admin/api/app-releases。
3. 用真实 APK 或 CI URL 跑 releasectl artifact-upload。
4. 创建并发布 App release。
5. 用 Android 客户端启动页 update-check 验证拿到 APK 更新并校验 SHA-256。
6. 用带 MANIFEST_PRIVATE_KEY 的 resource-pack/resource-upload 上传真实资源版本。
7. 发布资源版本并用 Android 客户端 resource-check 验证 manifest/packages、Ed25519 验签、ZIP 校验和本地激活。
8. 真机上报 activation_failed，确认资源版本自动 paused 且出现 resource.auto_pause 审计。
```
