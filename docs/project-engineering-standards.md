# 项目工程规范

更新时间：2026-06-02

本文用于约束发布中心后续研发方式。目标是让代码优雅、易扩展、方便调试、性能可控，并且每个功能点和里程碑都有清晰 Git 留痕、文档留档和交接依据。

## 1. 总体原则

1. 优先保持系统边界清晰：项目、制品、发布、部署、观测、审计、系统管理各自独立演进。
2. 优先复用现有模式：Go 后端沿用 service / store / handler 分层，前端沿用 `api/`、`pages/`、`components/` 组织。
3. 先做可验证闭环，再做抽象扩展。抽象必须服务真实复用，不能为了“看起来通用”增加复杂度。
4. 代码默认可测试、可观测、可回滚。新功能要能说明如何验证，失败时要能定位。
5. 每完成一个功能点或里程碑就提交一次 Git commit，避免大量无边界改动混在一起。
6. 文档必须跟随代码同步更新，任何人接手时都应能通过文档了解当前状态、启动方式、验证方式和下一步。
7. 每次交付都要默认进入“可交接”状态：代码、文档、验证结果和下一步必须对齐，不能只在聊天记录里保留关键信息。

## 2. 代码优雅

后端：

- Handler 只负责 HTTP 参数解析、认证上下文传递和响应，不写复杂业务规则。
- Service 负责业务流程、状态流转、校验和审计触发。
- Store / Repository 负责持久化，不把 HTTP 结构泄露到数据库层。
- Model / DTO 命名要表达业务含义，避免 `Data`、`Info`、`Tmp` 这类模糊名字。
- 错误要包含稳定 code 和可读 message，外部响应不要暴露数据库细节。
- 状态流转要集中管理，避免多个 handler 各自拼状态字符串。

前端：

- 页面负责组合数据和交互，通用控件放 `components/`。
- API 调用集中放 `web/src/api/`，页面不直接散写 fetch。
- UI 状态要区分 loading、empty、error、success。
- 表格、弹窗、确认框、危险操作按钮要保持一致交互。
- 不在页面里硬编码大量 mock 数据；mock 应集中管理，并标注后续真实 API 对接点。

文档：

- 每个模块至少有目标、边界、主要接口、验证方式。
- 重要决策写入文档，不只留在聊天或临时备注里。
- 文档要写清楚当前状态：已完成、待验收、待实现。
- 完成功能点、里程碑、架构调整、上线验证或事故处理后，必须及时留档。
- 文档更新要和代码提交放在同一个功能 commit 或紧邻的 docs commit 中，避免代码与文档长期不一致。

## 3. 留档与交接准备

项目随时要保持可交接状态。任何开发者只看仓库文档、提交记录和验证日志，就应该能判断系统做到哪、怎么跑、怎么验、下一步做什么。

### 3.0 日常协作规则

- 开始一个功能点前，先确认对应计划文档、实现说明或交接文档中的目标和边界。
- 推进过程中发现范围变化、风险、临时方案或验证限制，必须当天写入对应文档。
- 完成一个功能点或里程碑后，必须同步更新实现状态、验证命令、验证结果和剩余事项。
- 阶段性交付前，必须确认文档能让后续接手人独立启动服务、复现验证、判断哪些能力已完成、哪些能力待验收。
- 不把关键信息只留在聊天记录、个人笔记或本地临时文件中。

### 3.1 必须留档的场景

以下场景完成后必须更新文档：

- 完成一个功能点。
- 完成一个里程碑。
- 新增或调整数据库 migration。
- 新增或调整 API。
- 新增前端页面、菜单或关键交互。
- 调整认证、权限、发布策略、质量门禁。
- 完成一次生产或本地 smoke。
- 遇到重要问题、风险、临时绕行方案。
- 修改部署方式、环境变量、启动命令、CI/CD 流程。

### 3.2 留档内容

每次留档至少包含：

```text
当前状态：已完成 / 待验收 / 待实现 / 有风险
改动范围：涉及模块、文件、API、表结构
验证结果：执行过的命令、smoke 结果、未验证原因
运行方式：启动命令、环境变量、依赖服务
交接提示：下一步建议、风险点、回滚方式
```

原则上，完成一个功能点时不能只提交代码。对应文档至少要说明本次做了什么、怎么验证、还有什么没做；如果本次没有文档变化，也要在 commit message 或交接文档里说明原因。

