# 独立发布中心详细方案

更新时间：2026-06-02

## 1. 背景与目标

当前仓库已有 `appreleases` 模块，覆盖游戏助手 App 的 APK 整包发布、启动页资源增量更新、资源白名单校验、CLI 打包上传、灰度发布、自动暂停和审计。独立发布中心的目标是在此基础上抽象出一套可独立部署、可服务多应用、多环境、多渠道、多制品类型的发布平台。

独立发布中心不再只是某个业务服务里的“App 发版页面”，而是一个独立系统：

```text
CI/CD
  ↓
发布中心制品接入
  ↓
版本与策略编排
  ↓
审批与发布
  ↓
客户端下载/服务端拉取
  ↓
设备/实例事件回传
  ↓
观测、回滚、审计
```

核心目标：

1. 支持多应用、多平台、多环境、多渠道统一发布。
2. 支持 APK、AAB、增量资源包、配置包、Web 前端包、服务端二进制、Docker 镜像等制品类型。
3. 区分构建、制品、发布计划、发布批次和运行事件。
4. 支持普通发布、灰度发布、定向发布、强制更新、暂停、召回、回滚。
5. 支持发布审批、权限隔离、操作审计和发布冻结窗口。
6. 支持从 GitHub、自建 Gitea 拉取项目，并对 Android、iOS、Java、Go、Python、Node.js、PHP、Docker 等主流技术栈执行持续集成。
7. 支持独立客户端 API、管理后台 API、CI Runner API 和 Webhook API。
8. 支持与现有 `appreleases` 模块平滑迁移，优先复用 APK 与资源增量能力。
9. 支持生产级安全：制品校验、Manifest 签名、访问令牌、下载防盗链、敏感操作二次确认。
10. 支持发布质量观测：下载成功率、安装成功率、激活失败率、版本分布、回滚原因。
11. 支持后续扩展为通用发布平台，而不是只服务游戏助手 App。

## 2. 建设原则

### 2.1 独立部署

发布中心应作为独立服务部署，拥有独立域名、数据库、对象存储目录、后台管理入口和 API Token。

建议域名：

```text
管理后台：https://release-admin.079999.xyz
开放 API：https://release.079999.xyz
文件下载：https://file.079999.xyz
```

### 2.2 构建与发布分离

构建成功不等于发布。发布中心可以接入外部 CI，也可以在后续提供内置 CI Runner。无论制品来自 GitHub Actions、Gitea Actions、Jenkins，还是发布中心自建 Runner，都必须先形成构建记录和制品记录，再进入发布策略、审批、灰度和回滚流程。

```text
build
  ↓
artifact
  ↓
release_plan
  ↓
release_batch
  ↓
runtime_event
```

### 2.6 流水线模板优先

不同语言的构建命令、缓存目录、测试命令和制品路径差异较大。发布中心不应让用户每次手写完整脚本，而应提供主流语言模板，再允许按项目覆盖。

模板分层：

```text
系统模板
  ↓
项目模板
  ↓
应用流水线
  ↓
单次运行参数
```

### 2.3 制品不可变

已登记的制品不可覆盖。任何重新打包都必须产生新的 `artifact_id`、`build_number` 或 `resource_version`。

不可变字段：

```text
artifact_type
file_name
storage_key
size_bytes
sha256
signature
build_commit
created_at
```

### 2.4 策略可变但必须审计

发布说明、灰度比例、目标人群、暂停、召回、回滚可以修改，但必须记录操作人、变更前后内容和时间。

### 2.5 增量资源只发布数据和静态资源

延续现有规范：增量资源不得发布 Dex、JAR、动态库、脚本、未审计可执行代码。APK、AAB、二进制和脚本类能力必须走整包或服务端发布流程。

## 3. 范围边界

### 3.1 一期范围

一期聚焦 App 发布中心独立化和外部 CI 接入：

- 多应用管理。
- Android APK 整包发布。
- 启动页资源增量发布。
- 构建登记与制品上传。
- GitHub / Gitea 仓库绑定。
- Git Webhook 接收。
- 发布创建、编辑、审批、发布、暂停、召回、回滚。
- 指定渠道、指定设备、指定用户、百分比灰度。
- 客户端检查更新 API。
- 资源 Manifest 获取与包下载。
- 设备安装记录和升级事件上报。
- 管理后台基础页面。
- CLI 上传和外部 CI 接入。

### 3.2 二期范围

- 内置 CI Runner。
- 多语言流水线模板：Android、iOS、Java、Go、Python、Node.js、PHP、Docker。
- GitHub 和 Gitea 代码拉取、分支选择、Tag 构建、PR/MR 构建。
- Docker 镜像构建、推送和部署 Webhook。
- iOS TestFlight / App Store 发布状态接入。
- Web 前端包发布。
- 服务端二进制或容器镜像发布编排。
- 多租户项目空间。
- 发布日历和冻结窗口。
- 自动化质量门禁和自动回滚。
- Manifest Ed25519 签名和客户端验签。
- 更完整的报表与告警。

### 3.3 暂不做

- 绕过 Android 系统安装流程的 APK 更新。
- 一期不做 APK 差分补丁。
- 不通过增量资源发布可执行代码。
- 一期不强制把 CI 构建逻辑完全搬进发布中心；先支持 GitHub Actions、Gitea Actions、Jenkins 等外部 CI 登记结果，二期再建设内置 Runner。

## 4. 总体架构

