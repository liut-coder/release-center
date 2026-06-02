package appreleases

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildResourceBundlePackagesAllowedIncrementalResources(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	writeTestFile(t, filepath.Join(root, "templates-bear", "bear.png"), "png")
	writeTestFile(t, filepath.Join(root, "templates-bear", "metadata.json"), `{"name":"bear"}`)
	writeTestFile(t, filepath.Join(root, "ocr-dictionary", "keywords.txt"), "bear\n")

	bundle, err := BuildResourceBundle(BuildResourceBundleRequest{
		AppKey:            "game-helper-android",
		ResourceVersion:   "20260602.1",
		Channel:           "dev",
		UpdateLevel:       "forced",
		Title:             "运行资源更新",
		Summary:           "打熊模板和 OCR 关键词",
		MinAppVersionCode: 10,
		OutputDir:         out,
		Sources: []ResourcePackageSource{
			{PackageKey: "templates-bear", Root: filepath.Join(root, "templates-bear")},
			{PackageKey: "ocr-dictionary", Root: filepath.Join(root, "ocr-dictionary")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Packages) != 2 {
		t.Fatalf("expected two packages, got %+v", bundle.Packages)
	}
	for _, pkg := range bundle.Packages {
		if pkg.FileSize <= 0 || pkg.SHA256 == "" {
			t.Fatalf("package missing size/hash: %+v", pkg)
		}
		if _, err := os.Stat(pkg.Path); err != nil {
			t.Fatalf("package %s not written: %v", pkg.Path, err)
		}
		body, err := os.ReadFile(pkg.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateIncrementalResourceArchive(pkg.PackageKey, body); err != nil {
			t.Fatalf("built package did not pass archive validation: %v", err)
		}
	}
	if _, err := os.Stat(bundle.MetadataPath); err != nil {
		t.Fatalf("metadata not written: %v", err)
	}
	manifestBody, err := os.ReadFile(bundle.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["appKey"] != "game-helper-android" || manifest["resourceVersion"] != "20260602.1" {
		t.Fatalf("manifest missing app/resource identity: %s", manifestBody)
	}
	if manifest["updateLevel"] != "forced" {
		t.Fatalf("manifest missing update level: %s", manifestBody)
	}
	if manifest["signature"] != nil || manifest["signatureAlgorithm"] != nil {
		t.Fatalf("manifest should stay unsigned without private key: %s", manifestBody)
	}
}

func TestBuildResourceManifestSignsAndVerifiesEd25519(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(strings.NewReader(strings.Repeat("a", 64)))
	if err != nil {
		t.Fatal(err)
	}
	manifestBody, err := BuildResourceManifest(ResourceManifestRequest{
		AppKey:             "game-helper-android",
		ResourceVersion:    "20260602.3",
		Channel:            "stable",
		UpdateLevel:        "normal",
		Title:              "签名资源",
		ManifestPrivateKey: base64.StdEncoding.EncodeToString(privateKey),
	}, []AppResourcePackageAdmin{{
		PackageKey:  "templates-bear",
		PackageType: "zip",
		FileURL:     "/api/v1/app/resources/packages/pkg-1/download",
		FileSize:    12,
		SHA256:      strings.Repeat("a", 64),
	}})
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["signatureAlgorithm"] != ManifestSignatureAlgorithmEd25519 || manifest["signature"] == "" {
		t.Fatalf("manifest missing signature fields: %s", manifestBody)
	}
	publicKeyText := base64.StdEncoding.EncodeToString(publicKey)
	exportedPublicKey, err := ManifestPublicKeyFromPrivateKey(base64.StdEncoding.EncodeToString(privateKey))
	if err != nil {
		t.Fatal(err)
	}
	if exportedPublicKey != publicKeyText {
		t.Fatalf("exported public key mismatch: %s != %s", exportedPublicKey, publicKeyText)
	}
	if err := VerifyResourceManifestSignature(manifestBody, publicKeyText); err != nil {
		t.Fatalf("expected manifest signature to verify: %v\n%s", err, manifestBody)
	}
	tampered := bytes.Replace(manifestBody, []byte("20260602.3"), []byte("20260602.4"), 1)
	if err := VerifyResourceManifestSignature(tampered, publicKeyText); err == nil {
		t.Fatal("expected tampered manifest signature to fail")
	}
}

func TestBuildResourceBundleRejectsExecutableIncrementalContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "templates-bear", "classes.dex"), "dex")

	_, err := BuildResourceBundle(BuildResourceBundleRequest{
		ResourceVersion: "20260602.2",
		Title:           "非法资源",
		OutputDir:       filepath.Join(t.TempDir(), "out"),
		Sources: []ResourcePackageSource{
			{PackageKey: "templates-bear", Root: filepath.Join(root, "templates-bear")},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Dex") {
		t.Fatalf("expected dex rejection, got %v", err)
	}
}

func TestValidateIncrementalResourceArchiveRejectsForbiddenZipEntries(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writer, err := zw.Create("lib/arm64-v8a/libnative.so")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("native")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	err = ValidateIncrementalResourceArchive("templates-common", buf.Bytes())
	if err == nil || !strings.Contains(err.Error(), "原生动态库") {
		t.Fatalf("expected native library rejection, got %v", err)
	}
}

func TestPackageKeyFromFilenameStripsResourceVersion(t *testing.T) {
	got := packageKeyFromFilename("templates-bear-20260602.1.zip", "20260602.1")
	if got != "templates-bear" {
		t.Fatalf("package key = %q", got)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
