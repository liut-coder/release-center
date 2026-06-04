package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/liut-coder/game-helper-server/internal/modules/appreleases"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "artifact-upload":
		err = runArtifactUpload(os.Args[2:])
	case "resource-catalog":
		err = runResourceCatalog(os.Args[2:])
	case "resource-pack":
		err = runResourcePack(os.Args[2:])
	case "manifest-verify":
		err = runManifestVerify(os.Args[2:])
	case "manifest-public-key":
		err = runManifestPublicKey(os.Args[2:])
	case "resource-upload":
		err = runResourceUpload(os.Args[2:])
	case "worker-run":
		err = runWorkerRun(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  releasectl artifact-upload -base-url http://127.0.0.1:18080 -file app-release.apk -git-ref main -version-name 0.2.0 -version-code 200 [-build-number 37]
  releasectl artifact-upload -base-url http://127.0.0.1:18080 -artifact-url https://ci.example/app.apk -git-ref main -version-name 0.2.0 -version-code 200
  releasectl resource-catalog
  releasectl resource-pack -version 20260601.1 -title "运行资源更新" -out dist/resources -root resources
  releasectl manifest-public-key -private-key "$MANIFEST_PRIVATE_KEY"
  releasectl manifest-verify -manifest dist/resources/manifest-20260601.1.json -public-key "$MANIFEST_PUBLIC_KEY"
  releasectl resource-upload -base-url http://127.0.0.1:18080 -bundle dist/resources/bundle-20260601.1.json -username admin -password admin123 [-publish]
  releasectl worker-run -base-url http://127.0.0.1:18080 -token "$GAME_HELPER_CI_TOKEN" -worker-key cf-prod-1 -labels linux,node,cloudflare -execute`)
}

func runArtifactUpload(args []string) error {
	fs := flag.NewFlagSet("artifact-upload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", env("GAME_HELPER_RELEASE_BASE_URL", "http://127.0.0.1:18080"), "server base URL")
	token := fs.String("token", env("GAME_HELPER_ADMIN_TOKEN", ""), "admin or CI bearer token")
	username := fs.String("username", env("GAME_HELPER_ADMIN_USER", "admin"), "admin username")
	password := fs.String("password", env("GAME_HELPER_ADMIN_PASSWORD", ""), "admin password")
	filePath := fs.String("file", "", "artifact file to upload")
	appKey := fs.String("app-key", "game-helper-android", "app key")
	artifactName := fs.String("artifact-name", "", "artifact name within the build")
	gitRef := fs.String("git-ref", env("GITHUB_REF_NAME", env("GITEA_REF_NAME", "")), "git ref or branch")
	gitCommit := fs.String("git-commit", env("GITHUB_SHA", env("GITEA_SHA", "")), "git commit sha")
	gitBranch := fs.String("git-branch", "", "git branch")
	buildType := fs.String("build-type", "debug", "build type")
	channel := fs.String("channel", "dev", "release channel")
	versionName := fs.String("version-name", "", "app versionName")
	versionCode := fs.Int("version-code", 0, "app versionCode")
	buildNumber := fs.Int("build-number", 0, "CI build number")
	artifactType := fs.String("artifact-type", "apk", "apk, aab, zip, web_dist, binary, docker_image, or artifact")
	artifactURL := fs.String("artifact-url", "", "existing artifact URL to register")
	fileName := fs.String("file-name", "", "artifact file name when registering a URL")
	sizeBytes := fs.Int64("size-bytes", 0, "artifact size when registering a URL")
	sha256Value := fs.String("sha256", "", "artifact sha256 when registering a URL")
	provider := fs.String("provider", env("GITHUB_ACTIONS_PROVIDER", "ci"), "ci provider")
	workflow := fs.String("workflow", env("GITHUB_WORKFLOW", env("GITEA_WORKFLOW", "")), "workflow name")
	runID := fs.String("run-id", env("GITHUB_RUN_ID", env("GITEA_RUN_ID", "")), "workflow run id")
	buildURL := fs.String("build-url", "", "CI build URL")
	releaseNotes := fs.String("notes", "", "release notes")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*versionName) == "" || *versionCode <= 0 {
		return fmt.Errorf("-version-name and -version-code are required")
	}
	if strings.TrimSpace(*gitRef) == "" {
		return fmt.Errorf("-git-ref is required")
	}
	if strings.TrimSpace(*filePath) == "" && strings.TrimSpace(*artifactURL) == "" {
		return fmt.Errorf("either -file or -artifact-url is required")
	}
	if strings.TrimSpace(*filePath) != "" && strings.TrimSpace(*artifactURL) != "" {
		return fmt.Errorf("-file and -artifact-url cannot be used together")
	}

	authToken := strings.TrimSpace(*token)
	var err error
	if authToken == "" {
		authToken, err = login(*baseURL, *username, *password)
		if err != nil {
			return err
		}
	}
	req := appreleases.CreateArtifactRequest{
		AppKey:       *appKey,
		ArtifactName: *artifactName,
		GitRef:       *gitRef,
		GitCommit:    *gitCommit,
		GitBranch:    *gitBranch,
		BuildType:    *buildType,
		Channel:      *channel,
		VersionName:  *versionName,
		VersionCode:  *versionCode,
		BuildNumber:  *buildNumber,
		ArtifactType: *artifactType,
		ArtifactURL:  *artifactURL,
		FileName:     *fileName,
		SizeBytes:    *sizeBytes,
		SHA256:       *sha256Value,
		Provider:     *provider,
		Workflow:     *workflow,
		RunID:        *runID,
		BuildURL:     *buildURL,
		ReleaseNotes: *releaseNotes,
	}
	var resp map[string]any
	if strings.TrimSpace(*filePath) != "" {
		resp, err = uploadArtifactFile(*baseURL, authToken, req, *filePath)
	} else {
		resp, err = registerArtifactURL(*baseURL, authToken, req)
	}
	if err != nil {
		return err
	}
	printJSON(resp)
	return nil
}