```text
┌─────────────────────────────────────────────────────────┐
│                      管理后台 Web                         │
│  应用 / 构建 / 制品 / 发布 / 灰度 / 审批 / 监控 / 审计       │
└───────────────────────────┬─────────────────────────────┘
                            │ Admin API
┌───────────────────────────▼─────────────────────────────┐
│                     Release Center API                   │
│                                                         │
│  App 管理      制品管理      发布编排      策略引擎         │
│  审批流        灰度引擎      事件接收      观测统计         │
│  审计日志      回滚控制      Manifest      Webhook        │
└───────────────┬───────────────────────┬─────────────────┘
                │                       │
        ┌───────▼────────┐      ┌───────▼────────┐
        │ PostgreSQL      │      │ 对象存储/文件中心 │
        │ 元数据/事件/审计 │      │ APK/ZIP/Manifest │
        └────────────────┘      └────────────────┘
                ▲                       ▲
                │                       │
┌───────────────┴──────────────┐ ┌──────┴──────────────────┐
│ CI/CD / releasectl            │ │ 客户端 / 设备 / 服务实例   │
│ 构建登记、制品上传、资源打包    │ │ 检查更新、下载、事件回传     │
└──────────────────────────────┘ └─────────────────────────┘
```

## 5. 核心模块

### 5.1 应用与项目模块

职责：

- 管理应用基本信息。
- 管理平台：Android、iOS、Web、Server。
- 管理包名、应用 Key、负责人、默认渠道、默认发布策略。
- 管理项目空间，后续支持多团队隔离。

关键概念：

```text
project      项目空间
application  应用
platform     平台
environment 运行环境
channel      发布渠道
```

### 5.2 构建模块

职责：

- 接收外部 CI 登记构建结果。
- 创建内置 CI 构建任务。
- 保存 Git 信息、构建参数、构建状态、日志摘要。
- 关联一个或多个制品。
- 允许失败构建留档，便于追溯。

构建状态：

```text
building
success
failed
archived
```

### 5.3 代码源模块

职责：

- 绑定 GitHub 仓库。
- 绑定自建 Gitea 仓库。
- 支持 HTTPS Token、SSH Key、GitHub App、Gitea Token。
- 同步仓库、分支、Tag、提交记录。
- 接收 Push、Tag、PR/MR Webhook。
- 按项目配置默认构建分支和触发规则。

支持代码源：

```text
github
gitea
gitlab
generic_git
```

一期建议优先支持：

```text
GitHub
Gitea
```

Webhook 事件：

```text
push
tag_push
pull_request
merge_request
release
manual
schedule
```

### 5.4 CI Runner 模块

职责：

- 根据流水线定义创建构建任务。
- 拉取 Git 仓库代码。
- 准备语言运行环境。
- 执行安装依赖、测试、构建、打包、上传制品。
- 实时采集日志。
- 归档构建缓存和测试报告。
- 将构建产物登记为 `artifacts`。

Runner 类型：

```text
external_ci     外部 CI 回传结果
docker_runner   发布中心基于 Docker 执行构建
shell_runner    指定机器 Shell 执行构建
macos_runner    iOS/macOS 构建专用 Runner
k8s_runner      Kubernetes Job 执行构建
```

一期策略：

```text
外部 CI 接入为主
Docker Runner 做 Linux 技术栈
iOS 构建预留 macOS Runner，不在 Linux 机器上强行实现
```

### 5.5 流水线模板模块

职责：

- 为主流语言提供默认构建模板。
- 根据仓库文件自动识别技术栈。
- 允许项目覆盖安装命令、测试命令、构建命令和制品路径。
- 支持缓存、环境变量、Secret、构建矩阵。
- 支持流水线复用和版本化。

自动识别规则：

| 技术栈 | 识别文件 | 默认制品 |
|---|---|---|
| Android | `build.gradle`、`settings.gradle`、`AndroidManifest.xml` | APK / AAB |
| iOS | `*.xcodeproj`、`*.xcworkspace`、`Package.swift` | IPA / xcarchive |
| Java Maven | `pom.xml` | JAR / WAR |
| Java Gradle | `build.gradle`、`settings.gradle` | JAR / WAR |
| Go | `go.mod` | 二进制 |
| Python | `pyproject.toml`、`requirements.txt`、`setup.py` | Wheel / sdist / Docker 镜像 |
| Node.js | `package.json`、`pnpm-lock.yaml`、`yarn.lock` | dist / tarball / Docker 镜像 |
| PHP | `composer.json` | vendor 包 / Web 包 / Docker 镜像 |
| Docker | `Dockerfile`、`docker-compose.yml` | Docker 镜像 |

### 5.6 部署编排模块

职责：

- 将发布中心里的制品交给目标环境部署。
- 支持 App 更新、资源更新、Web 静态包发布、服务端二进制发布、Docker 镜像发布。
- 支持通过 Webhook 调用外部部署系统。
- 支持后续接入 Kubernetes、Docker Compose、SSH 主机、对象存储、CDN 刷新。

部署目标：

```text
android_client     Android 客户端更新
ios_client         iOS TestFlight / App Store
static_web         静态 Web 站点
server_host        服务器目录或 systemd 服务
docker_host        Docker 主机
kubernetes         Kubernetes Deployment / Helm Release
object_storage     对象存储发布
```

### 5.7 制品模块

职责：

- 上传、校验、存储和下载制品。
- 计算 SHA-256、文件大小、MIME 类型。
- 对资源 ZIP 进行内容安全校验。
- 生成下载 URL。
- 支持后续接入对象存储、CDN 和签名下载。

制品类型：

```text
android_apk
android_aab
resource_zip
resource_manifest
web_bundle
server_binary
container_image
config_bundle
docker_image
ios_ipa
java_jar
java_war
python_wheel
node_dist
php_bundle
```

### 5.8 发布编排模块

职责：

- 基于构建和制品创建发布。
- 编辑标题、摘要、发布说明、升级提示。
- 设置发布渠道、发布时间、更新级别和最低兼容版本。
- 控制状态流转。

发布状态：

```text
draft
pending_approval
approved
scheduled
testing
rolling_out
released
paused
recalled
rollback_running
rolled_back
archived
```

### 5.9 策略与灰度模块

职责：

- 判断某个设备、用户、版本、渠道是否命中发布。
- 支持百分比灰度和稳定哈希。
- 支持用户、设备、用户组、地区、当前版本、系统版本、设备型号等规则。
- 支持互斥规则和优先级。

