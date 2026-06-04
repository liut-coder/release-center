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
- Admin API 新增制品中心，`GET /admin/api/artifacts` 聚合 `app_build_artifacts` 和 `build_center_run_artifacts`，详情接口返回稳定制品引用 `immutable_ref`。
- Admin 前端新增制品中心页面，支持统一浏览、筛选、查看制品详情和复制发布/部署引用。
- 发布计划创建表单已接入制品中心，可选择制品后自动带入版本、build number、commit、`immutable_ref` 和 build/run/artifact 关联 ID。
- 发布计划已支持直接按目标创建部署记录；前端发布计划列表可选择部署目标并生成 dry-run 部署记录。
- 发布单元保存、发布计划创建、publish/pause/rollback 状态动作和发布计划部署都会写入 `audit_events`。
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
POST /admin/api/build-center/projects
GET  /admin/api/build-center/projects/{project_key}
POST /admin/api/build-center/projects/{project_key}/repositories
POST /admin/api/build-center/projects/{project_key}/profiles
POST /admin/api/build-center/projects/{project_key}/webhook-routes
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
POST /admin/api/release-plans/{plan_id}/deployments
```

部署：

```text
GET  /admin/api/deployment-targets
POST /admin/api/deployment-targets
POST /admin/api/deployments
GET  /admin/api/deployments
GET  /admin/api/deployments/{deployment_id}
POST /admin/api/deployments/{deployment_id}/approve
POST /admin/api/deployments/{deployment_id}/rollback
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
- Admin API 支持查看 Worker 池和任务队列，并可从后台创建 queued worker task。
- Admin 前端新增 Worker 接入页面，支持 Worker 状态、标签能力、任务队列和手工投递任务可视化。
- 非 dry-run 部署会自动投递 `deploy` 类型 Worker 任务，Worker complete/fail 后回填 `deployment_records` 状态、日志和外部部署信息。
- `releasectl worker-run` 已作为外部机器 agent 接入现有 Worker API，支持注册、心跳、领取任务、执行 `metadata.command` / `metadata.prepared_command`、回传日志和 complete/fail。
- 部署中心已支持从当前部署记录回滚到同一目标上一条成功部署；dry-run 只落回滚记录，非 dry-run 会继续投递 Worker。
- 部署目标保存、部署记录创建、完成、失败和回滚都会写 `audit_events`，用于审计部署链路。
- 构建项目、代码仓库、构建 profile、Webhook 路由和手工构建任务创建都会写 `audit_events`，用于审计构建链路。
- prod 非 dry-run 部署会先进入 `pending_approval`，审批通过后才投递 Worker。

外部机器最小启动方式：

```bash
releasectl worker-run \
  -base-url http://193.123.98.20:18085 \
  -token "$GAME_HELPER_CI_TOKEN" \
  -worker-key cf-prod-1 \
  -labels linux,node,cloudflare \
  -workdir /srv/release-center \
  -execute
```

`-execute` 是显式执行开关。未开启时 agent 会领取任务、写入拒绝执行日志并 fail 任务，避免外部机器误执行来自服务端的命令。

Worker 任务命令来源优先级：

```text
metadata.prepared_command  # 推荐用于部署，数组形式避免 shell 解析差异
metadata.command           # 手工任务，可为字符串命令
metadata.commands[action]  # 构建 profile 或动作映射
metadata.commands.default
```

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
- `cmd/server` 已接入 GitHub/Gitea webhook HMAC 签名校验中间件；配置 secret 后才放行真实 webhook 请求。
- Admin API 已支持保存 `release_projects`、`code_repositories`、`build_profiles` 和 `webhook_routes`，用于后台接入 GitHub 仓库和构建 profile。
- Admin 前端新增集成配置页面，支持维护项目、Git 仓库、构建 profile、Webhook route 和对应 credential/secret 引用。
- 第 5 号 migration 已预置 release-center 的 main push 和 tag route，但默认 disabled；配置 GitHub secret/认证后再开启仓库和 route。

Webhook secret 环境变量：

```text
GITHUB_WEBHOOK_SECRET    GitHub X-Hub-Signature-256 校验
GITEA_WEBHOOK_SECRET     Gitea X-Gitea-Signature 校验
WEBHOOK_SECRET           两者共用的 fallback secret
```

真实 GitHub 接入验收顺序：

```text
1. 在 GitHub 仓库 Webhooks 配置 Payload URL: /api/v1/webhooks/github。
2. 设置 Secret，并在 server 环境配置 GITHUB_WEBHOOK_SECRET。
3. 在集成配置页启用 code_repositories.webhook_enabled 和 trigger_on_push / trigger_on_tag。
4. 启用匹配的 webhook_routes。
5. push main 或 tag 后检查 webhook_events、build_center_runs 和构建中心页面。
```

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
- Cloudflare Pages/Workers/R2 目标会生成 wrangler 准备命令写入部署记录 metadata。
- 非 dry-run Cloudflare 部署会按 provider 投递带 `cloudflare` 标签的 Worker 任务，外部 worker 可读取 prepared command 执行 wrangler。
- `releasectl worker-run -labels linux,node,cloudflare -execute` 可在已安装 wrangler 的机器上执行这些 prepared command。
- 支持 `/complete` 和 `/fail` 回填外部部署状态、URL、日志和错误信息。
- Admin 前端新增部署中心，支持部署目标维护、从制品中心选择制品创建 Cloudflare Pages/Worker/R2 dry-run 或 Worker 投递部署记录、prepared command 展示和状态回填。
- Admin 前端部署记录支持回滚操作，复用部署投递区的 Dry-run 开关控制只落记录或直接投递 Worker。

