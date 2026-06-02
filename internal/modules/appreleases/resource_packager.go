package appreleases

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type IncrementalResourcePackageDefinition struct {
	PackageKey        string   `json:"package_key"`
	Description       string   `json:"description"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

type ResourcePackageSource struct {
	PackageKey string
	Root       string
}

type BuildResourceBundleRequest struct {
	AppKey               string
	ResourceVersion      string
	Channel              string
	UpdateLevel          string
	Title                string
	Summary              string
	ReleaseNotesMarkdown string
	MinAppVersionCode    int
	MaxAppVersionCode    int
	ManifestPrivateKey   string
	OutputDir            string
	Sources              []ResourcePackageSource
}

type ResourceManifestRequest struct {
	AppKey               string
	ResourceVersion      string
	Channel              string
	UpdateLevel          string
	MinAppVersionCode    int
	MaxAppVersionCode    int
	Title                string
	Summary              string
	ReleaseNotesMarkdown string
	ManifestPrivateKey   string
}

type BuiltResourcePackage struct {
	PackageKey  string   `json:"package_key"`
	PackageType string   `json:"package_type"`
	FileName    string   `json:"file_name"`
	Path        string   `json:"path"`
	FileSize    int64    `json:"file_size"`
	SHA256      string   `json:"sha256"`
	FileCount   int      `json:"file_count"`
	Files       []string `json:"files"`
}

type BuiltResourceBundle struct {
	MetadataPath string                       `json:"metadata_path"`
	ManifestPath string                       `json:"manifest_path"`
	BundlePath   string                       `json:"bundle_path"`
	Metadata     CreateResourceVersionRequest `json:"metadata"`
	Packages     []BuiltResourcePackage       `json:"packages"`
}

var incrementalResourcePackageCatalog = []IncrementalResourcePackageDefinition{
	{PackageKey: "scenes-core", Description: "基础场景配置", AllowedExtensions: []string{".json", ".yaml", ".yml", ".toml", ".txt"}},
	{PackageKey: "scenes-events", Description: "活动脚本配置数据", AllowedExtensions: []string{".json", ".yaml", ".yml", ".toml", ".txt"}},
	{PackageKey: "templates-common", Description: "通用识图模板", AllowedExtensions: []string{".png", ".jpg", ".jpeg", ".webp", ".bmp", ".json", ".yaml", ".yml", ".txt"}},
	{PackageKey: "templates-bear", Description: "打熊识图模板", AllowedExtensions: []string{".png", ".jpg", ".jpeg", ".webp", ".bmp", ".json", ".yaml", ".yml", ".txt"}},
	{PackageKey: "ocr-dictionary", Description: "OCR 关键词库", AllowedExtensions: []string{".json", ".yaml", ".yml", ".txt", ".csv", ".dict"}},
	{PackageKey: "ui-assets", Description: "启动页和 UI 静态资源", AllowedExtensions: []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".css", ".html", ".json", ".txt", ".ttf", ".otf", ".woff", ".woff2"}},
	{PackageKey: "feature-flags", Description: "功能开关", AllowedExtensions: []string{".json", ".yaml", ".yml", ".toml", ".txt"}},
}

var forbiddenIncrementalResourceExtensions = map[string]string{
	".aab":   "Android App Bundle 属于整包发布",
	".apk":   "APK 属于整包发布",
	".bash":  "脚本不能通过资源增量发布",
	".bat":   "脚本不能通过资源增量发布",
	".cjs":   "可执行脚本不能通过资源增量发布",
	".class": "JVM 字节码不能通过资源增量发布",
	".cmd":   "脚本不能通过资源增量发布",
	".dex":   "Dex 字节码不能通过资源增量发布",
	".dll":   "原生动态库不能通过资源增量发布",
	".dylib": "原生动态库不能通过资源增量发布",
	".exe":   "可执行文件不能通过资源增量发布",
	".fish":  "脚本不能通过资源增量发布",
	".jar":   "JAR 不能通过资源增量发布",
	".js":    "未审计脚本不能通过资源增量发布",
	".mjs":   "未审计脚本不能通过资源增量发布",
	".ps1":   "脚本不能通过资源增量发布",
	".sh":    "脚本不能通过资源增量发布",
	".so":    "原生动态库不能通过资源增量发布",
	".wasm":  "可执行模块不能通过资源增量发布",
	".zsh":   "脚本不能通过资源增量发布",
}

func IncrementalResourcePackageCatalog() []IncrementalResourcePackageDefinition {
	out := make([]IncrementalResourcePackageDefinition, len(incrementalResourcePackageCatalog))
	copy(out, incrementalResourcePackageCatalog)
	return out
}

func ValidateIncrementalResourcePackageKey(packageKey string) error {
	if _, ok := incrementalResourcePolicy(packageKey); !ok {
		return fmt.Errorf("unsupported incremental resource package_key %q", packageKey)
	}
	return nil
}

func ValidateIncrementalResourceFile(packageKey, relPath string) error {
	policy, ok := incrementalResourcePolicy(packageKey)
	if !ok {
		return fmt.Errorf("unsupported incremental resource package_key %q", packageKey)
	}
	base := filepath.Base(relPath)
	if base == "" || strings.HasPrefix(base, ".") {
		return fmt.Errorf("%s is hidden or empty", relPath)
	}
	ext := strings.ToLower(filepath.Ext(base))
	if reason, forbidden := forbiddenIncrementalResourceExtensions[ext]; forbidden {
		return fmt.Errorf("%s: %s", relPath, reason)
	}
	for _, allowed := range policy.AllowedExtensions {
		if ext == allowed {
			return nil
		}
	}
	return fmt.Errorf("%s: extension %q is not allowed for %s", relPath, ext, packageKey)
}

func ValidateIncrementalResourceArchive(packageKey string, content []byte) error {
	if err := ValidateIncrementalResourcePackageKey(packageKey); err != nil {
		return err
	}
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return fmt.Errorf("%s: invalid zip package: %w", packageKey, err)
	}
	fileCount := 0
	for _, file := range reader.File {
		name := strings.TrimSpace(filepath.ToSlash(file.Name))
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "../") {
			return fmt.Errorf("%s: invalid zip entry %q", packageKey, file.Name)
		}
		if strings.HasSuffix(name, "/") {
			continue
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlink entry %q is not allowed", packageKey, file.Name)
		}
		if err := ValidateIncrementalResourceFile(packageKey, name); err != nil {
			return err
		}
		fileCount++
	}
	if fileCount == 0 {
		return fmt.Errorf("%s: zip package has no resource files", packageKey)
	}
	return nil
}

func BuildResourceBundle(req BuildResourceBundleRequest) (BuiltResourceBundle, error) {
	req.ResourceVersion = strings.TrimSpace(req.ResourceVersion)
	req.Channel = normalizeChannel(req.Channel, "dev")
	req.UpdateLevel = normalizeUpdateLevel(req.UpdateLevel)
	req.Title = strings.TrimSpace(req.Title)
	req.OutputDir = strings.TrimSpace(req.OutputDir)
	if req.ResourceVersion == "" {
		return BuiltResourceBundle{}, fmt.Errorf("resource_version is required")
	}
	if req.Title == "" {
		return BuiltResourceBundle{}, fmt.Errorf("title is required")
	}
	if req.OutputDir == "" {
		return BuiltResourceBundle{}, fmt.Errorf("output_dir is required")
	}
	if len(req.Sources) == 0 {
		return BuiltResourceBundle{}, fmt.Errorf("at least one resource package source is required")
	}
	if err := os.MkdirAll(req.OutputDir, 0o755); err != nil {
		return BuiltResourceBundle{}, err
	}

	metadata := CreateResourceVersionRequest{
		ResourceVersion:      req.ResourceVersion,
		Channel:              req.Channel,
		Title:                req.Title,
		Summary:              strings.TrimSpace(req.Summary),
		ReleaseNotesMarkdown: strings.TrimSpace(req.ReleaseNotesMarkdown),
		MinAppVersionCode:    req.MinAppVersionCode,
		MaxAppVersionCode:    req.MaxAppVersionCode,
		UpdateLevel:          req.UpdateLevel,
		RolloutPercentage:    100,
	}

	bundle := BuiltResourceBundle{Metadata: metadata}
	for _, source := range req.Sources {
		pkg, err := buildOneResourcePackage(req.OutputDir, req.ResourceVersion, source)
		if err != nil {
			return BuiltResourceBundle{}, err
		}
		bundle.Packages = append(bundle.Packages, pkg)
	}
	sort.Slice(bundle.Packages, func(i, j int) bool {
		return bundle.Packages[i].PackageKey < bundle.Packages[j].PackageKey
	})

	manifestPackages := make([]AppResourcePackageAdmin, 0, len(bundle.Packages))
	for _, pkg := range bundle.Packages {
		manifestPackages = append(manifestPackages, AppResourcePackageAdmin{
			PackageKey:  pkg.PackageKey,
			PackageType: pkg.PackageType,
			FileURL:     pkg.FileName,
			FileSize:    pkg.FileSize,
			SHA256:      pkg.SHA256,
		})
	}
	manifestBytes, err := BuildResourceManifest(ResourceManifestRequest{
		AppKey:               req.AppKey,
		ResourceVersion:      req.ResourceVersion,
		Channel:              req.Channel,
		UpdateLevel:          req.UpdateLevel,
		MinAppVersionCode:    req.MinAppVersionCode,
		MaxAppVersionCode:    req.MaxAppVersionCode,
		Title:                req.Title,
		Summary:              req.Summary,
		ReleaseNotesMarkdown: req.ReleaseNotesMarkdown,
		ManifestPrivateKey:   req.ManifestPrivateKey,
	}, manifestPackages)
	if err != nil {
		return BuiltResourceBundle{}, err
	}

	bundle.MetadataPath = filepath.Join(req.OutputDir, "metadata-"+safeFilePart(req.ResourceVersion)+".json")
	bundle.ManifestPath = filepath.Join(req.OutputDir, "manifest-"+safeFilePart(req.ResourceVersion)+".json")
	bundle.BundlePath = filepath.Join(req.OutputDir, "bundle-"+safeFilePart(req.ResourceVersion)+".json")
	if err := writeJSON(bundle.MetadataPath, metadata); err != nil {
		return BuiltResourceBundle{}, err
	}
	if err := os.WriteFile(bundle.ManifestPath, append(manifestBytes, '\n'), 0o644); err != nil {
		return BuiltResourceBundle{}, err
	}
	if err := writeJSON(bundle.BundlePath, bundle); err != nil {
		return BuiltResourceBundle{}, err
	}
	return bundle, nil
}

func BuildResourceManifest(req ResourceManifestRequest, packages []AppResourcePackageAdmin) ([]byte, error) {
	type manifestPackage struct {
		PackageKey  string `json:"packageKey"`
		PackageType string `json:"packageType"`
		URL         string `json:"url"`
		Size        int64  `json:"size"`
		SHA256      string `json:"sha256"`
	}
	manifest := struct {
		AppKey               string            `json:"appKey"`
		ResourceVersion      string            `json:"resourceVersion"`
		ResourceVersionSnake string            `json:"resource_version"`
		Channel              string            `json:"channel"`
		UpdateLevel          string            `json:"updateLevel"`
		MinAppVersionCode    int               `json:"minAppVersionCode,omitempty"`
		MaxAppVersionCode    int               `json:"maxAppVersionCode,omitempty"`
		Title                string            `json:"title"`
		Summary              string            `json:"summary,omitempty"`
		ReleaseNotesMarkdown string            `json:"releaseNotesMarkdown,omitempty"`
		Packages             []manifestPackage `json:"packages"`
		GeneratedAt          time.Time         `json:"generatedAt"`
	}{
		AppKey:               firstNonBlank(req.AppKey, "game-helper-android"),
		ResourceVersion:      req.ResourceVersion,
		ResourceVersionSnake: req.ResourceVersion,
		Channel:              req.Channel,
		UpdateLevel:          req.UpdateLevel,
		MinAppVersionCode:    req.MinAppVersionCode,
		MaxAppVersionCode:    req.MaxAppVersionCode,
		Title:                req.Title,
		Summary:              req.Summary,
		ReleaseNotesMarkdown: req.ReleaseNotesMarkdown,
		GeneratedAt:          time.Now().UTC(),
	}
	for _, pkg := range packages {
		manifest.Packages = append(manifest.Packages, manifestPackage{
			PackageKey:  pkg.PackageKey,
			PackageType: firstNonBlank(pkg.PackageType, "zip"),
			URL:         pkg.FileURL,
			Size:        pkg.FileSize,
			SHA256:      pkg.SHA256,
		})
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return SignResourceManifest(manifestBytes, req.ManifestPrivateKey)
}

func buildOneResourcePackage(outputDir, resourceVersion string, source ResourcePackageSource) (BuiltResourcePackage, error) {
	packageKey := safeFilePart(firstNonBlank(source.PackageKey, filepath.Base(source.Root)))
	if err := ValidateIncrementalResourcePackageKey(packageKey); err != nil {
		return BuiltResourcePackage{}, err
	}
	root, err := filepath.Abs(source.Root)
	if err != nil {
		return BuiltResourcePackage{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return BuiltResourcePackage{}, err
	}
	if !info.IsDir() {
		return BuiltResourcePackage{}, fmt.Errorf("%s is not a directory", source.Root)
	}

	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(name, ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symlinks are not allowed in incremental resource packages", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if err := ValidateIncrementalResourceFile(packageKey, rel); err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return BuiltResourcePackage{}, err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return BuiltResourcePackage{}, fmt.Errorf("%s has no eligible incremental resource files", source.Root)
	}

	fileName := packageKey + "-" + safeFilePart(resourceVersion) + ".zip"
	zipPath := filepath.Join(outputDir, fileName)
	if err := writeResourceZip(root, zipPath, files); err != nil {
		return BuiltResourcePackage{}, err
	}
	sum, size, err := fileSHA256(zipPath)
	if err != nil {
		return BuiltResourcePackage{}, err
	}
	return BuiltResourcePackage{
		PackageKey:  packageKey,
		PackageType: "zip",
		FileName:    fileName,
		Path:        zipPath,
		FileSize:    size,
		SHA256:      sum,
		FileCount:   len(files),
		Files:       files,
	}, nil
}

func writeResourceZip(root, zipPath string, files []string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, rel := range files {
		fullPath := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(fullPath)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		header.Modified = time.Unix(0, 0).UTC()
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(fullPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func fileSHA256(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func incrementalResourcePolicy(packageKey string) (IncrementalResourcePackageDefinition, bool) {
	packageKey = safeFilePart(packageKey)
	for _, policy := range incrementalResourcePackageCatalog {
		if policy.PackageKey == packageKey {
			return policy, true
		}
	}
	return IncrementalResourcePackageDefinition{}, false
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}
