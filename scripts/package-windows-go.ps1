param(
  [string]$ProjectName = "",
  [string]$MainPackage = ".",
  [string]$OutputDir = "dist\windows",
  [string]$BinaryName = "",
  [string]$VersionName = "",
  [string]$VersionCode = "",
  [string]$GitRef = "",
  [string]$GitCommit = "",
  [string]$Channel = "",
  [string]$Ldflags = "",
  [switch]$SkipZip
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-EnvValue([string]$Name) {
  $value = [Environment]::GetEnvironmentVariable($Name)
  if ($null -eq $value) {
    return ""
  }
  return $value
}

function First-NonBlank([string[]]$Values) {
  foreach ($value in $Values) {
    if ($null -ne $value -and $value.Trim().Length -gt 0) {
      return $value.Trim()
    }
  }
  return ""
}

$ProjectName = First-NonBlank @($ProjectName, (Get-EnvValue "BUILD_CENTER_PROJECT"))
$VersionName = First-NonBlank @($VersionName, (Get-EnvValue "VERSION_NAME"))
$VersionCode = First-NonBlank @($VersionCode, (Get-EnvValue "VERSION_CODE"))
$GitRef = First-NonBlank @($GitRef, (Get-EnvValue "GIT_REF"))
$GitCommit = First-NonBlank @($GitCommit, (Get-EnvValue "GIT_COMMIT"))
$Channel = First-NonBlank @($Channel, (Get-EnvValue "CHANNEL"))

function Resolve-GitCommit {
  if ($GitCommit.Length -gt 0) {
    return $GitCommit
  }
  $git = Get-Command git -ErrorAction SilentlyContinue
  if (-not $git) {
    return ""
  }
  try {
    return (git rev-parse HEAD).Trim()
  } catch {
    return ""
  }
}

function Resolve-ProjectName {
  if ($ProjectName.Length -gt 0) {
    return $ProjectName
  }
  $leaf = Split-Path -Leaf (Get-Location)
  if ($leaf.Length -gt 0) {
    return $leaf
  }
  return "windows-go-app"
}

function Normalize-FileName([string]$Value) {
  $name = $Value.Trim()
  if ($name.Length -eq 0) {
    $name = "windows-go-app"
  }
  $invalid = [IO.Path]::GetInvalidFileNameChars()
  foreach ($char in $invalid) {
    $name = $name.Replace([string]$char, "-")
  }
  return $name
}

function Get-Sha256([string]$Path) {
  return (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant()
}

$project = Resolve-ProjectName
if ($BinaryName.Length -eq 0) {
  $BinaryName = Normalize-FileName $project
} else {
  $BinaryName = Normalize-FileName $BinaryName
}
if (-not $BinaryName.EndsWith(".exe", [StringComparison]::OrdinalIgnoreCase)) {
  $BinaryName = "$BinaryName.exe"
}

$commit = Resolve-GitCommit
$version = if ($VersionName.Length -gt 0) { $VersionName } else { "0.0.0" }
$build = if ($VersionCode.Length -gt 0) { $VersionCode } else { "0" }
$artifactDir = Join-Path (Get-Location) $OutputDir
New-Item -ItemType Directory -Path $artifactDir -Force | Out-Null

$exePath = Join-Path $artifactDir $BinaryName
$zipName = "{0}-{1}-windows-amd64.zip" -f (Normalize-FileName $project), (Normalize-FileName $version)
$zipPath = Join-Path $artifactDir $zipName
$manifestPath = Join-Path $artifactDir "windows-artifact-manifest.json"

$effectiveLdflags = $Ldflags.Trim()
if ($effectiveLdflags.Length -eq 0) {
  $parts = @()
  if ($version.Length -gt 0) { $parts += "-X main.version=$version" }
  if ($commit.Length -gt 0) { $parts += "-X main.commit=$commit" }
  if ($build.Length -gt 0) { $parts += "-X main.build=$build" }
  if ($parts.Count -gt 0) { $effectiveLdflags = [string]::Join(" ", $parts) }
}

[Environment]::SetEnvironmentVariable("GOOS", "windows", "Process")
if ((Get-EnvValue "GOARCH").Length -eq 0) {
  [Environment]::SetEnvironmentVariable("GOARCH", "amd64", "Process")
}

Write-Host "windows go package start"
Write-Host "project=$project version=$version build=$build ref=$GitRef commit=$commit channel=$Channel"
Write-Host "go env GOOS=$(Get-EnvValue 'GOOS') GOARCH=$(Get-EnvValue 'GOARCH')"
Write-Host "output=$exePath"

$arguments = @("build", "-trimpath", "-o", $exePath)
if ($effectiveLdflags.Length -gt 0) {
  $arguments += @("-ldflags", $effectiveLdflags)
}
$arguments += $MainPackage
& go @arguments
if ($LASTEXITCODE -ne 0) {
  throw "go build failed with exit code $LASTEXITCODE"
}

$artifacts = New-Object System.Collections.Generic.List[object]
$exeInfo = Get-Item $exePath
$artifacts.Add([ordered]@{
  name = "windows-exe"
  type = "windows_exe"
  file_name = $exeInfo.Name
  path = $exeInfo.FullName
  size_bytes = $exeInfo.Length
  sha256 = Get-Sha256 $exeInfo.FullName
})

if (-not $SkipZip) {
  if (Test-Path $zipPath) {
    Remove-Item -Force $zipPath
  }
  Compress-Archive -Path $exePath -DestinationPath $zipPath -Force
  $zipInfo = Get-Item $zipPath
  $artifacts.Add([ordered]@{
    name = "windows-archive"
    type = "windows_archive"
    file_name = $zipInfo.Name
    path = $zipInfo.FullName
    size_bytes = $zipInfo.Length
    sha256 = Get-Sha256 $zipInfo.FullName
  })
}

$manifest = [ordered]@{
  build_id = (Get-EnvValue "BUILD_RUN_ID")
  project = $project
  git_ref = $GitRef
  git_commit = $commit
  version_name = $version
  version_code = [int]$build
  build_number = [int]$build
  status = "success"
  artifact_dir = $artifactDir
  log_dir = ""
  artifacts = $artifacts
}
$manifest | ConvertTo-Json -Depth 8 | Set-Content -Path $manifestPath -Encoding UTF8

Write-Host "windows go package complete"
Write-Host "manifest=$manifestPath"
foreach ($artifact in $artifacts) {
  Write-Host ("artifact {0} type={1} path={2}" -f $artifact.name, $artifact.type, $artifact.path)
}