推荐命中顺序：

```text
应用是否启用
  ↓
渠道是否匹配
  ↓
客户端版本是否低于发布版本
  ↓
是否满足最低系统/最低 App 版本
  ↓
是否命中定向规则
  ↓
是否命中百分比灰度
  ↓
返回最新可用发布
```

百分比灰度使用稳定哈希：

```text
hash(app_key + channel + release_id + device_id) % 100 < rollout_percentage
```

### 5.10 审批模块

职责：

- 对生产渠道或强制更新进行审批。
- 支持不同渠道不同审批要求。
- 支持发布前检查项确认。
- 记录审批意见和审批结果。

建议规则：

| 场景 | 审批要求 |
|---|---|
| `dev` 发布 | 不需要审批 |
| `internal` 发布 | 不需要审批或一人确认 |
| `beta` 发布 | 一人审批 |
| `stable` 发布 | 两人审批 |
| `emergency` 发布 | 一人审批，事后复盘 |
| 强制更新 | 必须审批 |
| 回滚 | 一人审批，可配置紧急免审 |

### 5.11 事件与观测模块

职责：

- 接收客户端更新事件。
- 更新设备安装记录。
- 聚合发布成功率和失败率。
- 识别资源激活失败并自动暂停。
- 输出报表和告警。

关键指标：

```text
检查更新次数
命中更新次数
下载开始次数
下载完成次数
校验失败次数
安装开始次数
安装成功次数
安装失败次数
资源激活成功次数
资源激活失败次数
版本分布
设备分布
回滚次数
```

### 5.12 审计模块

职责：

- 记录所有敏感操作。
- 保存变更前后 JSON。
- 支持按应用、操作人、目标对象、时间范围查询。

审计动作示例：

```text
app.create
build.create
artifact.upload
release.create
release.update_notes
release.submit_approval
release.approve
release.reject
release.publish
release.pause
release.recall
release.rollback
release.rollout_update
resource.create
resource.publish
resource.pause
resource.auto_pause
```

## 6. 发布对象模型

独立发布中心建议将数据拆成六类主对象。

```text
apps              应用
repositories      代码仓库
pipelines         流水线
pipeline_runs     流水线运行
builds            构建
artifacts         制品
release_plans     发布计划
release_batches   发布批次
deployments       部署记录
runtime_events    运行事件
```

关系：

```text
app 1 ── n build
app 1 ── n repository
repository 1 ── n pipeline
pipeline 1 ── n pipeline_run
pipeline_run 1 ── 1 build
build 1 ── n artifact
release_plan n ── n artifact
release_plan 1 ── n release_batch
release_plan 1 ── n deployment
release_batch 1 ── n runtime_event
```

### 6.1 为什么保留 release_batch

发布计划表示“要发布什么和发布规则”，发布批次表示“一次实际放量动作”。

例如：

```text
发布计划：game-helper 1.0.0 stable
批次 1：5% 灰度
批次 2：20% 灰度
批次 3：50% 灰度
批次 4：100% 全量
```

这样可以追踪每次放量的质量数据，而不是只知道最终发布到了 100%。

## 7. 数据模型建议

以下是独立发布中心的一期核心表。现有 `apps`、`app_builds`、`app_releases`、`app_resource_versions` 可以逐步迁移或映射到这些通用表。

### 7.1 projects