Cloudflare worker 机器凭据约定：

```text
CLOUDFLARE_API_TOKEN     wrangler 使用的真实 token，不写入数据库和 Git
CLOUDFLARE_ACCOUNT_ID    需要账号上下文时在机器环境提供
```

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
  -> 制品中心生成 immutable_ref
  -> 创建 release_plan
  -> 由 release_plan 创建 deployment_records
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
  -> dry-run 记录或投递 Worker 执行 wrangler
  -> 写 deployment_records
  -> 回填 URL / deployment id / 状态
  -> 失败时按同一目标上一条 success 部署创建 rollback deployment
  -> dry-run 验证或投递 Worker 执行回滚
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

当前已落地：

- 部署记录已支持按同目标上一条成功记录生成回滚部署，并可 dry-run 或投递 Worker。
- 构建中心配置/任务创建、发布单元/发布计划/发布计划部署、部署目标保存、部署创建、完成、失败和回滚已写入 `audit_events`。
- prod 部署审批门禁已接入部署中心：非 dry-run 先落 `pending_approval` 记录，批准后投递 Worker。

## 9. 当前分支建议范围

当前分支 `feat/build-center-visualization` 建议先交付：

- 第 4 号 migration 和数据库切换。
- 构建中心 Admin API。
- 构建中心前端页面。
- 本机 `buildctl all release-center` 可视化触发。
- 构建日志、产物、上传状态展示。
- 制品中心统一聚合构建中心产物和 App 构建制品，作为发布计划和部署记录的入口。

多环境发布计划、部署记录、外部 worker API 已进入当前分支；Cloudflare 真部署和审批/RBAC 继续放在后续小分支，避免一次改动过大。

## 10. 2026-06-05 进度落档

当前工作分支：

```text
feat/build-center-visualization
```

当前公网验收入口：

```text
http://193.123.98.20:18085/
```

本分支已经把轻量化架构从“方案”推进到可视化闭环雏形：

- 构建中心：项目、仓库、profile、webhook route、构建运行和产物已经有 Admin API 和后台工作台。
- 制品中心：统一聚合 `app_build_artifacts` 与 `build_center_run_artifacts`，发布计划和部署记录可以复用不可变制品引用。
- 发布中心：新增多环境、多发布单元和发布计划，不再只面向 APK；旧 APK 发布入口保留。
- 部署中心：支持多项目部署目标、Cloudflare Pages/Workers/R2、Docker、Webhook、SSH、Kubernetes 等 provider 预留，支持 dry-run 和非 dry-run 投递。
- 集成配置：Git 仓库、构建 profile、webhook route 和 credential/secret 引用已经进入后台维护页面。
- Worker 接入：外部机器可注册、心跳、领取任务、上传日志和产物、complete/fail 回填状态。
- 部署闭环：非 dry-run 部署会创建 `deploy` 类型 Worker task，Worker 回传后更新 `deployment_records` 的状态、日志、外部部署 ID 和 URL。

当前本地提交序列：

```text
0dbde09 feat(artifacts): add artifact center workbench
d35fb2a feat(release): select artifacts for release plans
1a82bd7 feat(deploy): select artifacts for deployments
6928ebd feat(build-center): add integration config api
0732b3f feat(integrations): add integration config workbench
8f28956 feat(release): create deployments from release plans
761ce0c feat(deploy): dispatch deployments to workers
```

本次落档前验证通过：

```text
go test ./...
npm --prefix web run typecheck
npm --prefix web run build
```

GitHub 推送状态：

```text
origin=https://github.com/liut-coder/release-center.git
当前环境缺少 GitHub HTTPS 凭据，push 会失败在用户名读取阶段。
配置凭据后执行：git push -u origin feat/build-center-visualization
```

下一批建议按小步提交推进：

- Cloudflare 真实执行器：提供 worker 侧 wrangler 执行脚本、凭据注入约定、Pages/Workers/R2 smoke。
- GitHub 接入验收：配置 webhook secret/route enable，完成 push/tag 自动创建构建任务的真实仓库 smoke。
- Android 发布单元迁移：把旧 APK 发布页收敛进统一 release unit，同时保留 update-check 兼容 API。
- 审批/RBAC：构建/发布/部署权限拦截和角色权限矩阵验收。
- 回滚执行闭环：基于上一条成功 `deployment_records` 或 `release_plans` 生成回滚计划并投递 Worker。
