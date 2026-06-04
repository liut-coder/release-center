# 构建中心可视化与多项目发布改造方案

更新时间：2026-06-05

## 1. 目标

当前发布中心已经具备 APK 发布、构建产物登记、CI Token、Webhook 记录和本机构建中心目录。下一步要从“App 发版中心”升级为轻量通用发布平台：

```text
GitHub / 手工 / 外部机器 API
  -> 构建中心
  -> 制品登记
  -> 发布计划
  -> 多环境发布
  -> Cloudflare / Webhook / SSH / Docker 等部署目标
  -> 状态回传、审计、回滚
```

一期不自研完整 CI/CD 平台，不替代 GitHub Actions、Cloudflare Wrangler、Docker、SSH 等成熟工具。发布中心负责项目、制品、发布、部署记录、审计和可视化；构建中心负责按项目 profile 调用本机或 worker 执行构建。

## 2. 产品导航

后台主导航从单一 App 发版改为通用发布平台：

```text
首页
项目中心
构建中心
制品中心
发布中心
部署中心
集成配置
观测审计
系统管理
```

### 构建中心页面

构建中心做成工作台，不做装饰型大屏。

核心区域：

- 顶部筛选：项目、环境、分支、构建 profile、状态、触发来源。
- 左侧项目栏：项目列表、项目类型、默认仓库、默认 profile。
- 中间运行列表：run id、状态、commit、分支、触发人、耗时、产物数量、上传状态。
- 右侧详情抽屉：运行参数、实时/尾部日志、产物、上传结果、失败原因、关联发布计划。
- 操作按钮：拉取代码、构建、构建并上传、全流程、重试、取消。

状态设计：

```text
queued
running
success
failed
canceled
uploading
uploaded
```

### 发布中心页面

发布中心不再只显示 APK。页面按发布计划组织：

- 发布单元：Android、Web、Docs、Worker、Server、Docker、Config。
- 环境：dev、test、staging、prod。
- 渠道：dev、beta、stable 或项目自定义渠道。
- 制品选择：从 `app_builds` 和 `app_build_artifacts` 选择不可变制品。
- 发布策略：普通发布、灰度发布、定向发布、强制更新、计划发布。
- 操作：发布、暂停、召回、回滚、归档。

APK 更新检查继续保留为 Android 发布单元的运行入口。

## 3. 数据模型

已落库的第 4 号 migration 覆盖轻量接入基础表：

```text
release_projects
code_repositories
build_profiles
build_center_runs
build_center_run_artifacts
deployment_targets
deployment_records
webhook_routes
```

这些表只保存接入配置和运行态。现有表继续作为制品和发布事实表：

```text
apps
app_builds
app_build_artifacts
app_releases
webhook_events
audit_events
```

后续需要补充的通用发布表：

```text
release_environments
release_units
release_plans
release_plan_artifacts
build_workers
worker_heartbeats
worker_tasks
integration_credentials
```

当前已落地：

- 第 7 号 migration 新增 `release_environments`、`release_units`、`release_plans`、`release_plan_artifacts`。
- 已预置 dev/test/staging/prod 环境和 release-center 的 Web/Server/Worker 发布单元。
- Admin API 支持创建发布单元、创建/查询发布计划，并支持 publish/pause/rollback 状态动作。
- Admin 前端发布中心默认进入发布计划工作台，支持项目/单元/环境矩阵、发布单元维护、发布计划创建和状态动作。
- 旧 APK 发布接口继续保留，后续再迁移成 Android 发布单元入口。

### 表职责

`release_projects`

项目空间。一个项目可以绑定多个仓库、多个应用、多个构建 profile 和多个部署目标。

`code_repositories`

GitHub、Gitea、generic Git 仓库绑定。Token、SSH Key、Webhook Secret 只存引用，不存明文。

`build_profiles`

构建配置。当前本机构建中心里，`build_center_project` 对应 `/root/build-center/config/{project}.yml`，执行入口是 `buildctl {action} {project}`。

`build_center_runs`

一次构建运行。记录触发来源、git ref、commit、版本、build number、状态、日志目录、产物目录、manifest、上传状态。

`build_center_run_artifacts`

本地构建产物。上传成功后关联 `app_build_artifacts`。

`deployment_targets`

