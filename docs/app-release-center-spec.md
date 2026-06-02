# App 发版中心与启动页增量更新规范

> 适用项目：游戏助手  
> 适用阶段：开发测试阶段 → 内测阶段 → 正式发布阶段  
> 适用客户端：Android 优先，后续可扩展 iOS  
> 文档目标：统一 App 整包发版、启动页资源增量更新、版本命名、后台管理、灰度发布、更新说明、回滚与审计规范。

---

# 1. 设计目标

App 发版不能只保存一个 APK 下载链接。对于游戏辅助类 App，客户端同时承担登录、悬浮窗、无障碍服务、截图、OCR、OpenCV、任务轮询和本机脚本执行，因此必须将以下能力串成一套完整链路：

```text
代码提交
  ↓
构建 APK
  ↓
生成构建记录
  ↓
创建发布版本
  ↓
后台编辑更新内容
  ↓
发布到指定渠道
  ↓
App 启动检查整包版本
  ↓
App 启动检查资源增量版本
  ↓
设备上报安装结果和资源版本
  ↓
任务拉取前再次校验兼容性
```

核心目标：

1. 区分开发构建、测试构建和正式发布。
2. 支持 APK 整包更新。
3. 支持启动页资源增量更新。
4. 支持后台编辑版本更新内容。
5. 支持普通更新、推荐更新、强制更新。
6. 支持指定用户、指定设备、测试组和百分比灰度。
7. 支持暂停发布、撤回版本和自动回滚。
8. 支持查看设备版本分布和升级失败原因。
9. 支持任务执行前校验 App 版本和资源版本。
10. 保留完整操作记录，方便定位问题。

---

# 2. 更新类型定义

系统统一支持两层更新。

## 2.1 App 整包更新

用于更新 Android 原生代码和基础能力。

适用内容：

- Kotlin / Java 代码
- Android 权限
- 无障碍服务
- 悬浮窗逻辑
- OCR / OpenCV 引擎
- 本机任务执行器
- 网络协议基础层
- 本地数据库结构
- 安全校验逻辑
- 登录和账号体系
- 原生 UI 页面

处理逻辑：

```text
检查 App 版本
  ↓
发现新版本
  ↓
下载安装 APK
  ↓
校验 SHA-256
  ↓
拉起系统安装器
  ↓
用户确认安装
  ↓
重新启动 App
```

整包更新仍然需要用户安装 APK。

---

## 2.2 启动页资源增量更新

用于快速更新运行资源，不重新安装 APK。

适用内容：

- 场景配置
- 任务模板
- 点击步骤
- 坐标规则
- OCR 关键词库
- OpenCV 模板图片
- 活动配置
- 功能开关
- 启动页图片
- UI 静态资源
- 页面文案
- 服务端下发的 JSON 配置

不允许通过增量更新替换：

```text
classes.dex
plugin.dex
xxx.jar
libxxx.so
可执行脚本引擎
原生动态库
未经过安全审计的执行代码
```

原则：

> 增量更新只更新“数据和资源”，不更新“原生可执行代码”。

---

## 2.3 APK 差分补丁更新

APK 差分补丁不是一期必做内容。

逻辑：

```text
旧 APK
  +
差分补丁 patch
  ↓
合成新 APK
  ↓
校验新 APK
  ↓
拉起系统安装器
```

它只能减少下载流量，不能绕过 Android 安装流程。

建议：

- 一期：不做 APK 差分补丁。
- 二期：当 APK 体积明显增大、设备数量增加后再评估。
- 当前优先实现：整包 APK 更新 + 启动页资源增量更新。

---

# 3. 版本命名规范

必须区分：

- 用户看到的版本号
- 程序比较用版本号
- 构建编号
- 资源版本号
- 发布渠道
- 文件名

---

## 3.1 App 版本号：`versionName`

采用语义化版本：

```text
主版本.次版本.修订版本
```

示例：

```text
0.1.0
0.1.1
0.2.0
1.0.0
1.1.0
1.1.1
```

规则：

| 变更类型 | 示例 | 说明 |
|---|---|---|
| 主版本升级 | `1.0.0 → 2.0.0` | 大改版、不兼容变更 |
| 次版本升级 | `1.2.0 → 1.3.0` | 新增功能、兼容旧版本 |
| 修订版本升级 | `1.3.0 → 1.3.1` | Bug 修复、小范围优化 |

当前开发阶段建议：

```text
0.x.y
```

正式对外发布后使用：

```text
1.0.0
```

---

## 3.2 程序比较版本：`versionCode`

Android 使用整数比较版本新旧，必须持续递增，不能重复。

推荐规则：

```text
versionCode = 主版本 * 10000 + 次版本 * 100 + 修订版本
```

示例：

| versionName | versionCode |
|---|---:|
| `0.1.0` | `100` |
| `0.1.1` | `101` |
| `0.2.0` | `200` |
| `1.0.0` | `10000` |
| `1.3.2` | `10302` |
| `2.0.0` | `20000` |

如果同一个版本号需要重复打包，使用独立的 `buildNumber` 区分，不要重复使用相同构建记录。

---

## 3.3 构建编号：`buildNumber`

每次 CI 或手工打包都递增：

