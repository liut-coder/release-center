# 构建中心可视化与多项目发布交接

更新时间：2026-06-05

## 1. 当前结论

当前分支：

```text
feat/build-center-visualization
```

交接文档创建前最新业务提交：

```text
7a9b8bc feat(web): seed release plans from android builds
```

公网验收入口：

```text
http://193.123.98.20:18085/
```

当前已把发布中心从单一 APK 发版页推进为轻量通用发布平台雏形：构建中心、制品中心、发布计划、部署中心、集成配置、Worker 接入、审批、审计、回滚和 Android 兼容迁移都已经有可视化入口和后端 API 基础闭环。

GitHub push 仍未成功。原因是当前机器没有可用的 GitHub HTTPS 凭据：

```text
fatal: could not read Username for 'https://github.com': No such device or address
```

配置凭据后执行：

```bash
git push -u origin feat/build-center-visualization
```

不要把 GitHub token 或后台 token 写入文档、聊天或 Git。公网后台登录 token 在服务器环境文件中维护，浏览器登录框填写 `ADMIN_TOKEN`。

## 2. 运行状态

服务：

```text
release-center-public.service
```

当前 smoke 结果：

```text
service=active
readyz=204
web_bundle=index-FT3BOcYR.js
release_plans=6
release_units=4
android_units=1
environments=dev:false,test:false,staging:true,prod:true
```

运行二进制：

```text
/root/build-center/runtime/release-center-server-patched
```

前端 dist 通过 `/etc/release-center-public.env` 里的 `WEB_DIST` 同步。

常用部署命令：

```bash
env GOCACHE=/root/build-center/cache/go/build \
  GOMODCACHE=/root/build-center/cache/go/pkg/mod \
  /root/build-center/cache/tools/go1.25.0/bin/go build -buildvcs=false \
  -o /root/build-center/runtime/release-center-server-patched.new ./cmd/server

install -m 0755 \
  /root/build-center/runtime/release-center-server-patched.new \
  /root/build-center/runtime/release-center-server-patched

systemctl restart release-center-public.service

npm --prefix web run typecheck
npm --prefix web run build

set -a
. /etc/release-center-public.env
set +a
cp -a web/dist/. "$WEB_DIST"/
```

## 3. 已完成范围

### 构建中心

- 已有项目、代码仓库、构建 profile、webhook route、构建运行和产物的 Admin API。
- 后台构建中心页面已可查看项目、运行、日志、产物和上传状态。
- 构建项目、仓库、profile、webhook route、手工构建任务都会写 `audit_events`。

### 制品中心

- 已统一聚合 `app_build_artifacts` 和 `build_center_run_artifacts`。
- 发布计划和部署记录可从制品中心选择不可变制品引用。
- Worker 回传 artifacts 或 complete 附带 artifacts 时，已按 `build_run_id` 镜像到 `build_center_run_artifacts`，外部机器构建产物能进入制品中心。

### 发布中心

- 已新增 `release_environments`、`release_units`、`release_plans`、`release_plan_artifacts`。
- 已预置 dev/test/staging/prod 环境和 release-center 的 Web/Server/Worker 发布单元。
- 发布计划支持创建、查询、publish、pause、rollback。
- 发布计划支持从制品中心带入版本、build number、commit、`immutable_ref` 和 artifact 关联 ID。
- 发布计划可直接按部署目标创建部署记录。
- prod/staging 等需审批环境已接入发布计划审批：publish 先进入 `pending_approval`，approve 后进入 `released`。
- 发布计划部署入口已防止绕过审批：需审批环境的非 dry-run 部署只允许 `released` / `rolling_out` 计划投递。
- release plan rollback 已支持基于同发布单元、同环境上一条 released/rolling_out 计划生成回滚计划，并可选择部署目标继续创建 dry-run 或 Worker 投递部署记录。

### Android 兼容迁移