```sql
CREATE TABLE projects (
  id            BIGSERIAL PRIMARY KEY,
  project_key   VARCHAR(64) UNIQUE NOT NULL,
  name          VARCHAR(128) NOT NULL,
  owner         VARCHAR(128),
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 7.2 applications

```sql
CREATE TABLE applications (
  id              BIGSERIAL PRIMARY KEY,
  project_id      BIGINT REFERENCES projects(id),
  app_key         VARCHAR(64) UNIQUE NOT NULL,
  name            VARCHAR(128) NOT NULL,
  platform        VARCHAR(32) NOT NULL,
  package_name    VARCHAR(255),
  description     TEXT,
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 7.3 environments

```sql
CREATE TABLE environments (
  id              BIGSERIAL PRIMARY KEY,
  env_key         VARCHAR(32) UNIQUE NOT NULL,
  name            VARCHAR(64) NOT NULL,
  approval_policy VARCHAR(64),
  freeze_policy   JSONB,
  created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
```

推荐环境：

```text
dev
test
staging
prod
```

### 7.4 channels

```sql
CREATE TABLE channels (
  id              BIGSERIAL PRIMARY KEY,
  app_id          BIGINT NOT NULL REFERENCES applications(id),
  channel_key     VARCHAR(32) NOT NULL,
  name            VARCHAR(64) NOT NULL,
  environment_id  BIGINT REFERENCES environments(id),
  public_visible  BOOLEAN NOT NULL DEFAULT FALSE,
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, channel_key)
);
```

推荐渠道：

```text
dev
internal
beta
stable
emergency
```

### 7.5 repositories

```sql
CREATE TABLE repositories (
  id                  BIGSERIAL PRIMARY KEY,
  project_id          BIGINT REFERENCES projects(id),
  app_id              BIGINT REFERENCES applications(id),
  provider            VARCHAR(32) NOT NULL,
  repo_url            TEXT NOT NULL,
  owner               VARCHAR(128),
  repo_name           VARCHAR(128),
  default_branch      VARCHAR(128) DEFAULT 'main',
  auth_type           VARCHAR(32) NOT NULL,
  credential_ref      VARCHAR(255),
  webhook_secret_ref  VARCHAR(255),
  enabled             BOOLEAN NOT NULL DEFAULT TRUE,
  last_synced_at      TIMESTAMP,
  created_by          VARCHAR(128),
  created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`provider`：

```text
github
gitea
gitlab
generic_git
```

`auth_type`：

```text
none
https_token
ssh_key
github_app
gitea_token
```

### 7.6 pipeline_templates

```sql
CREATE TABLE pipeline_templates (
  id                BIGSERIAL PRIMARY KEY,
  template_key      VARCHAR(64) UNIQUE NOT NULL,
  name              VARCHAR(128) NOT NULL,
  language          VARCHAR(32) NOT NULL,
  runner_type       VARCHAR(32) NOT NULL,
  image             VARCHAR(255),
  definition        JSONB NOT NULL,
  artifact_rules    JSONB,
  cache_rules       JSONB,
  enabled           BOOLEAN NOT NULL DEFAULT TRUE,
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);
```

模板示例：

```text
android-gradle-apk
ios-xcode-ipa
java-maven-jar
java-gradle-jar
go-build-binary
python-wheel
node-web-dist
php-composer-web
docker-build-push
```

### 7.7 pipelines

```sql
CREATE TABLE pipelines (
  id                 BIGSERIAL PRIMARY KEY,
  app_id             BIGINT NOT NULL REFERENCES applications(id),
  repository_id      BIGINT REFERENCES repositories(id),
  template_id        BIGINT REFERENCES pipeline_templates(id),
  name               VARCHAR(128) NOT NULL,
  language           VARCHAR(32) NOT NULL,
  trigger_rules      JSONB NOT NULL,
  variables          JSONB,
  secret_refs        JSONB,
  definition         JSONB NOT NULL,
  artifact_rules     JSONB,
  enabled            BOOLEAN NOT NULL DEFAULT TRUE,
  created_by         VARCHAR(128),
  created_at         TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`trigger_rules` 示例：

```json
{
  "pushBranches": ["main", "release/*"],
  "tags": ["v*"],
  "pullRequest": true,
  "manual": true,
  "schedule": "0 2 * * *"
}
```

### 7.8 pipeline_runs

```sql
CREATE TABLE pipeline_runs (
  id                  BIGSERIAL PRIMARY KEY,
  pipeline_id          BIGINT NOT NULL REFERENCES pipelines(id),
  build_id             BIGINT,
  runner_id            BIGINT,
  status               VARCHAR(32) NOT NULL,
  trigger_type         VARCHAR(32) NOT NULL,
  git_ref              VARCHAR(255),
  git_branch           VARCHAR(128),
  git_tag              VARCHAR(128),
  git_commit           VARCHAR(64),
  commit_message       TEXT,
  started_by           VARCHAR(128),
  started_at           TIMESTAMP,
  finished_at          TIMESTAMP,
  duration_ms          BIGINT,
  log_storage_key      TEXT,
  error_message        TEXT,
  metadata             JSONB,
  created_at           TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`status`：

```text
queued
running
success
failed
canceled
timeout
```

### 7.9 pipeline_steps

```sql
CREATE TABLE pipeline_steps (
  id              BIGSERIAL PRIMARY KEY,
  run_id          BIGINT NOT NULL REFERENCES pipeline_runs(id),
  step_key        VARCHAR(64) NOT NULL,
  name            VARCHAR(128) NOT NULL,
  status          VARCHAR(32) NOT NULL,
  started_at      TIMESTAMP,
  finished_at     TIMESTAMP,
  duration_ms     BIGINT,
  log_tail        JSONB,
  error_message   TEXT,
  created_at      TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 7.10 runner_agents

```sql
CREATE TABLE runner_agents (
  id                BIGSERIAL PRIMARY KEY,
  runner_key        VARCHAR(64) UNIQUE NOT NULL,
  name              VARCHAR(128) NOT NULL,
  runner_type       VARCHAR(32) NOT NULL,
  os_type           VARCHAR(32),
  arch              VARCHAR(32),
  labels            JSONB,
  max_concurrency   INT NOT NULL DEFAULT 1,
  status            VARCHAR(32) NOT NULL,
  last_seen_at      TIMESTAMP,
  registered_by     VARCHAR(128),
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP NOT NULL DEFAULT NOW()
);
```

Runner 标签示例：

```text
linux
docker
android
node20
go1.22
python3.12
php8.3
macos
xcode
ios
```

### 7.11 builds

```sql
CREATE TABLE builds (
  id                 BIGSERIAL PRIMARY KEY,
  app_id             BIGINT NOT NULL REFERENCES applications(id),
  ci_provider        VARCHAR(64),
  ci_pipeline_id     VARCHAR(128),
  ci_job_id          VARCHAR(128),
  git_repo           TEXT,
  git_ref            VARCHAR(255),
  git_branch         VARCHAR(128),
  git_commit         VARCHAR(64),
  build_type         VARCHAR(32),
  version_name       VARCHAR(64),
  version_code       BIGINT,
  build_number       BIGINT NOT NULL,
  status             VARCHAR(32) NOT NULL,
  started_by         VARCHAR(128),
  started_at         TIMESTAMP,
  finished_at        TIMESTAMP,
  log_url            TEXT,
  log_tail           JSONB,
  error_message      TEXT,
  created_at         TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, build_number)
);
```

### 7.12 artifacts

```sql
CREATE TABLE artifacts (
  id                BIGSERIAL PRIMARY KEY,
  app_id            BIGINT NOT NULL REFERENCES applications(id),
  build_id          BIGINT REFERENCES builds(id),
  artifact_type     VARCHAR(64) NOT NULL,
  file_name         VARCHAR(255) NOT NULL,
  storage_key       TEXT NOT NULL,
  download_url      TEXT,
  size_bytes        BIGINT NOT NULL,
  sha256            VARCHAR(64) NOT NULL,
  signature         TEXT,
  metadata          JSONB,
  immutable         BOOLEAN NOT NULL DEFAULT TRUE,
  created_by        VARCHAR(128),
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, sha256)
);
```

### 7.13 release_plans

```sql
CREATE TABLE release_plans (
  id                       BIGSERIAL PRIMARY KEY,
  app_id                   BIGINT NOT NULL REFERENCES applications(id),
  channel_id               BIGINT REFERENCES channels(id),
  release_type             VARCHAR(64) NOT NULL,
  version_name             VARCHAR(64),
  version_code             BIGINT,
  resource_version         VARCHAR(64),
  status                   VARCHAR(32) NOT NULL,
  title                    VARCHAR(255) NOT NULL,
  summary                  TEXT,
  release_notes_markdown   TEXT,
  upgrade_message          TEXT,
  update_level             VARCHAR(32) NOT NULL DEFAULT 'normal',
  min_supported_code       BIGINT,
  min_app_version_code     BIGINT,
  max_app_version_code     BIGINT,
  block_old_versions       BOOLEAN NOT NULL DEFAULT FALSE,
  scheduled_at             TIMESTAMP,
  published_at             TIMESTAMP,
  paused_at                TIMESTAMP,
  recalled_at              TIMESTAMP,
  created_by               VARCHAR(128),
  created_at               TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`release_type`：

```text
android_full
android_resource
web_bundle
server_binary
docker_image
config_bundle
```

### 7.14 release_artifacts

```sql
CREATE TABLE release_artifacts (
  release_id    BIGINT NOT NULL REFERENCES release_plans(id),
  artifact_id   BIGINT NOT NULL REFERENCES artifacts(id),
  role          VARCHAR(64) NOT NULL,
  created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (release_id, artifact_id)
);
```

`role` 示例：

```text
apk
manifest
resource_package
mapping_file
symbols
web_bundle
docker_image
```

### 7.15 release_rules

```sql
CREATE TABLE release_rules (
  id            BIGSERIAL PRIMARY KEY,
  release_id    BIGINT NOT NULL REFERENCES release_plans(id),
  rule_type     VARCHAR(64) NOT NULL,
  operator      VARCHAR(32) NOT NULL DEFAULT 'in',
  rule_value    JSONB NOT NULL,
  priority      INT NOT NULL DEFAULT 100,
  enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`rule_type`：

```text
user_id
device_id
user_group
region
percentage
current_version
os_version
device_model
package_name
custom
```

### 7.16 release_batches

```sql
CREATE TABLE release_batches (
  id                    BIGSERIAL PRIMARY KEY,
  release_id            BIGINT NOT NULL REFERENCES release_plans(id),
  batch_no              INT NOT NULL,
  rollout_percentage    INT NOT NULL,
  status                VARCHAR(32) NOT NULL,
  started_by            VARCHAR(128),
  started_at            TIMESTAMP,
  completed_at          TIMESTAMP,
  stopped_at            TIMESTAMP,
  metrics_snapshot      JSONB,
  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(release_id, batch_no)
);
```

### 7.17 approvals

```sql
CREATE TABLE approvals (
  id              BIGSERIAL PRIMARY KEY,
  release_id      BIGINT NOT NULL REFERENCES release_plans(id),
  status          VARCHAR(32) NOT NULL,
  requested_by    VARCHAR(128) NOT NULL,
  requested_at    TIMESTAMP NOT NULL DEFAULT NOW(),
  decided_by      VARCHAR(128),
  decided_at      TIMESTAMP,
  comment         TEXT
);
```

### 7.18 deployment_targets

```sql
CREATE TABLE deployment_targets (
  id                BIGSERIAL PRIMARY KEY,
  app_id            BIGINT REFERENCES applications(id),
  environment_id    BIGINT REFERENCES environments(id),
  target_key        VARCHAR(64) NOT NULL,
  target_type       VARCHAR(64) NOT NULL,
  name              VARCHAR(128) NOT NULL,
  config            JSONB NOT NULL,
  credential_ref    VARCHAR(255),
  enabled           BOOLEAN NOT NULL DEFAULT TRUE,
  created_by        VARCHAR(128),
  created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, target_key)
);
```

`target_type`：

```text
app_update_api
testflight
app_store_connect
static_web
ssh_host
docker_host
kubernetes
webhook
object_storage
cdn
```

### 7.19 deployments

```sql
CREATE TABLE deployments (
  id                  BIGSERIAL PRIMARY KEY,
  release_id          BIGINT REFERENCES release_plans(id),
  target_id           BIGINT REFERENCES deployment_targets(id),
  status              VARCHAR(32) NOT NULL,
  strategy            VARCHAR(32) NOT NULL,
  started_by          VARCHAR(128),
  started_at          TIMESTAMP,
  finished_at         TIMESTAMP,
  log_storage_key     TEXT,
  result              JSONB,
  error_message       TEXT,
  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`strategy`：

```text
manual
webhook
rolling
blue_green
canary
replace
```

### 7.20 installations

```sql
CREATE TABLE installations (
  id                    BIGSERIAL PRIMARY KEY,
  app_id                BIGINT NOT NULL REFERENCES applications(id),
  user_id               VARCHAR(128),
  device_id             VARCHAR(128) NOT NULL,
  device_key            VARCHAR(128),
  device_name           VARCHAR(128),
  platform              VARCHAR(32),
  os_version            VARCHAR(64),
  device_model          VARCHAR(128),
  package_name          VARCHAR(255),
  installed_version     VARCHAR(64),
  installed_code        BIGINT,
  build_number          BIGINT,
  resource_version      VARCHAR(64),
  channel               VARCHAR(32),
  last_seen_at          TIMESTAMP,
  last_upgrade_at       TIMESTAMP,
  last_upgrade_status   VARCHAR(64),
  last_error            TEXT,
  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  UNIQUE(app_id, device_id)
);
```

### 7.21 runtime_events

```sql
CREATE TABLE runtime_events (
  id                  BIGSERIAL PRIMARY KEY,
  app_id              BIGINT NOT NULL REFERENCES applications(id),
  release_id          BIGINT REFERENCES release_plans(id),
  batch_id            BIGINT REFERENCES release_batches(id),
  user_id             VARCHAR(128),
  device_id           VARCHAR(128) NOT NULL,
  event_category      VARCHAR(32) NOT NULL,
  event_type          VARCHAR(64) NOT NULL,
  from_version        VARCHAR(64),
  to_version          VARCHAR(64),
  from_version_code   BIGINT,
  to_version_code     BIGINT,
  package_key         VARCHAR(64),
  error_message       TEXT,
  metadata            JSONB,
  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 7.22 audit_logs

```sql
CREATE TABLE audit_logs (
  id            BIGSERIAL PRIMARY KEY,
  app_id        BIGINT REFERENCES applications(id),
  operator      VARCHAR(128) NOT NULL,
  action        VARCHAR(64) NOT NULL,
  target_type   VARCHAR(64) NOT NULL,
  target_id     VARCHAR(128),
  before_json   JSONB,
  after_json    JSONB,
  ip_address    VARCHAR(64),
  user_agent    TEXT,
  created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 8. 版本与文件规范

### 8.1 App 版本

沿用现有规范：

```text
versionName = 主版本.次版本.修订版本
versionCode = 主版本 * 10000 + 次版本 * 100 + 修订版本
buildNumber = 每次构建递增
```

示例：

```text
0.3.0-beta.1 build 48
1.0.0 stable build 100
```

### 8.2 资源版本

启动页资源增量版本：

```text
YYYYMMDD.N
```

示例：

```text
20260602.1
20260602.2
```

### 8.3 文件命名

APK：

```text
{appKey}-{versionName}-{channel}-build{buildNumber}.apk
```

资源包：

```text
{packageKey}-{resourceVersion}.zip
```

Manifest：

```text
manifest-{resourceVersion}.json
```

## 9. 存储规划

建议对象存储或文件中心目录：

```text
/srv/files/release-center/
├── artifacts/
│   └── {app_key}/
│       └── {artifact_type}/
│           └── {yyyy}/{mm}/{artifact_id}-{file_name}
├── app-releases/
│   └── {app_key}/
│       └── android/
│           └── {channel}/
│               └── {version_name}/
├── app-resources/
│   └── {app_key}/
│       ├── manifests/
│       └── packages/
│           └── {package_key}/
└── logs/
    └── builds/
```

生产建议：

- 元数据存 PostgreSQL。
- 文件存对象存储或 `/srv/files`。
- 下载走 CDN 或文件服务。
- 后台上传必须限制大小和类型。
- 内部渠道下载可加短期签名 URL。

## 10. API 设计

API 分三组：管理后台 API、CI/CLI API、客户端 API。

### 10.1 管理后台 API

```text
GET    /admin/api/release-center/overview

GET    /admin/api/apps
POST   /admin/api/apps
GET    /admin/api/apps/{app_id}
PATCH  /admin/api/apps/{app_id}

GET    /admin/api/apps/{app_id}/builds
POST   /admin/api/apps/{app_id}/builds
GET    /admin/api/builds/{build_id}
GET    /admin/api/builds/{build_id}/logs

GET    /admin/api/apps/{app_id}/artifacts
POST   /admin/api/apps/{app_id}/artifacts
GET    /admin/api/artifacts/{artifact_id}

GET    /admin/api/apps/{app_id}/releases
POST   /admin/api/apps/{app_id}/releases
GET    /admin/api/releases/{release_id}
PATCH  /admin/api/releases/{release_id}
POST   /admin/api/releases/{release_id}/submit-approval
POST   /admin/api/releases/{release_id}/approve
POST   /admin/api/releases/{release_id}/reject
POST   /admin/api/releases/{release_id}/publish
POST   /admin/api/releases/{release_id}/pause
POST   /admin/api/releases/{release_id}/recall
POST   /admin/api/releases/{release_id}/rollback
PATCH  /admin/api/releases/{release_id}/rollout

GET    /admin/api/releases/{release_id}/events
GET    /admin/api/releases/{release_id}/metrics
GET    /admin/api/audit-logs
```

### 10.2 CI/CLI API

```text
POST /api/v1/ci/builds
POST /api/v1/ci/artifacts
POST /api/v1/ci/releases
POST /api/v1/ci/releases/{release_id}/publish
GET  /api/v1/ci/resource-catalog
```

认证：

```text
Authorization: Bearer {release_center_ci_token}
```

### 10.3 客户端 API

整包检查：

```text
POST /api/v1/apps/update-check
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-001",
  "userId": "10001",
  "platform": "android",
  "packageName": "cc.misk.gamehelper",
  "versionName": "0.3.0-beta.1",
  "versionCode": 300,
  "buildNumber": 48,
  "channel": "beta",
  "osVersion": "14",
  "deviceModel": "Pixel 8"
}
```

响应：

```json
{
  "hasUpdate": true,
  "updateLevel": "recommended",
  "blockTaskExecution": false,
  "release": {
    "releaseId": "123",
    "versionName": "0.3.1-beta.1",
    "versionCode": 301,
    "buildNumber": 51,
    "title": "游戏助手更新",
    "summary": "优化识别稳定性",
    "releaseNotesMarkdown": "## 本次更新\n- 优化 OCR\n",
    "downloadUrl": "https://file.misk.cc/...",
    "sha256": "xxxx",
    "fileSize": 12345678
  }
}
```

资源检查：

```text
POST /api/v1/apps/resource-check
```

Manifest：

```text
GET /api/v1/apps/resources/{resource_id}/manifest
```

资源包下载：

```text
GET /api/v1/apps/resources/packages/{package_id}/download
```

事件上报：

```text
POST /api/v1/apps/update-events
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-001",
  "userId": "10001",
  "eventCategory": "resource",
  "eventType": "activation_failed",
  "fromVersion": "20260602.1",
  "toVersion": "20260602.2",
  "packageKey": "templates-bear",
  "errorMessage": "checksum mismatch"
}
```

## 11. 发布流程

### 11.1 APK 整包发布流程

```text
开发提交代码
  ↓
CI 构建 APK
  ↓
CI 上传 APK 并登记 build/artifact
  ↓
发布中心创建 release_plan
  ↓
编辑发布说明和升级策略
  ↓
提交审批
  ↓
审批通过
  ↓
发布到 internal
  ↓
观察安装成功率
  ↓
发布到 beta 或 stable
  ↓
逐步调整灰度比例
  ↓
全量发布或暂停/回滚
```

客户端：

```text
启动检查整包版本
  ↓
命中新版本
  ↓
展示更新说明
  ↓
下载 APK
  ↓
校验 SHA-256
  ↓
拉起系统安装器
  ↓
重启后上报 install_success
```

### 11.2 资源增量发布流程

```text
准备资源目录
  ↓
releasectl resource-pack
  ↓
校验 package_key 和 ZIP 内容
  ↓
生成 Manifest
  ↓
上传资源包和 Manifest
  ↓
创建 resource release_plan
  ↓
发布到 internal
  ↓
客户端下载到 staging
  ↓
校验、解压、原子切换 current
  ↓
上报 activation_success
  ↓
灰度扩大或自动暂停
```

资源激活失败保护：

```text
客户端上报 activation_failed
  ↓
服务端定位 app_id + resource_version
  ↓
暂停 released / rolling_out / testing 状态资源版本
  ↓
写入 paused_at
  ↓
记录 audit_logs: resource.auto_pause
  ↓
告警通知发布负责人
```

### 11.3 回滚流程

整包 APK 回滚：

- Android 不能静默降级安装。
- 回滚通常通过发布一个更高 `versionCode` 的修复包实现。
- 对严重问题版本执行召回或暂停。
- 对任务执行类 App，可通过 `block_old_versions` 阻止问题版本继续执行任务。

资源回滚：

- 服务端将资源发布切回上一稳定版本。
- 客户端检查资源时拿到旧稳定版本 Manifest。
- 客户端下载并激活旧版本，或直接切回本地保留的上一版本。

## 12. 管理后台页面

### 12.1 总览页

展示：

- 应用数量。
- 今日发布次数。
- 进行中灰度。
- 异常发布。
- 最新失败事件。
- 各应用当前稳定版本。

### 12.2 应用详情页

展示：

- 应用基本信息。
- 平台、包名、负责人。
- 渠道列表。
- 当前版本和资源版本。
- 最近构建、最近发布、版本分布。

### 12.3 构建与制品页

功能：

- 查看构建列表。
- 查看构建日志摘要。
- 上传制品。
- 查看 SHA-256、大小、下载地址。
- 标记制品归档。

### 12.4 发布管理页

功能：

- 创建发布。
- 选择构建和制品。
- 编辑发布说明。
- 设置更新级别。
- 设置灰度比例和定向规则。
- 提交审批、发布、暂停、召回、回滚。

### 12.5 资源发布页

功能：

- 查看资源版本。
- 查看 Manifest。
- 查看资源包列表。
- 查看白名单 package_key。
- 上传 ZIP。
- 发布、暂停、回滚资源版本。

### 12.6 观测页

展示：

- 更新检查量。
- 下载成功率。
- 安装成功率。
- 资源激活成功率。
- 失败原因 Top N。
- 设备版本分布。
- 灰度批次指标对比。

### 12.7 审批与审计页

功能：

- 待审批发布列表。
- 审批通过/拒绝。
- 查看操作记录。
- 按操作人、应用、动作、时间过滤。

## 13. 权限设计

角色：

| 角色 | 权限 |
|---|---|
| viewer | 查看应用、发布和指标 |
| developer | 登记构建、上传 dev/internal 制品 |
| releaser | 创建发布、编辑发布说明、调整灰度 |
| approver | 审批 beta/stable/emergency 发布 |
| admin | 应用管理、权限管理、系统配置 |
| auditor | 查看审计和导出记录 |

敏感操作：

```text
stable 发布
emergency 发布
强制更新
召回
回滚
删除/归档制品
修改审批策略
修改下载权限
```

敏感操作必须写审计，生产环境建议二次确认。

## 14. 安全方案

### 14.1 制品安全

- 上传后计算 SHA-256。
- 客户端下载后必须校验 SHA-256。
- 资源 ZIP 拒绝路径穿越、隐藏文件、符号链接、Dex、JAR、动态库、APK、AAB 和脚本。
- Manifest 增加签名。
- 客户端内置 Manifest 公钥。

### 14.2 接口安全

- 管理后台使用登录态或 SSO。
- CI/CLI 使用专用 Token。
- Token 支持应用级、环境级、操作级权限。
- 客户端 API 可使用 AppKey、设备标识、签名时间戳和限流。
- 上传接口限制文件大小、扩展名和 Content-Type。

### 14.3 下载安全

- 内部渠道使用短期签名 URL。
- stable 渠道可使用公开 CDN URL。
- 下载日志记录 device_id、release_id、artifact_id。
- 异常下载频率触发限流。

## 15. 自动化与 CLI

现有 `releasectl` 应扩展为独立发布中心 CLI。

推荐命令：

```bash
releasectl app list
releasectl build create
releasectl artifact upload
releasectl release create
releasectl release publish
releasectl release rollout
releasectl release pause
releasectl release rollback
releasectl resource catalog
releasectl resource pack
releasectl resource upload
```

CI 推荐流程：

```text
构建 APK
  ↓
计算 SHA-256
  ↓
releasectl artifact upload
  ↓
releasectl build create
  ↓
可选：releasectl release create --channel internal
```

只有受保护环境允许自动发布：

```text
GAME_HELPER_RELEASE_CENTER_TOKEN
RELEASE_CENTER_BASE_URL
```

## 16. 质量门禁

发布前检查：

- 制品 SHA-256 存在。
- APK versionCode 大于当前渠道线上版本。
- 资源版本号未复用。
- Manifest 包列表完整。
- 发布说明不为空。
- 强制更新必须填写原因。
- stable 发布必须通过审批。
- 当前不在冻结窗口。

发布中监控：

- 下载失败率。
- 校验失败率。
- 安装失败率。
- 资源激活失败率。
- 崩溃率。
- 任务执行失败率。

自动暂停条件建议：

```text
资源 activation_failed >= 3 且失败率 >= 5%
APK checksum_failed >= 3
APK install_failed >= 10 且失败率 >= 10%
关键任务执行失败率超过基线 20%
```

一期可以先实现资源 `activation_failed` 自动暂停，后续再扩展为可配置规则。

## 17. 与现有模块的迁移方案

### 17.1 当前可复用能力

已有能力：

- APK 上传和构建登记。
- 发布创建、发布、暂停、召回、回滚。
- App 检查更新。
- 资源版本上传、发布、暂停。
- 资源 Manifest 生成。
- 资源包下载。
- 资源 ZIP 白名单校验。
- CLI `resource-pack` 和 `resource-upload`。
- `activation_failed` 自动暂停。

### 17.2 迁移路径

第一步：模块命名和路由独立化。

```text
internal/modules/appreleases
  ↓
internal/modules/releasecenter
```

保留旧 API 一段时间：

```text
/admin/api/app-releases/*
/api/v1/app/*
```

新增统一 API：

```text
/admin/api/release-center/*
/api/v1/apps/*
```

第二步：抽象通用表。

- `apps` 迁移到 `applications`。
- `app_builds` 迁移到 `builds`。
- `app_releases` 迁移到 `release_plans`。
- `app_resource_versions` 迁移到 `release_plans + release_artifacts`。
- `app_upgrade_events` 和资源事件迁移到 `runtime_events`。

第三步：保留兼容适配层。

- 旧客户端继续调用旧更新接口。
- 服务端内部转到统一策略引擎。
- 新客户端使用统一 `/api/v1/apps/update-check`。

第四步：上线独立后台。

- 先只展示游戏助手。
- 验证发布流程和观测闭环。
- 再接入其他 App 或 Web/Server 发布。

### 17.3 兼容期策略

兼容期建议不少于一个稳定版本周期。兼容期内：

- 旧 API 不删除。
- 新旧 API 返回相同核心字段。
- 审计记录统一写入新表。
- 旧表可只读或通过迁移任务同步。

## 18. 落地里程碑

### M1：独立发布中心一期骨架

交付：

- 独立服务配置。
- 应用、构建、制品、发布表。
- 管理后台基础总览。
- CI/CLI 构建登记与制品上传。
- APK 发布兼容现有能力。

验收：

- 能创建应用。
- 能上传 APK 制品。
- 能创建 internal 发布。
- 客户端能检查到更新。

### M2：资源增量独立化

交付：

- 资源白名单管理。
- 资源打包上传。
- Manifest 生成。
- 资源发布和暂停。
- `activation_failed` 自动暂停。

验收：

- 能上传并发布 `templates-bear` 资源包。
- 客户端能拿到 Manifest。
- 上报 `activation_failed` 后版本自动暂停。

### M3：灰度、审批与审计

交付：

- 发布审批流。
- 灰度批次。
- 定向规则。
- 审计列表。
- 发布前检查。

验收：

- stable 发布必须审批。
- 能从 5% 调整到 20%、50%、100%。
- 每次操作都有审计记录。

### M4：观测与自动化回滚

交付：

- 发布指标面板。
- 失败原因统计。
- 自动暂停规则配置。
- 告警通知。
- 资源回滚入口。

验收：

- 能看到每个发布批次的成功率。
- 达到阈值时自动暂停并告警。
- 能一键回滚资源版本。

### M5：扩展制品类型

交付：

- Web 前端包发布。
- 服务端二进制或镜像发布登记。
- Webhook 接入部署系统。

验收：

- 发布中心能管理非 App 制品。
- Webhook 可触发外部部署流程。

## 19. 风险与应对

| 风险 | 影响 | 应对 |
|---|---|---|
| 发布中心不可用 | 客户端无法检查更新 | 客户端本地缓存上次结果；服务端多副本部署 |
| 错误强制更新 | 大量用户被阻断 | 强制更新必须审批；支持快速暂停 |
| 资源包损坏 | 客户端启动失败 | staging 下载、校验、原子切换、失败回滚 |
| 灰度命中不稳定 | 用户反复进出灰度 | 使用稳定哈希 |
| 制品被覆盖 | 无法追溯 | 制品不可变，按 sha256 唯一 |
| Token 泄露 | 非授权发布 | Token 最小权限、定期轮换、审计告警 |
| 旧客户端不兼容 | 更新能力失效 | 保留旧 API 兼容期 |

## 20. 一期最小可用版本

如果希望快速上线独立发布中心，一期最小集合如下：

```text
1. applications
2. builds
3. artifacts
4. release_plans
5. release_rules
6. installations
7. runtime_events
8. audit_logs
9. APK update-check
10. resource-check + manifest + package download
11. releasectl artifact/resource upload
12. 管理后台：应用、构建、发布、资源、事件、审计
```

必须具备的安全能力：

```text
SHA-256 校验
资源 ZIP 禁止项校验
CI Token
后台权限
敏感操作审计
资源 activation_failed 自动暂停
```

可以后置：

```text
Manifest Ed25519 签名
复杂审批流
发布冻结窗口
多租户
Web/Server 发布
自动回滚策略配置
高级报表
```

## 21. 推荐下一步

建议按以下顺序推进：

1. 先确认独立发布中心一期是否只服务游戏助手 Android。
2. 确认是否独立部署新服务，还是先在现有服务中以新路由独立。
3. 设计新表 migration，并保留旧接口兼容。
4. 将现有 `appreleases` 策略和资源打包能力迁移到 `releasecenter`。
5. 建立管理后台最小页面。
6. 跑通 internal APK 发布和资源增量发布 smoke。
7. 再开放 beta/stable 审批和灰度。