```text
1
2
3
...
128
129
```

构建编号用于区分：

```text
0.2.0 build 37
0.2.0 build 38
```

即使 `versionName` 不变，只要重新打包，`buildNumber` 也必须变化。

---

## 3.4 发布渠道：`channel`

统一使用以下渠道：

| 渠道 | 用途 | 是否面向普通用户 |
|---|---|---|
| `dev` | 开发联调包 | 否 |
| `internal` | 内部测试包 | 否 |
| `beta` | 小范围测试用户 | 否 |
| `stable` | 正式版本 | 是 |
| `emergency` | 紧急修复 | 视情况 |

当前开发测试阶段建议主要使用：

```text
dev
internal
beta
```

正式上线后再开放：

```text
stable
emergency
```

---

## 3.5 预发布标签

开发测试阶段允许使用：

```text
0.2.0-dev.3
0.2.0-internal.5
0.2.0-beta.2
```

正式发布使用：

```text
1.0.0
1.0.1
1.1.0
```

建议规则：

| 场景 | 示例 |
|---|---|
| 开发联调 | `0.3.0-dev.7` |
| 内部测试 | `0.3.0-internal.2` |
| 外部测试 | `0.3.0-beta.1` |
| 正式发布 | `1.0.0` |
| 紧急修复 | `1.0.1` |

---

## 3.6 资源版本号：`resourceVersion`

启动页增量资源使用独立版本号，不与 APK 强绑定。

推荐格式：

```text
YYYYMMDD.N
```

示例：

```text
20260601.1
20260601.2
20260602.1
```

解释：

- `20260601`：发布日期
- `.1`：当天第 1 个资源版本
- `.2`：当天第 2 个资源版本

资源版本可以频繁变化：

```text
App 版本：0.3.0-beta.2
资源版本：20260601.4
```

---

# 4. 文件命名规范

## 4.1 APK 文件名

统一格式：

```text
{appKey}-{versionName}-{channel}-build{buildNumber}.apk
```

示例：

```text
game-helper-0.3.0-dev.7-dev-build37.apk
game-helper-0.3.0-internal.2-internal-build42.apk
game-helper-0.3.0-beta.1-beta-build48.apk
game-helper-1.0.0-stable-build100.apk
game-helper-1.0.1-emergency-build103.apk
```

如果 `versionName` 已经包含渠道标签，也可以简化：

```text
game-helper-0.3.0-beta.1-build48.apk
game-helper-1.0.0-build100.apk
```

建议后端统一生成文件名，避免人工填写。

---

## 4.2 AAB 文件名

正式提交 Google Play 时：

```text
{appKey}-{versionName}-{channel}-build{buildNumber}.aab
```

示例：

```text
game-helper-1.0.0-stable-build100.aab
```

---

## 4.3 资源包文件名

统一格式：

```text
{packageKey}-{resourceVersion}.zip
```

示例：

```text
scenes-core-20260601.1.zip
templates-bear-20260601.2.zip
ocr-dictionary-20260601.2.zip
ui-assets-20260601.3.zip
feature-flags-20260601.3.zip
```

---

## 4.4 Manifest 文件名

统一格式：

```text
manifest-{resourceVersion}.json
```

示例：

```text
manifest-20260601.3.json
```

---

# 5. 服务端目录规范

结合现有 `/srv/files` 文件中心，建议新增以下目录。

```text
/srv/files/
├── app-releases/
│   └── game-helper/
│       └── android/
│           ├── dev/
│           │   └── 0.3.0-dev.7/
│           │       ├── game-helper-0.3.0-dev.7-build37.apk
│           │       ├── sha256.txt
│           │       └── release.json
│           ├── internal/
│           ├── beta/
│           ├── stable/
│           └── emergency/
│
└── app-resources/
    └── game-helper/
        ├── manifests/
        │   ├── manifest-20260601.1.json
        │   ├── manifest-20260601.2.json
        │   └── manifest-20260601.3.json
        │
        └── packages/
            ├── scenes-core/
            │   └── scenes-core-20260601.1.zip
            ├── scenes-events/
            │   └── scenes-events-20260601.2.zip
            ├── templates-common/
            │   └── templates-common-20260601.1.zip
            ├── templates-bear/
            │   └── templates-bear-20260601.2.zip
            ├── ocr-dictionary/
            │   └── ocr-dictionary-20260601.2.zip
            ├── ui-assets/
            │   └── ui-assets-20260601.3.zip
            └── feature-flags/
                └── feature-flags-20260601.3.zip
```

外部访问地址示例：

```text
https://file.misk.cc/app-releases/game-helper/android/beta/0.3.0-beta.1/game-helper-0.3.0-beta.1-build48.apk

https://file.misk.cc/app-resources/game-helper/manifests/manifest-20260601.3.json

https://file.misk.cc/app-resources/game-helper/packages/templates-bear/templates-bear-20260601.2.zip
```

---

# 6. 服务端数据模型

建议将“构建记录”和“发布版本”分开。

原因：

- 构建成功不代表一定发布。
- 同一个构建可以先发 internal，再发 beta。
- 构建失败也需要保留日志。
- 发布后需要单独记录灰度策略、更新说明和操作记录。