部署目标。支持 Cloudflare Pages、Cloudflare Worker、Cloudflare R2、generic webhook、SSH、Docker、Kubernetes 预留。

`deployment_records`

一次部署记录。记录 provider deployment id、URL、状态、日志、失败原因，并关联构建运行和制品。

`webhook_routes`

把 GitHub/Gitea webhook 事件映射到构建 profile 和动作。

## 4. 后端 API

### Admin API

项目与构建：

```text
GET  /admin/api/projects
POST /admin/api/projects
GET  /admin/api/projects/{project_key}

GET  /admin/api/build-center/projects
GET  /admin/api/build-center/projects/{project_key}
POST /admin/api/build-center/projects/{project_key}/runs
GET  /admin/api/build-center/runs/{run_id}
GET  /admin/api/build-center/runs/{run_id}/logs
POST /admin/api/build-center/runs/{run_id}/cancel
POST /admin/api/build-center/runs/{run_id}/retry
```

制品与发布：

```text
GET  /admin/api/artifacts
GET  /admin/api/artifacts/{artifact_id}
POST /admin/api/release-plans
GET  /admin/api/release-plans
GET  /admin/api/release-plans/{plan_id}
POST /admin/api/release-plans/{plan_id}/publish
POST /admin/api/release-plans/{plan_id}/pause
POST /admin/api/release-plans/{plan_id}/rollback
```

部署：

```text
GET  /admin/api/deployment-targets
POST /admin/api/deployment-targets
POST /admin/api/deployments
GET  /admin/api/deployments
GET  /admin/api/deployments/{deployment_id}
```

### Worker API

其他机器通过 API 接入构建和发布：

```text
POST /api/v1/workers/register
POST /api/v1/workers/heartbeat
GET  /api/v1/workers/tasks/next
POST /api/v1/workers/tasks/{task_id}/logs
POST /api/v1/workers/tasks/{task_id}/artifacts
POST /api/v1/workers/tasks/{task_id}/complete
POST /api/v1/workers/tasks/{task_id}/fail
```

当前已落地：

- 第 6 号 migration 新增 `build_workers`、`worker_heartbeats`、`worker_tasks`。
- Worker API 使用和 CI API 相同的机器 Bearer Token 保护。
- 已支持 worker 注册、心跳、领取匹配标签的 queued task、日志 tail、产物 manifest、complete/fail 状态回传。
- `worker_tasks.required_labels` 使用标签子集匹配，支持 Linux/Windows/Android/Docker/Cloudflare 等构建能力调度。

worker 通过标签匹配任务：

```text
linux
windows
arm64
amd64
android
node
go
docker
cloudflare
```

### CI/API 兼容入口

继续保留现有轻量入口：

```text
POST /api/v1/ci/builds
POST /api/v1/ci/artifacts
POST /api/v1/ci/releases
POST /api/v1/webhooks/github
POST /api/v1/webhooks/gitea
```

## 5. GitHub 接入

一期支持：

- 绑定仓库 URL、full name、default ref。
- 配置 webhook secret 引用。
- 接收 push、tag、workflow_run。
- 原始事件写入 `webhook_events`。
- 根据 `webhook_routes` 创建 `build_center_runs`。
- 手工触发时允许覆盖 ref、version、channel、build action。

GitHub Actions 仍然可作为外部 CI 使用。外部 CI 构建完成后直接调用 `releasectl artifact-upload` 或 `POST /api/v1/ci/artifacts` 登记制品。

本机构建中心适合：

- 需要在本机缓存工具链和本地 registry 的项目。
- 需要统一从后台可视化触发的项目。
- 需要接其他机器 worker 的项目。

当前已落地：

- `/api/v1/webhooks/github` 和 `/api/v1/webhooks/gitea` 继续先写入 `webhook_events`。
- Webhook 入库后会匹配已启用的 `code_repositories.webhook_enabled` 和 `webhook_routes.enabled`。
- 命中 route 后创建 `build_center_runs` 并进入构建中心执行链路。
- 第 5 号 migration 已预置 release-center 的 main push 和 tag route，但默认 disabled；配置 GitHub secret/认证后再开启仓库和 route。

## 6. Cloudflare 接入

Cloudflare 不替代核心发布中心，作为部署和边缘层接入。

一期目标：