阶段收尾时，交接信息至少要覆盖：

- 最新分支、最新 commit 和是否已 push。
- 本次完成的功能点、修复点和影响范围。
- 最近一次验证命令及结果。
- 未验证事项、外部依赖、账号或 token 映射说明，不记录明文密钥。
- 下一步优先级和可能的回滚方式。

### 3.3 文档位置

推荐按用途更新：

- `docs/app-release-center-implementation.md`：当前实现状态、验证结果、推进计划。
- `docs/lightweight-release-center-plan.md`：轻量闭环方案和近期路线。
- `docs/independent-release-center-plan.md`：独立发布中心长期方案。
- `docs/phase1-closure-audit.md`：阶段闭环审计和验收结果。
- `docs/*handoff*.md`：交接说明、当前阻塞、下一步。
- `docs/project-engineering-standards.md`：工程规范和协作规则。

### 3.4 交接文档要求

阶段交接文档必须写清楚：

- 当前分支和最新 commit。
- 已完成事项。
- 未完成事项。
- 外部依赖和账号 / token / 环境变量说明，不写明文密钥。
- 本地启动步骤。
- 生产部署步骤。
- 验证命令和最近一次结果。
- 已知风险。
- 下一步优先级。

交接文档要保持“接手即能工作”的颗粒度：新人不需要翻聊天记录，就能知道服务如何启动、页面入口在哪、哪些能力已验收、哪些能力只是 demo 或待接真实环境。

### 3.5 交接前检查

交接或结束一个阶段前必须检查：

```bash
git status --short
git log --oneline -5
```

并确认：

- 工作区没有遗漏的关键修改。
- 已完成的功能点已有 commit。
- 文档已记录当前状态。
- 文档中的启动命令、环境变量、验证命令和最新限制仍然准确。
- 未验证事项已经明确标注。
- 后续接手人能按文档复现启动和验证流程。

## 4. 易扩展

通用发布中心必须按发布类型扩展，而不是把所有能力塞进 App 发版模块：

```text
release_type:
  android_full
  android_resource
  web_bundle
  docs_site
  server_binary
  config_bundle
  docker_image
  cloudflare_worker
```

扩展要求：

- 新发布类型优先复用 artifact、release、deployment、audit、quality 这几类通用对象。
- 特定领域逻辑放到独立 adapter 或 service，不污染通用发布流程。
- 数据库字段优先稳定结构；确实需要扩展时再用 `metadata JSONB`，不能所有核心字段都塞进 JSON。
- API 路径按资源组织，不按页面组织。
- 后台菜单按项目、制品、发布、部署、观测、审计、系统管理组织，App 更新只是一个发布类型。

新增模块时至少回答：

```text
它属于哪个边界？
是否能复用已有表和 API？
状态流转是什么？
如何审计？
如何验证？
如何回滚？
```

## 5. 方便调试

后端调试要求：

- 每个请求要有 request_id，日志和错误响应中能关联。
- 关键操作必须记录 actor、action、target_type、target_id。
- 外部 CI、Webhook、部署回调要保存原始摘要和失败原因。
- 长流程要拆出步骤日志，例如上传、校验、创建发布、发布、部署回调。
- 错误信息要区分用户可修复错误和系统错误。

前端调试要求：

- API 错误页面展示 message、code、request_id。
- 重要操作前有确认摘要，失败后保留用户输入。
- 页面状态不应静默失败，至少展示错误提示和重试入口。
- 开发环境可以启用 mock，但页面上要能区分 mock 和真实数据。

运维调试要求：

- 服务必须提供 `/healthz` 和 `/readyz`。
- 生产部署文档要包含启动命令、环境变量、日志位置、迁移命令和回滚步骤。
- Smoke 脚本要能复跑，并说明依赖的环境变量。

## 6. 性能要求

后端：

- 列表接口必须分页，默认限制返回数量。
- 常用查询要有索引，尤其是 tenant、app/project、status、created_at、event_time。
- 大文件上传和下载不能经过不必要的内存拷贝。
- ZIP 校验要限制文件数量、单文件大小、总大小和压缩炸弹风险。
- 质量统计要避免每次全表扫描；数据量上来后需要按时间窗口或预聚合优化。
- 外部调用必须有 timeout，不能无限等待 CI、Webhook 或部署平台。

前端：

