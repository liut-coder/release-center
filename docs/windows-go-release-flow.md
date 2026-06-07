# Windows Go 打包发布闭环

## 目标

Windows Go 项目通过发布中心完成以下链路：

1. 集成配置创建 `windows` build profile。
2. 触发构建后，服务端创建 `build_center_runs` 并投递 `required_labels=["windows"]` 的 Worker 任务。
3. Windows Worker 执行项目仓库中的 PowerShell 打包命令。
4. Worker 按 build profile 的 `artifact_rules` 收集 exe/zip，随任务完成回传。
5. 服务端把 Worker 制品镜像到构建中心制品表，制品中心和发布计划可以继续使用。

## 默认模板

集成配置页的 `GitHub Windows` 模板会生成：

```json
{
  "stack_type": "windows",
  "build_type": "windows",
  "commands": {
    "all": "powershell -ExecutionPolicy Bypass -File scripts\\package-windows-go.ps1"
  },
  "artifact_rules": [
    { "name": "windows-exe", "type": "windows_exe", "path": "dist/windows/*.exe" },
    { "name": "windows-archive", "type": "windows_archive", "path": "dist/windows/*.zip" }
  ]
}
```

项目仓库需要提供 `scripts/package-windows-go.ps1`。发布中心仓库内的 [package-windows-go.ps1](/root/build-center/projects/release-center/scripts/package-windows-go.ps1) 可作为通用脚本复制或改造。

## Worker 环境

Windows Worker 需要注册包含 `windows` 标签，并开启执行：

```powershell
releasectl worker-run `
  -base-url http://127.0.0.1:18080 `
  -token $env:GAME_HELPER_WORKER_TOKEN `
  -worker-key "$env:COMPUTERNAME-windows-worker" `
  -labels windows,amd64,go,powershell `
  -workdir C:\build-worker\workspace\desktop-tool `
  -execute
```

Worker 执行命令时会注入：

```text
BUILD_CENTER_PROJECT
BUILD_RUN_ID
GIT_REF
VERSION_NAME
VERSION_CODE
CHANNEL
```

脚本输出到 `dist/windows` 后，Worker 会按 `artifact_rules` 自动计算 `size_bytes` 和 `sha256` 并回传。

## 发布计划

发布计划支持 `windows` unit type。Windows 构建成功后，可从制品中心选择 `windows_exe` 或 `windows_archive` 创建发布计划，再进入审批和部署流程。

## 验证

本地服务端闭环验证：

```bash
/root/build-center/cache/tools/go1.25.0/bin/go test ./...
npm --prefix web run build
```

脚本烟测验证：

```bash
PATH=/root/build-center/cache/tools/go1.25.0/bin:$PATH \
BUILD_CENTER_PROJECT=release-center \
BUILD_RUN_ID=local-pwsh-smoke \
VERSION_NAME=0.1.0 \
VERSION_CODE=1 \
CHANNEL=local \
/root/build-center/cache/tools/powershell-7.5.4/pwsh \
  -NoLogo -NoProfile -ExecutionPolicy Bypass \
  -File scripts/package-windows-go.ps1 \
  -MainPackage ./cmd/releasectl \
  -BinaryName releasectl
```

该烟测在 Linux arm64 的 PowerShell Core 中已生成 `dist/windows/releasectl.exe`、`dist/windows/release-center-0.1.0-windows-amd64.zip` 和 `windows-artifact-manifest.json`，其中 exe 被识别为 Windows x86-64 PE 可执行文件。

真实 Windows 环境仍以 GitHub Actions 的 `windows-go-package-smoke` 或带 `windows` 标签的实际 Worker 运行为准。CI 支持 `workflow_dispatch`，改动推送到远端后可手动触发。