---

## 6.1 App 基础表：`apps`

```sql
CREATE TABLE apps (
  id                  BIGSERIAL PRIMARY KEY,
  app_key             VARCHAR(64) UNIQUE NOT NULL,
  name                VARCHAR(100) NOT NULL,
  platform            VARCHAR(20) NOT NULL,
  package_name        VARCHAR(255) NOT NULL,
  description         TEXT,
  enabled             BOOLEAN NOT NULL DEFAULT TRUE,
  created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

示例：

```text
app_key: game-helper-android
name: 游戏助手
platform: android
package_name: cc.misk.gamehelper
```

---

## 6.2 构建记录表：`app_builds`

```sql
CREATE TABLE app_builds (
  id                  BIGSERIAL PRIMARY KEY,
  app_id              BIGINT NOT NULL REFERENCES apps(id),

  version_name        VARCHAR(64) NOT NULL,
  version_code        BIGINT NOT NULL,
  build_number        BIGINT NOT NULL,
  channel             VARCHAR(32) NOT NULL,

  git_commit          VARCHAR(64),
  git_branch          VARCHAR(128),
  build_environment   VARCHAR(32),
  build_status        VARCHAR(32) NOT NULL,

  artifact_type       VARCHAR(32),
  artifact_path       TEXT,
  artifact_size       BIGINT,
  sha256              VARCHAR(64),

  min_os_version      VARCHAR(32),
  built_by            VARCHAR(128),
  build_log_path      TEXT,

  created_at          TIMESTAMP NOT NULL DEFAULT NOW(),

  UNIQUE(app_id, build_number)
);
```

`build_status`：

```text
building
success
failed
archived
```

---

## 6.3 App 发布表：`app_releases`

```sql
CREATE TABLE app_releases (
  id                    BIGSERIAL PRIMARY KEY,
  app_id                BIGINT NOT NULL REFERENCES apps(id),
  build_id              BIGINT NOT NULL REFERENCES app_builds(id),

  channel               VARCHAR(32) NOT NULL,
  status                VARCHAR(32) NOT NULL,

  title                 VARCHAR(255) NOT NULL,
  summary               TEXT,
  release_notes_markdown TEXT,
  upgrade_message       TEXT,

  update_level          VARCHAR(32) NOT NULL DEFAULT 'normal',
  rollout_percentage    INT NOT NULL DEFAULT 100,

  min_supported_code    BIGINT,
  block_old_versions    BOOLEAN NOT NULL DEFAULT FALSE,

  published_at          TIMESTAMP,
  paused_at             TIMESTAMP,

  created_by            VARCHAR(128),
  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`status`：

```text
draft
testing
scheduled
rolling_out
released
paused
recalled
archived
```

`update_level`：

```text
normal
recommended
forced
```

---

## 6.4 App 发布说明表：`app_release_notes`

更新说明必须支持后台编辑，并保留历史。

```sql
CREATE TABLE app_release_notes (
  id                  BIGSERIAL PRIMARY KEY,
  release_id          BIGINT NOT NULL REFERENCES app_releases(id),

  language            VARCHAR(16) NOT NULL DEFAULT 'zh-CN',
  title               VARCHAR(255) NOT NULL,
  summary             TEXT,
  content_markdown    TEXT NOT NULL,

  edited_by           VARCHAR(128),
  created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),

  UNIQUE(release_id, language)
);
```

说明：

- 后台使用 Markdown 编辑器。
- App 客户端可以直接渲染 Markdown，或者由服务端返回结构化内容。
- 后续可增加 `en-US` 等多语言内容。
- 已发布版本修改说明时必须保留审计记录。

示例内容：

```markdown
## 本次更新

- 新增悬浮窗快捷控制。
- 优化随缘打熊场景识别。
- 修复部分分辨率下 OCR 识别异常。
- 提升任务执行稳定性。

## 注意事项

首次启动后会自动更新运行资源，请保持网络连接。
```

---

## 6.5 App 灰度规则表：`app_release_rules`

```sql
CREATE TABLE app_release_rules (
  id                  BIGSERIAL PRIMARY KEY,
  release_id          BIGINT NOT NULL REFERENCES app_releases(id),

  rule_type           VARCHAR(32) NOT NULL,
  rule_value          TEXT NOT NULL,

  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
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
```

灰度百分比建议使用稳定哈希：

```text
hash(user_id + release_id) % 100 < rollout_percentage
```

这样进入灰度名单的用户不会频繁变化。

---

## 6.6 设备安装记录表：`app_installations`

```sql
CREATE TABLE app_installations (
  id                    BIGSERIAL PRIMARY KEY,
  app_id                BIGINT NOT NULL REFERENCES apps(id),

  user_id               BIGINT,
  device_id             VARCHAR(128) NOT NULL,
  device_name           VARCHAR(128),
  platform              VARCHAR(20),
  os_version            VARCHAR(32),
  device_model          VARCHAR(128),

  installed_version     VARCHAR(64),
  installed_code        BIGINT,
  build_number          BIGINT,

  resource_version      VARCHAR(64),

  last_seen_at          TIMESTAMP,
  last_upgrade_at       TIMESTAMP,
  last_upgrade_status   VARCHAR(32),
  last_error            TEXT,

  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP NOT NULL DEFAULT NOW(),

  UNIQUE(app_id, device_id)
);
```

---

## 6.7 App 升级事件表：`app_upgrade_events`

```sql
CREATE TABLE app_upgrade_events (
  id                  BIGSERIAL PRIMARY KEY,
  app_id              BIGINT NOT NULL REFERENCES apps(id),
  release_id          BIGINT REFERENCES app_releases(id),

  user_id             BIGINT,
  device_id           VARCHAR(128) NOT NULL,

  from_version_code   BIGINT,
  to_version_code     BIGINT,

  event_type          VARCHAR(32) NOT NULL,
  error_message       TEXT,

  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`event_type`：

```text
update_detected
download_started
download_completed
checksum_failed
install_started
install_failed
install_success
app_started
rollback_detected
```

---

# 7. 增量资源数据模型

---

## 7.1 资源版本表：`app_resource_versions`

```sql
CREATE TABLE app_resource_versions (
  id                    BIGSERIAL PRIMARY KEY,
  app_id                BIGINT NOT NULL REFERENCES apps(id),

  resource_version      VARCHAR(64) NOT NULL,
  channel               VARCHAR(32) NOT NULL DEFAULT 'stable',
  status                VARCHAR(32) NOT NULL DEFAULT 'draft',

  min_app_version_code  BIGINT,
  max_app_version_code  BIGINT,

  update_level          VARCHAR(32) NOT NULL DEFAULT 'normal',
  title                 VARCHAR(255) NOT NULL,
  summary               TEXT,
  release_notes_markdown TEXT,

  rollout_percentage    INT NOT NULL DEFAULT 100,

  manifest_url          TEXT NOT NULL,
  total_size            BIGINT NOT NULL DEFAULT 0,

  published_at          TIMESTAMP,
  created_by            VARCHAR(128),
  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMP NOT NULL DEFAULT NOW(),

  UNIQUE(app_id, resource_version)
);
```

`update_level`：

```text
normal
recommended
forced
```

---

## 7.2 资源包表：`app_resource_packages`

```sql
CREATE TABLE app_resource_packages (
  id                    BIGSERIAL PRIMARY KEY,
  resource_version_id   BIGINT NOT NULL REFERENCES app_resource_versions(id),

  package_key           VARCHAR(64) NOT NULL,
  package_type          VARCHAR(32) NOT NULL,

  file_url              TEXT NOT NULL,
  file_size             BIGINT NOT NULL,
  sha256                VARCHAR(64) NOT NULL,

  created_at            TIMESTAMP NOT NULL DEFAULT NOW(),

  UNIQUE(resource_version_id, package_key)
);
```

推荐资源包：

| `package_key` | 内容 |
|---|---|
| `scenes-core` | 基础场景配置 |
| `scenes-events` | 活动脚本配置 |
| `templates-common` | 通用识图模板 |
| `templates-bear` | 打熊模板 |
| `ocr-dictionary` | OCR 关键词库 |
| `ui-assets` | 启动页和 UI 静态资源 |
| `feature-flags` | 功能开关 |

---

## 7.3 资源更新事件表：`app_resource_update_events`

```sql
CREATE TABLE app_resource_update_events (
  id                  BIGSERIAL PRIMARY KEY,
  app_id              BIGINT NOT NULL REFERENCES apps(id),

  user_id             BIGINT,
  device_id           VARCHAR(128) NOT NULL,

  from_version        VARCHAR(64),
  to_version          VARCHAR(64),

  event_type          VARCHAR(32) NOT NULL,
  package_key         VARCHAR(64),
  error_message       TEXT,

  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

`event_type`：

```text
manifest_checked
update_detected
download_started
download_completed
checksum_failed
extract_failed
activation_success
activation_failed
rollback_success
```

---

# 8. Manifest 规范

Manifest 是客户端启动页增量更新的核心。

文件示例：

```json
{
  "appKey": "game-helper-android",
  "resourceVersion": "20260601.3",
  "channel": "beta",
  "updateLevel": "normal",
  "minAppVersionCode": 300,
  "title": "正在更新运行资源",
  "summary": "优化打熊识别和 OCR 关键词库",
  "packages": [
    {
      "packageKey": "templates-bear",
      "packageType": "zip",
      "url": "https://file.misk.cc/app-resources/game-helper/packages/templates-bear/templates-bear-20260601.3.zip",
      "size": 1848293,
      "sha256": "xxxxxx"
    },
    {
      "packageKey": "ocr-dictionary",
      "packageType": "zip",
      "url": "https://file.misk.cc/app-resources/game-helper/packages/ocr-dictionary/ocr-dictionary-20260601.3.zip",
      "size": 48392,
      "sha256": "xxxxxx"
    }
  ]
}
```

建议增加 Manifest 签名：

```json
{
  "signatureAlgorithm": "ed25519",
  "signature": "base64-signature"
}
```

客户端内置公钥，下载 Manifest 后先验签，再下载资源包。

---

# 9. App 启动更新流程

App 启动时先检查整包 APK，再检查资源增量。

```text
App 启动
  ↓
显示启动页
  ↓
读取本地 App 版本和资源版本
  ↓
请求 /api/v1/app/update-check
  ↓
判断 APK 是否需要更新
  ├── 强制整包更新
  │     ↓
  │   展示更新内容
  │     ↓
  │   下载 APK
  │     ↓
  │   校验 SHA-256
  │     ↓
  │   拉起系统安装器
  │
  └── App 版本可用
        ↓
      请求 /api/v1/app/resource-check
        ↓
      判断资源是否需要更新
        ├── 无更新
        │     ↓
        │   进入首页
        │
        ├── 普通资源更新
        │     ↓
        │   可后台下载
        │     ↓
        │   下次启动生效
        │
        └── 强制资源更新
              ↓
            启动页显示下载进度
              ↓
            下载到 staging
              ↓
            校验 SHA-256
              ↓
            解压到 staging
              ↓
            校验文件清单
              ↓
            原子切换 current 指针
              ↓
            进入首页
```

---

# 10. 客户端本地目录规范

```text
files/
└── runtime/
    ├── current.json
    ├── versions/
    │   ├── 20260601.1/
    │   │   ├── manifest.json
    │   │   ├── scenes/
    │   │   ├── templates/
    │   │   ├── ocr/
    │   │   └── ui/
    │   └── 20260601.2/
    ├── staging/
    └── rollback/
```

`current.json`：

```json
{
  "resourceVersion": "20260601.2",
  "activatedAt": "2026-06-01T15:30:00+08:00"
}
```

更新时不要直接覆盖当前目录。

正确流程：

```text
下载到 staging
  ↓
校验压缩包
  ↓
解压到 staging
  ↓
校验文件清单
  ↓
写入新版本目录
  ↓
原子切换 current.json
  ↓
保留旧版本用于回滚
```

至少保留最近两个资源版本。

---

# 11. 更新策略规范

## 11.1 APK 整包更新策略

| 类型 | 行为 |
|---|---|
| 普通更新 | 用户可稍后更新 |
| 推荐更新 | 每隔一段时间再次提示 |
| 强制更新 | 不更新无法继续进入核心功能 |
| 阻止旧版任务执行 | 可以进入 App，但不能拉取新任务 |

---

## 11.2 资源增量更新策略

| 场景 | 行为 |
|---|---|
| 普通资源更新，小于 3 MB | 启动页下载或后台下载 |
| 普通资源更新，大于 3 MB | 进入首页后后台下载，下次启动生效 |
| 强制资源更新 | 启动页阻塞下载 |
| 网络超时 | 使用本地资源进入首页 |
| SHA-256 校验失败 | 删除临时文件，继续使用旧版本 |
| 解压失败 | 删除 staging，继续使用旧版本 |
| 新资源启动失败 | 自动回滚上一个资源版本 |
| 正在执行任务 | 不切换资源，任务结束后再激活 |

---

# 12. 接口设计

---

## 12.1 检查整包更新

```http
POST /api/v1/app/update-check
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-abc-123",
  "userId": 10001,
  "platform": "android",
  "osVersion": "14",
  "deviceModel": "Pixel 8",
  "versionName": "0.3.0-beta.1",
  "versionCode": 300,
  "buildNumber": 48,
  "channel": "beta"
}
```

响应：

```json
{
  "hasUpdate": true,
  "updateLevel": "recommended",
  "blockTaskExecution": false,
  "release": {
    "versionName": "0.3.0-beta.2",
    "versionCode": 301,
    "buildNumber": 52,
    "title": "测试版本更新",
    "summary": "优化任务执行稳定性",
    "releaseNotesMarkdown": "## 本次更新\n\n- 优化悬浮窗稳定性\n- 修复 OCR 异常",
    "downloadUrl": "https://file.misk.cc/app-releases/game-helper/android/beta/0.3.0-beta.2/game-helper-0.3.0-beta.2-build52.apk",
    "sha256": "xxxxxx",
    "fileSize": 48392184
  }
}
```

---

## 12.2 检查资源增量更新

```http
POST /api/v1/app/resource-check
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-abc-123",
  "userId": 10001,
  "appVersionCode": 301,
  "resourceVersion": "20260601.1",
  "channel": "beta"
}
```

响应：

```json
{
  "hasUpdate": true,
  "updateLevel": "normal",
  "latestResourceVersion": "20260601.3",
  "title": "正在更新运行资源",
  "summary": "优化打熊识别和 OCR 关键词库",
  "totalSize": 1896685,
  "packages": [
    {
      "packageKey": "templates-bear",
      "url": "https://file.misk.cc/app-resources/game-helper/packages/templates-bear/templates-bear-20260601.3.zip",
      "size": 1848293,
      "sha256": "xxxxxx"
    },
    {
      "packageKey": "ocr-dictionary",
      "url": "https://file.misk.cc/app-resources/game-helper/packages/ocr-dictionary/ocr-dictionary-20260601.3.zip",
      "size": 48392,
      "sha256": "xxxxxx"
    }
  ]
}
```

---

## 12.3 上报设备心跳

```http
POST /api/v1/app/heartbeat
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-abc-123",
  "userId": 10001,
  "versionName": "0.3.0-beta.2",
  "versionCode": 301,
  "buildNumber": 52,
  "resourceVersion": "20260601.3",
  "osVersion": "14",
  "deviceModel": "Pixel 8"
}
```

---

## 12.4 上报更新事件

```http
POST /api/v1/app/update-event
```

请求：

```json
{
  "appKey": "game-helper-android",
  "deviceId": "device-abc-123",
  "eventType": "activation_success",
  "fromVersion": "20260601.1",
  "toVersion": "20260601.3",
  "packageKey": "templates-bear"
}
```

---

## 12.5 拉取任务前校验

```http
POST /api/v1/jobs/poll
```

客户端必须携带：

```json
{
  "deviceId": "device-abc-123",
  "appVersionCode": 301,
  "resourceVersion": "20260601.3"
}
```

当客户端太旧时返回：

```json
{
  "code": "APP_VERSION_UNSUPPORTED",
  "message": "当前版本已停止支持，请升级后继续执行任务。",
  "forceUpdate": true
}
```

资源版本过旧时返回：

```json
{
  "code": "RESOURCE_VERSION_UNSUPPORTED",
  "message": "运行资源已更新，请重新启动 App 完成更新。",
  "forceUpdate": true
}
```

---

# 13. 后台管理端页面设计

一级菜单：

```text
App 管理
├── App 列表
├── 构建记录
├── App 版本发布
├── 资源增量发布
├── 设备版本
├── 升级统计
└── 操作审计
```

---

## 13.1 App 列表

展示：

| 应用 | 平台 | 包名 | 当前正式版 | 当前测试版 | 最新资源版 | 活跃设备 | 操作 |
|---|---|---|---|---|---|---:|---|
| 游戏助手 | Android | `cc.misk.gamehelper` | 未发布 | `0.3.0-beta.2` | `20260601.3` | 28 | 查看 |

---

## 13.2 构建记录

展示：

| 构建编号 | 版本 | 渠道 | Git Commit | 环境 | 状态 | 文件大小 | 构建时间 | 操作 |
|---:|---|---|---|---|---|---:|---|---|
| 52 | `0.3.0-beta.2` | beta | `a93cd21` | test | success | 46.2 MB | 2026-06-01 14:30 | 创建发布 |
| 51 | `0.3.0-beta.2` | beta | `a93cd21` | test | failed | — | 2026-06-01 14:12 | 查看日志 |

操作：

```text
查看详情
查看构建日志
下载 APK
复制下载链接
创建发布
归档
```

---

## 13.3 创建 App 发布

表单字段：

```text
应用：游戏助手 Android
选择构建：0.3.0-beta.2 build 52
渠道：dev / internal / beta / stable / emergency
标题：测试版本更新
摘要：优化任务执行稳定性
更新内容：Markdown 编辑器
升级级别：普通 / 推荐 / 强制
是否阻止旧版本执行任务：是 / 否
灰度比例：5% / 10% / 20% / 50% / 100%
目标用户：全部 / 用户组 / 指定用户 / 指定设备
发布时间：立即 / 定时
```

更新内容支持后台编辑。

示例：

```markdown
## 本次更新

- 新增悬浮窗快捷控制。
- 优化随缘打熊场景识别。
- 修复部分设备 OCR 识别失败的问题。
- 提升任务执行稳定性。

## 已知问题

- 部分低版本 Android 设备首次安装时需要手工允许未知来源安装。
```

---

## 13.4 资源增量发布

表单字段：

```text
资源版本：20260601.3
渠道：dev / internal / beta / stable
标题：运行资源更新
摘要：优化打熊识别和 OCR 关键词库
更新内容：Markdown 编辑器
最低 App 版本：0.3.0-beta.2
升级级别：普通 / 推荐 / 强制
灰度比例：5% / 10% / 20% / 50% / 100%
上传资源包：
  - templates-bear-20260601.3.zip
  - ocr-dictionary-20260601.3.zip
```

后端自动完成：

```text
计算文件大小
  ↓
计算 SHA-256
  ↓
保存文件
  ↓
生成 Manifest
  ↓
保存资源版本记录
  ↓
等待发布
```

---

## 13.5 版本详情页

页签：

```text
概览
更新内容
设备列表
升级事件
灰度记录
操作审计
```

概览卡片：

```text
已升级设备
待升级设备
升级成功率
下载失败
安装失败
资源激活失败
当前灰度比例
```

---

## 13.6 设备版本页

展示：

| 设备 | 用户 | App 版本 | 构建编号 | 资源版本 | 系统版本 | 最近在线 | 状态 |
|---|---|---|---:|---|---|---|---|
| Redmi K70 | 用户 10001 | `0.3.0-beta.2` | 52 | `20260601.3` | Android 15 | 1 分钟前 | 正常 |
| 模拟器 03 | 用户 10012 | `0.3.0-beta.1` | 48 | `20260601.1` | Android 12 | 3 分钟前 | 待更新 |
| Pixel 8 | 用户 10032 | `0.3.0-beta.2` | 52 | `20260601.2` | Android 14 | 8 分钟前 | 资源激活失败 |

---

# 14. App 客户端页面规范

---

## 14.1 启动页无更新

```text
┌──────────────────────────┐
│                          │
│          游戏助手       │
│                          │
│       正在初始化服务…      │
│                          │
└──────────────────────────┘
```

---

## 14.2 启动页资源增量更新

```text
┌──────────────────────────┐
│                          │
│         游戏助手       │
│                          │
│      正在更新运行资源      │
│      ████████░░  78%      │
│                          │
│      1.8 MB / 2.3 MB      │
│                          │
└──────────────────────────┘
```

---

## 14.3 普通 APK 更新

```text
发现新版本 0.3.0-beta.2

本次更新
• 新增悬浮窗快捷控制
• 优化任务执行稳定性
• 修复 OCR 识别异常

安装包大小：46.2 MB

[稍后提醒]   [立即更新]
```

---

## 14.4 强制 APK 更新

```text
需要升级后继续使用

当前版本已停止支持。
请升级到最新版本后继续运行任务。

[立即升级]
```

---

## 14.5 普通资源更新失败

```text
资源更新失败

暂时无法获取最新资源，将继续使用本地版本。
部分新功能可能暂不可用。

[重新尝试]   [进入应用]
```

---

## 14.6 强制资源更新失败

```text
需要完成资源更新

当前运行资源版本已停止支持。
请检查网络连接后重新尝试。

[重新尝试]
```

---

# 15. 开发测试阶段发版规范

你目前还在开发测试阶段，建议不要一开始就做复杂正式发版。

当前阶段使用以下流程：

```text
开发人员提交代码
  ↓
生成 dev 构建
  ↓
开发人员自己安装测试
  ↓
功能基本可用
  ↓
生成 internal 构建
  ↓
内部设备测试
  ↓
关键流程跑通
  ↓
生成 beta 构建
  ↓
少量真实用户测试
  ↓
稳定后准备 stable 1.0.0
```

---

## 15.1 当前阶段建议渠道

| 阶段 | 渠道 | 示例版本 |
|---|---|---|
| 日常开发 | `dev` | `0.3.0-dev.7` |
| 内部验证 | `internal` | `0.3.0-internal.2` |
| 小范围测试 | `beta` | `0.3.0-beta.1` |
| 正式上线 | `stable` | `1.0.0` |

---

## 15.2 当前阶段发布要求

`dev`：

- 可以频繁打包。
- 不需要强制填写完整更新说明。
- 只发给开发人员。
- 可以允许手工上传 APK。
- 构建失败需要保留日志。

`internal`：

- 必须填写更新摘要。
- 必须标注主要改动。
- 必须完成基本启动、登录、悬浮窗、任务拉取测试。
- 建议只发内部测试设备。

`beta`：

- 必须填写完整更新内容。
- 必须支持后台修改更新说明。
- 必须记录灰度范围。
- 必须记录已知问题。
- 必须可暂停发布。
- 必须能查看设备升级情况。

`stable`：

- 必须经过 beta 验证。
- 必须先灰度发布。
- 必须保留上一个稳定版本 APK。
- 必须保留上一个稳定资源版本。
- 必须支持暂停、撤回和紧急修复。

---

# 16. 正式发布流程

正式上线后建议使用：

```text
开发完成
  ↓
dev 自测
  ↓
internal 内部测试
  ↓
beta 指定用户测试
  ↓
stable 5% 灰度
  ↓
观察升级成功率、任务失败率、资源激活失败率
  ↓
stable 20% 灰度
  ↓
stable 50% 灰度
  ↓
stable 100% 全量发布
```

出现异常：

```text
暂停灰度
  ↓
阻止扩大覆盖
  ↓
分析失败设备
  ↓
回滚资源版本或发布 hotfix
  ↓
必要时阻止问题版本拉取任务
```

---

# 17. 增量资源发布流程

```text
修改场景配置 / 图片模板 / OCR 关键词
  ↓
按模块打包 ZIP
  ↓
后台上传资源包
  ↓
自动生成 resourceVersion
  ↓
自动计算 SHA-256
  ↓
自动生成 Manifest
  ↓
填写更新内容
  ↓
发布到 dev
  ↓
内部验证
  ↓
发布到 beta
  ↓
观察激活成功率
  ↓
发布到 stable
```

资源更新优先级：

```text
normal       普通更新
recommended  推荐更新
forced       强制更新
```

---

# 18. 回滚规范

## 18.1 APK 回滚

Android APK 不能简单覆盖安装更低 `versionCode` 的旧版本，因此建议：

- 保留旧 APK 用于排查。
- 出现问题时快速发布新 `versionCode` 的 hotfix。
- 如果必须恢复旧逻辑，基于旧代码重新构建一个更高 `versionCode` 的新 APK。

示例：

```text
问题版本：1.0.1 versionCode 10001
恢复版本：1.0.2 versionCode 10002
```

虽然代码回退，但版本号仍然前进。

---

## 18.2 资源版本回滚

资源版本支持快速回滚。

```text
当前资源：20260601.3
  ↓
发现识图模板异常
  ↓
后台将稳定资源指针切回 20260601.2
  ↓
客户端下次启动检查到回滚版本
  ↓
激活旧资源
```

客户端至少保留最近两个资源版本。

---

# 19. 安全规范

## 19.1 文件校验

所有 APK 和 ZIP 必须保存：

```text
文件大小
SHA-256
上传时间
上传人
所属版本
```

客户端下载后必须校验 SHA-256。

---

## 19.2 签名密钥

禁止将以下内容放入 Git 仓库或数据库明文：

```text
Android keystore
keystore 密码
私钥
Google Play Service Account JSON
App Store API Key
```

建议路径：

```text
/srv/secrets/app-signing/
└── android/
    └── game-helper-release.keystore
```

权限：

```bash
chmod 700 /srv/secrets/app-signing
chmod 600 /srv/secrets/app-signing/android/game-helper-release.keystore
```

---

## 19.3 Manifest 签名

建议使用 Ed25519 对 Manifest 签名：

```text
服务端私钥签名
  ↓
客户端内置公钥验签
  ↓
验签通过后才允许下载和加载资源
```

---

## 19.4 权限控制

后台权限建议拆分：

```text
app.build.upload
app.release.create
app.release.publish
app.release.pause
app.release.recall
app.resource.upload
app.resource.publish
app.release.notes.edit
app.release.audit.view
```

正式发布、强制更新和撤回操作建议增加二次确认。

---

# 20. 操作审计规范

需要记录：

```text
谁上传了构建
谁创建了发布
谁修改了更新说明
谁调整了灰度比例
谁暂停了发布
谁撤回了版本
谁发布了资源增量
谁执行了资源回滚
```

建议表：

```sql
CREATE TABLE app_release_audit_logs (
  id                  BIGSERIAL PRIMARY KEY,
  app_id              BIGINT NOT NULL REFERENCES apps(id),

  operator            VARCHAR(128) NOT NULL,
  action              VARCHAR(64) NOT NULL,
  target_type         VARCHAR(32) NOT NULL,
  target_id           BIGINT,

  before_json         JSONB,
  after_json          JSONB,

  created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

# 21. 一期开发范围

当前阶段优先完成以下内容。

## 21.1 后端

```text
1. apps 表
2. app_builds 表
3. app_releases 表
4. app_release_notes 表
5. app_installations 表
6. app_resource_versions 表
7. app_resource_packages 表
8. app_upgrade_events 表
9. app_resource_update_events 表
10. APK 上传接口
11. 资源 ZIP 上传接口
12. 自动计算 SHA-256
13. 自动生成 Manifest
14. update-check 接口
15. resource-check 接口
16. heartbeat 接口
17. update-event 接口
18. jobs/poll 版本校验
```

---

## 21.2 后台前端

```text
1. App 列表
2. 构建记录
3. 创建 App 发布
4. Markdown 更新内容编辑器
5. 创建资源增量发布
6. 上传资源 ZIP
7. 发布、暂停、撤回
8. 灰度比例调整
9. 设备版本列表
10. 升级事件列表
11. 操作审计列表
```

---

## 21.3 Android 客户端

```text
1. 启动页
2. update-check
3. 普通 APK 更新弹窗
4. 强制 APK 更新弹窗
5. APK 下载
6. SHA-256 校验
7. 调起系统安装器
8. resource-check
9. 增量 ZIP 下载
10. staging 临时目录
11. ZIP 校验
12. 解压和文件清单校验
13. 原子切换 current.json
14. 保留最近两个资源版本
15. 自动回滚
16. 更新事件上报
17. 心跳上报
18. jobs/poll 版本参数
```

---

# 22. 二期增强范围

一期跑通后再做：

```text
1. CI 自动构建 APK
2. Git Tag 自动生成版本
3. 自动上传构建产物
4. APK 差分补丁
5. 多语言更新说明
6. 定时发布
7. 更新成功率图表
8. 任务失败率联动告警
9. 资源激活失败自动暂停发布
10. Google Play 发布状态同步
11. App Store / TestFlight 状态同步
12. Telegram 审批发布
```

---

# 23. 推荐实施顺序

建议按照以下顺序实施：

```text
第一步：构建记录 + APK 上传
  ↓
第二步：App 发布 + 后台更新说明编辑
  ↓
第三步：App 启动检查整包更新
  ↓
第四步：设备心跳 + 版本分布
  ↓
第五步：资源版本 + ZIP 上传 + Manifest
  ↓
第六步：启动页增量更新
  ↓
第七步：任务拉取前版本校验
  ↓
第八步：灰度发布 + 暂停 + 回滚
  ↓
第九步：升级统计 + 操作审计
```

---

# 24. 最终推荐方案

当前阶段采用：

```text
APK 整包更新
+
启动页资源增量更新
+
后台 Markdown 更新说明编辑
+
dev / internal / beta / stable 渠道管理
+
灰度发布
+
设备版本上报
+
任务执行前版本校验
+
资源版本自动回滚
```

不要在一期加入过多复杂能力。

第一版只需要确保：

```text
能上传 APK
能创建版本
能编辑更新内容
能发布测试版本
App 启动能检查更新
资源包能增量下载
下载失败不会破坏旧资源
任务执行前能拦截旧版本
后台能看到设备版本
出现问题能暂停和回滚
```

这套方案可以覆盖当前开发测试阶段，也可以平滑演进到正式发布阶段。