- 大表格要分页或虚拟列表，避免一次渲染过多行。
- 轮询要有间隔、停止条件和页面隐藏处理。
- 页面初始加载只请求首屏必要数据。
- 打包产物要避免无意义大依赖，新增依赖前说明用途。

数据库：

- Migration 必须可重复执行或明确只执行一次。
- 高风险 migration 要先写回滚方案。
- 不在高频路径使用无索引模糊查询。

## 7. 测试和验证

每个功能点至少满足一种验证：

- 单元测试：业务规则、状态流转、校验逻辑。
- Handler 测试：请求参数、认证、响应格式。
- Store 测试：关键 SQL、迁移、约束。
- 前端构建：`cd web && npm run build`。
- 后端测试：`go test ./...`。
- Smoke：真实 DB、发布、下载、事件、自动暂停等闭环。

推荐提交前检查：

```bash
go test ./...
go build -buildvcs=false ./cmd/server ./cmd/releasectl ./cmd/migrate
cd web && npm run build
make smoke-db
```

如果某项没有运行，提交说明或交付说明里要写清楚原因。

## 8. Git 提交规范

### 8.1 提交频率

必须按功能点或里程碑提交：

- 完成一个独立功能点后提交。
- 完成一个可验证里程碑后提交。
- 完成一次文档方案调整后提交。
- 修复一个明确 bug 后提交。

不要把无关改动混在一个 commit 中。例如 RBAC schema、前端菜单重构、资源上传 bug 修复应拆成不同提交。

### 8.2 提交格式

统一使用 Conventional Commits 风格：

```text
<type>(optional-scope): <summary>
```

常用 type：

```text
feat      新功能
fix       修复 bug
docs      文档
refactor  重构，不改变外部行为
test      测试
perf      性能优化
chore     构建、依赖、脚手架、杂项
ci        CI/CD 配置
style     纯格式调整
```

示例：

```text
feat(rbac): add role permission persistence
fix(release): reject resource package with hidden files
docs(plan): update multi-artifact release roadmap
perf(events): add indexes for quality metrics queries
test(appreleases): cover activation failed auto pause
```

### 8.3 提交内容要求

每个 commit 应满足：

- 标题说明“做了什么”，不要只写 `update`、`fix`、`wip`。
- 只包含一个主题。
- 代码和文档同步更新。
- 能通过对应层级验证，或明确说明未验证原因。
- 涉及功能点或里程碑时，必须同步更新实现文档、计划文档或交接文档。
- 涉及页面、API、权限、数据结构、部署方式或运行参数时，必须同步更新对应说明和交接提示。
- 涉及临时方案、风险、阻塞时，必须写明后续处理建议。
- 不提交本地临时文件、日志、密钥、构建缓存。

### 8.4 推荐提交节奏

RBAC 里程碑示例：

```text
docs(rbac): define system management scope
feat(rbac): add users roles permissions migrations
feat(rbac): add system management admin APIs
feat(rbac): enforce admin permission middleware
feat(web): connect system management pages to APIs
test(rbac): cover role permission checks
```

通用发布中心里程碑示例：

```text
docs(plan): define generic release center milestones
feat(release): add generic artifact model
feat(release): add deployment target records
feat(web): replace app-only navigation with release center sections
test(release): cover deployment callback flow
```

## 9. 分支和 Push 规范

- `main` 保持可构建、可运行、文档一致。
- 大功能建议使用短生命周期分支，完成后合并。
- 直接推 `main` 时必须确保工作区干净、提交主题清晰。
- Push 前至少执行 `git status` 和必要验证命令。
- 遇到远端有更新，先 `git pull --rebase`，解决冲突后再 push。
- Push 前确认需要交接的文档已经更新，避免代码先走、文档滞后。
- Push 前确认阶段文档能回答三件事：当前做到哪、怎么验证、下一步做什么。

## 10. 评审清单

合并或交付前检查：

- 是否符合模块边界？
- 是否有不必要的重复或过度抽象？
- 是否有测试或 smoke 验证？
- 是否记录审计？
- 是否有可定位错误信息？
- 是否考虑分页、索引、timeout、文件大小限制？
- 是否更新相关文档？
- 是否留下当前状态、验证结果和下一步？
- 是否能让其他人随时接手？
- 是否按功能点提交？
- 是否把本次未验证原因、环境限制或外部依赖写清楚？

这份规范后续如果和实际工程冲突，以更安全、更可验证、更易维护的做法为准，并同步更新本文。