- Cloudflare Pages：Web dist、Docs dist、管理后台静态前端。
- Cloudflare Workers：Webhook 预处理、下载鉴权、Manifest 缓存、轻量边缘 API。
- Cloudflare R2：APK、ZIP、Web dist、binary、配置包对象存储。

部署动作由 `deployment_targets` 决定：

```text
cloudflare_pages  -> wrangler pages deploy <directory>
cloudflare_worker -> wrangler deploy
cloudflare_r2     -> wrangler r2 object put / S3 compatible upload
```

所有 Cloudflare token 只存引用：

```text
credential_ref=cf_token_release_prod
```

实际明文放环境变量、secret 文件或后续 Vault，不写入数据库、不写入 Git。

当前已落地：

- Admin API 支持创建 `deployment_targets` 和写入/查询 `deployment_records`。
- `POST /admin/api/deployments` 可按 project + target 创建部署记录，支持 dry-run。
- Cloudflare Pages/Workers/R2 目标会生成 wrangler 准备命令写入部署记录 metadata，等待凭证和执行器接入。
- 支持 `/complete` 和 `/fail` 回填外部部署状态、URL、日志和错误信息。
- Admin 前端新增部署中心，支持部署目标维护、Cloudflare Pages/Worker/R2 dry-run 部署记录、prepared command 展示和状态回填。

## 7. 全流程闭环

### 手工构建发布

```text
后台选择项目
  -> 选择 build profile / ref / 环境
  -> 创建 build_center_runs
  -> 本机 buildctl 或 worker 执行
  -> 写入 build_center_run_artifacts
  -> 上传到 Release Center CI API
  -> 写入 app_builds / app_build_artifacts
  -> 创建 release_plan
  -> 发布到目标环境
  -> 写 audit_events
```

### GitHub Webhook 构建发布

```text
GitHub push/tag
  -> /api/v1/webhooks/github
  -> webhook_events
  -> webhook_routes 命中 build profile
  -> build_center_runs
  -> 构建、上传、发布或等待人工确认
```

### Cloudflare 部署

```text
Web dist / Worker bundle 已登记
  -> 选择 deployment_target
  -> 执行 wrangler
  -> 写 deployment_records
  -> 回填 URL / deployment id / 状态
```

### 外部机器构建

```text
worker 注册
  -> 心跳上报 labels
  -> 拉取匹配任务
  -> 执行构建
  -> 上传日志和产物
  -> complete/fail
  -> 发布中心统一展示和发布
```

## 8. 阶段拆分

### M1：构建中心可视化 MVP

- 读取 `release_projects`、`code_repositories`、`build_profiles`。
- 展示 `build_center_runs` 和 `build_center_run_artifacts`。
- 后台触发本机 `buildctl all release-center`。
- 日志和产物可在页面查看。

### M2：多项目多类型发布

- 新增 `release_units`、`release_environments`、`release_plans`。
- 把 APK 发布页迁移成 Android 发布单元。
- Web、Docs、Worker、Server、Docker 都走统一制品和发布计划。

### M3：GitHub Webhook 自动触发

- 实现 `webhook_routes` 命中逻辑。
- 支持 push/tag 自动创建构建任务。
- 支持 workflow_run 只登记外部构建结果。

### M4：Cloudflare 部署目标

- Pages 部署 Web/Docs dist。
- Workers 部署 Worker bundle。
- R2 作为大文件对象存储目标。
- 写入 `deployment_records`。

### M5：外部 worker API

- worker 注册、心跳、取任务、日志、产物、完成状态。
- 后台按 worker labels 分发构建。
- 支持 Linux/Windows/Android/Docker/Cloudflare 标签。

### M6：权限、审批、回滚和审计

- 构建、发布、部署动作接 RBAC。
- prod 环境发布需要审批。
- 回滚基于上一条成功 `deployment_records` 或 `release_plans`。
- 所有操作写 `audit_events`。

## 9. 当前分支建议范围

当前分支 `feat/build-center-visualization` 建议先交付：

- 第 4 号 migration 和数据库切换。
- 构建中心 Admin API。
- 构建中心前端页面。
- 本机 `buildctl all release-center` 可视化触发。
- 构建日志、产物、上传状态展示。

多环境发布计划、Cloudflare 真部署、外部 worker API 放在后续小分支，避免一次改动过大。