- 旧 APK 发布接口继续保留兼容 update-check。
- APK 创建、发布、暂停、回滚、灰度和说明更新会自动镜像到 Android release unit / release plan。
- 前端 Android 构建记录已新增“发布计划”入口，可一键切到统一发布计划工作台并预填 Android unit、环境、版本和 APK 制品引用。
- 旧 Android 发布表单仍保留，作为兼容入口。

### 部署中心

- 支持多项目部署目标维护。
- 已支持 Cloudflare Pages、Cloudflare Worker、Cloudflare R2、Docker、Webhook、SSH、Kubernetes 等 provider 类型预留。
- `POST /admin/api/deployments` 可按 project + target 创建部署记录，支持 dry-run。
- 非 dry-run 部署会投递 `deploy` 类型 Worker task。
- Worker complete/fail 后会回填 `deployment_records` 状态、日志、外部部署 ID 和 URL。
- prod 非 dry-run 部署先进入 `pending_approval`，审批通过后才投递 Worker。
- 部署记录支持按同目标上一条成功部署生成回滚部署，dry-run 或非 dry-run 投递都已接入。

### Worker 接入

- 已支持 worker 注册、心跳、领取匹配标签任务、日志 tail、产物 manifest、complete/fail 状态回传。
- `required_labels` 使用标签子集匹配，支持 linux/windows/android/docker/cloudflare 等能力调度。
- Admin 前端有 Worker 接入页面，可查看 Worker 池、任务队列并手工投递任务。
- `releasectl worker-run` 已作为外部机器 agent，可注册、心跳、领取任务、执行 `metadata.command` / `metadata.prepared_command`、回传日志和结果。
- Cloudflare worker 机器凭据注入约定已实现，Worker 侧只记录 credential ref 和 env key 来源，不记录 token 明文。

### GitHub / Webhook 接入

- `/api/v1/webhooks/github` 和 `/api/v1/webhooks/gitea` 会先写入 `webhook_events`。
- 已实现 webhook route 命中逻辑，命中后创建 `build_center_runs`。
- `cmd/server` 已接入 GitHub/Gitea HMAC 签名校验中间件。
- 集成配置页面已能维护项目、Git 仓库、构建 profile、Webhook route 和 credential/secret 引用。
- 集成配置页面已新增快速接入向导：支持 GitHub Web / Go Server / Android 模板、粘贴仓库 URL 自动解析 provider/full name/project key、自动生成 profile/route/credential ref、预检必填项、复制 webhook endpoint、一键保存项目 + 仓库 + profile + route。已有项目、仓库、profile 和 route 可一键载入编辑，减少重复手填。
- 集成配置页面已新增外部接入闭环面板：展示项目、仓库、profile、credential ref、webhook secret ref、仓库 webhook、route 是否就绪；提供完整 Payload URL、Secret 引用、触发事件、ref pattern、route 状态、复制按钮、GitHub Webhooks 设置入口和“push 推荐配置”快捷填充。
- 集成配置页面已新增 Webhook Route 试跑：输入事件和 ref 后调用 Admin dry-run API，复用后端 route 匹配逻辑，只返回命中/未命中、build ref 和阻塞原因，不创建真实构建任务。
- release-center 的 main push 和 tag route 已有 migration 预置，但默认 disabled；配置 secret 和认证后再启用。

### RBAC / 权限

- Admin 写接口已接入路由层 RBAC：构建、集成配置、发布计划、旧 APK/资源发布、部署、Worker 任务和系统管理写操作都会先校验权限。
- 默认 `ADMIN_TOKEN` 请求注入 `system_admin`，也支持受信任上游或前端登录态通过 `X-Admin-Account` / `X-Admin-Role` 传入账号和角色。
- 当前内置角色权限矩阵：
  - `system_admin`：`system:*`、`release:*`、`build:*`、`deploy:*`、`worker:*`、`integration:*`。
  - `release_admin`：发布、构建、部署、Worker 和集成配置写权限，但不能维护系统用户、角色、权限、菜单和字典。
  - `release_viewer`：发布、构建、部署、Worker、集成配置只读和审计查看，写接口返回 403。