func runResourceCatalog(args []string) error {
	fs := flag.NewFlagSet("resource-catalog", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}
	body, err := json.MarshalIndent(appreleases.IncrementalResourcePackageCatalog(), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func runResourcePack(args []string) error {
	var sources sourceFlags
	fs := flag.NewFlagSet("resource-pack", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	appKey := fs.String("app-key", "game-helper-android", "app key")
	version := fs.String("version", defaultResourceVersion(time.Now()), "resource version")
	channel := fs.String("channel", "dev", "release channel")
	updateLevel := fs.String("update-level", "normal", "normal, recommended, or forced")
	title := fs.String("title", "运行资源更新", "release title")
	summary := fs.String("summary", "", "release summary")
	notes := fs.String("notes", "", "release notes markdown")
	minAppCode := fs.Int("min-app-code", 0, "minimum app version code")
	maxAppCode := fs.Int("max-app-code", 0, "maximum app version code")
	manifestPrivateKey := fs.String("manifest-private-key", env("GAME_HELPER_MANIFEST_PRIVATE_KEY", ""), "Ed25519 private key seed/private key for manifest signing")
	out := fs.String("out", filepath.Join("dist", "app-resources"), "output directory")
	root := fs.String("root", "", "resource root containing package_key directories")
	fs.Var(&sources, "source", "package source in package_key=dir form; may repeat")
	if err := fs.Parse(args); err != nil {
		return err
	}

	pkgSources := []appreleases.ResourcePackageSource(sources)
	if strings.TrimSpace(*root) != "" {
		discovered, err := discoverPackageSources(*root)
		if err != nil {
			return err
		}
		pkgSources = append(pkgSources, discovered...)
	}
	if len(pkgSources) == 0 {
		return fmt.Errorf("no resource package sources; use -root or -source package_key=dir")
	}

	bundle, err := appreleases.BuildResourceBundle(appreleases.BuildResourceBundleRequest{
		AppKey:               *appKey,
		ResourceVersion:      *version,
		Channel:              *channel,
		UpdateLevel:          *updateLevel,
		Title:                *title,
		Summary:              *summary,
		ReleaseNotesMarkdown: *notes,
		MinAppVersionCode:    *minAppCode,
		MaxAppVersionCode:    *maxAppCode,
		ManifestPrivateKey:   *manifestPrivateKey,
		OutputDir:            *out,
		Sources:              pkgSources,
	})
	if err != nil {
		return err
	}
	printJSON(bundle)
	return nil
}

func runManifestVerify(args []string) error {
	fs := flag.NewFlagSet("manifest-verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	manifestPath := fs.String("manifest", "", "manifest json path")
	publicKey := fs.String("public-key", env("GAME_HELPER_MANIFEST_PUBLIC_KEY", ""), "Ed25519 public key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*manifestPath) == "" {
		return fmt.Errorf("-manifest is required")
	}
	body, err := os.ReadFile(*manifestPath)
	if err != nil {
		return err
	}
	if err := appreleases.VerifyResourceManifestSignature(body, *publicKey); err != nil {
		return err
	}
	printJSON(map[string]any{"ok": true, "manifest": *manifestPath})
	return nil
}

func runManifestPublicKey(args []string) error {
	fs := flag.NewFlagSet("manifest-public-key", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	privateKey := fs.String("private-key", env("GAME_HELPER_MANIFEST_PRIVATE_KEY", ""), "Ed25519 private key seed/private key")
	if err := fs.Parse(args); err != nil {
		return err
	}
	publicKey, err := appreleases.ManifestPublicKeyFromPrivateKey(*privateKey)
	if err != nil {
		return err
	}
	printJSON(map[string]any{"public_key": publicKey, "algorithm": appreleases.ManifestSignatureAlgorithmEd25519})
	return nil
}

func runResourceUpload(args []string) error {
	fs := flag.NewFlagSet("resource-upload", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	baseURL := fs.String("base-url", env("GAME_HELPER_RELEASE_BASE_URL", "http://127.0.0.1:18080"), "server base URL")
	bundlePath := fs.String("bundle", "", "bundle json from resource-pack")
	token := fs.String("token", env("GAME_HELPER_ADMIN_TOKEN", ""), "admin bearer token")
	username := fs.String("username", env("GAME_HELPER_ADMIN_USER", "admin"), "admin username")
	password := fs.String("password", env("GAME_HELPER_ADMIN_PASSWORD", ""), "admin password")
	publish := fs.Bool("publish", false, "publish resource version after upload")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*bundlePath) == "" {
		return fmt.Errorf("-bundle is required")
	}
	bundle, err := readBundle(*bundlePath)
	if err != nil {
		return err
	}
	authToken := strings.TrimSpace(*token)
	if authToken == "" {
		authToken, err = login(*baseURL, *username, *password)
		if err != nil {
			return err
		}
	}
	resp, err := uploadResourceBundle(*baseURL, authToken, bundle)
	if err != nil {
		return err
	}
	if *publish {
		resource, _ := resp["resource_version"].(map[string]any)
		id, _ := resource["id"].(string)
		if id == "" {
			return fmt.Errorf("upload response did not include resource_version.id")
		}
		if err := postEmpty(*baseURL, authToken, "/admin/api/app-resources/"+url.PathEscape(id)+"/publish"); err != nil {
			return err
		}
		resp["published"] = true
	}
	printJSON(resp)
	return nil
}

type sourceFlags []appreleases.ResourcePackageSource

func (s *sourceFlags) String() string {
	return fmt.Sprint([]appreleases.ResourcePackageSource(*s))
}

func (s *sourceFlags) Set(value string) error {
	key, dir, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(dir) == "" {
		return fmt.Errorf("source must be package_key=dir")
	}
	*s = append(*s, appreleases.ResourcePackageSource{PackageKey: strings.TrimSpace(key), Root: strings.TrimSpace(dir)})
	return nil
}

func discoverPackageSources(root string) ([]appreleases.ResourcePackageSource, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var result []appreleases.ResourcePackageSource
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		key := entry.Name()
		if err := appreleases.ValidateIncrementalResourcePackageKey(key); err != nil {
			continue
		}
		result = append(result, appreleases.ResourcePackageSource{
			PackageKey: key,
			Root:       filepath.Join(root, entry.Name()),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].PackageKey < result[j].PackageKey
	})
	return result, nil
}

func readBundle(path string) (appreleases.BuiltResourceBundle, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return appreleases.BuiltResourceBundle{}, err
	}
	var bundle appreleases.BuiltResourceBundle
	if err := json.Unmarshal(body, &bundle); err != nil {
		return appreleases.BuiltResourceBundle{}, err
	}
	if len(bundle.Packages) == 0 {
		return appreleases.BuiltResourceBundle{}, fmt.Errorf("%s has no packages", path)
	}
	return bundle, nil
}

func login(baseURL, username, password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("admin password is required when token is empty")
	}
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequest(http.MethodPost, joinURL(baseURL, "/api/v1/admin/auth/login"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("login failed: %s", resp.Status)
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	token, _ := decoded["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("login response did not include access_token")
	}
	return token, nil
}

func uploadResourceBundle(baseURL, token string, bundle appreleases.BuiltResourceBundle) (map[string]any, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadataBytes, err := json.Marshal(bundle.Metadata)
	if err != nil {
		return nil, err
	}
	if err := writer.WriteField("metadata", string(metadataBytes)); err != nil {
		return nil, err
	}
	for _, pkg := range bundle.Packages {
		part, err := writer.CreateFormFile("packages", filepath.Base(pkg.Path))
		if err != nil {
			return nil, err
		}
		file, err := os.Open(pkg.Path)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(part, file)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, joinURL(baseURL, "/admin/api/app-resources"), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("resource upload failed: %s %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func uploadArtifactFile(baseURL, token string, metadata appreleases.CreateArtifactRequest, path string) (map[string]any, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if err := writer.WriteField("metadata", string(metadataBytes)); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, copyErr := io.Copy(part, file)
	closeErr := file.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, joinURL(baseURL, "/api/v1/ci/artifacts"), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	return decodeJSONResponse(resp, err, "artifact upload")
}

func registerArtifactURL(baseURL, token string, metadata appreleases.CreateArtifactRequest) (map[string]any, error) {
	body, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, joinURL(baseURL, "/api/v1/ci/artifacts"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	return decodeJSONResponse(resp, err, "artifact registration")
}

func decodeJSONResponse(resp *http.Response, err error, operation string) (map[string]any, error) {
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("%s failed: %s %s", operation, resp.Status, strings.TrimSpace(string(msg)))
	}
	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func postEmpty(baseURL, token, path string) error {
	req, err := http.NewRequest(http.MethodPost, joinURL(baseURL, path), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s failed: %s %s", path, resp.Status, strings.TrimSpace(string(msg)))
	}
	return nil
}

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}

func defaultResourceVersion(now time.Time) string {
	return now.UTC().Format("20060102.150405")
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func printJSON(value any) {
	body, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(body))
}