- 前端登录页可选择系统管理员、发版管理员、只读观察员，用于验证权限矩阵；真实密钥仍只通过后台 token 认证，不在页面或文档中暴露。
- 前端已同步权限矩阵，系统管理菜单、构建触发、Worker 投递、部署目标/部署记录、集成配置、发布计划、旧 Android 发布和资源发布等写按钮会按当前角色禁用，并在按钮 title 中提示缺少的权限点。

## 4. 最新提交

近期关键提交：

```text
e886a2f feat(audit): record build and release plan actions
affd651 feat(worker): inject cloudflare credentials for deploy tasks
6650973 feat(release): mirror apk releases to android plans
009fc88 feat(release): create rollback plans from release plans
8741cd6 feat(web): select targets for release plan rollbacks
d6b116f feat(release): require approval for prod release plans
7fc19e1 feat(release): block unapproved plan deployments
d035700 feat(worker): mirror task artifacts to build center
7a9b8bc feat(web): seed release plans from android builds
```

每个任务完成后都已本地提交，并尝试 push。push 当前统一失败在 GitHub HTTPS 凭据读取阶段。

## 5. 已执行验证

最近多轮任务已执行：

```bash
env GOCACHE=/root/build-center/cache/go/build \
  GOMODCACHE=/root/build-center/cache/go/pkg/mod \
  /root/build-center/cache/tools/go1.25.0/bin/go test ./...

npm --prefix web run typecheck
npm --prefix web run build

git diff --check
```

集成配置快速接入向导完成后已执行：

```bash
npm --prefix web run typecheck
npm --prefix web run build
```

公网 smoke：

```bash
systemctl is-active release-center-public.service
curl -sS -o /dev/null -w 'readyz=%{http_code}\n' http://193.123.98.20:18085/readyz
curl -sS http://193.123.98.20:18085/ | grep -o 'index-[A-Za-z0-9_\-]*\.js' | head -1
```

带后台 token 的 API smoke：

```bash
set -a
. /etc/release-center-public.env
set +a
curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" \
  http://193.123.98.20:18085/admin/api/release-plans
```

## 6. 当前阻塞

### GitHub push

当前远端：

```text
origin=https://github.com/liut-coder/release-center.git
```

push 失败：

```text
fatal: could not read Username for 'https://github.com': No such device or address
```

需要在服务器配置 GitHub HTTPS credential helper、PAT 或改用 SSH remote。不要把 PAT 写入仓库或交接文档。

### 真实外部验收

以下能力代码已接入，但仍需要真实外部资源做 smoke：

- GitHub webhook：配置真实 repo webhook secret，启用 repository 和 route 后 push/tag 验收。
- Cloudflare 真部署：在外部 worker 机器安装 wrangler，配置 Cloudflare token/account id 后执行 Pages/Workers/R2 smoke。
- Android 真机：旧 APK update-check 与统一 Android release plan 的端到端安装/升级 smoke。
- 生产回滚演练：用真实 target 做 release plan rollback + deployment rollback。

## 7. 下一步建议

优先顺序：

1. 配置 GitHub push 凭据，把当前分支推上远端，避免本地提交堆积。
2. 做 GitHub webhook 真实仓库 smoke：启用 secret、repository、route，push/tag 后确认 `webhook_events` 和 `build_center_runs`。
3. 做 Cloudflare worker 真部署 smoke：准备外部 worker 机器，执行 `releasectl worker-run -labels linux,node,cloudflare -execute`。
4. 做 Android 迁移收敛：逐步弱化旧 Android 发布表单，把日常入口集中到统一发布计划工作台。
5. 做真实回滚演练：生产目标上验证 release plan rollback、deployment rollback、审批和审计记录。

## 8. 安全注意

- `/etc/release-center-public.env` 包含后台和 CI token，只能在服务器本机读取，不要复制到聊天、文档或 Git。
- Cloudflare token 只应存在 worker 机器环境变量或 secret 文件中，数据库只保存 `credential_ref`。
- Webhook secret 只通过环境变量配置，不写入 `webhook_routes` 明文字段。
- 交接时只传 token 名称、环境变量名和配置位置，不传明文值。
