package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liut-coder/game-helper-server/internal/modules/appreleases"
	"github.com/liut-coder/game-helper-server/internal/platform/httpx"
	"github.com/liut-coder/game-helper-server/internal/platform/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx := context.Background()

	cfg := appreleases.Config{
		AppKey:                  env("APP_KEY", "game-helper-android"),
		Name:                    env("APP_NAME", "游戏助手"),
		PackageName:             env("PACKAGE_NAME", "com.kingdomhelper.executor"),
		LatestVersionName:       env("LATEST_VERSION_NAME", ""),
		LatestVersionCode:       envInt("LATEST_VERSION_CODE", 0),
		BuildNumber:             envInt("BUILD_NUMBER", 0),
		Channel:                 env("CHANNEL", "dev"),
		BuildType:               env("BUILD_TYPE", "debug"),
		APKURL:                  env("APK_URL", ""),
		DownloadURL:             env("DOWNLOAD_URL", ""),
		SHA256:                  env("SHA256", ""),
		SizeBytes:               int64(envInt("SIZE_BYTES", 0)),
		FileName:                env("FILE_NAME", ""),
		ForceUpdate:             envBool("FORCE_UPDATE", false),
		CurrentVersionAvailable: envBool("CURRENT_VERSION_AVAILABLE", true),
		MinSupportedVersionCode: envInt("MIN_SUPPORTED_VERSION_CODE", 0),
		ReleaseNotes:            env("RELEASE_NOTES", ""),
		MessageZh:               env("MESSAGE_ZH", ""),
		UnavailableReason:       env("UNAVAILABLE_REASON", ""),
		ManifestPrivateKey:      env("MANIFEST_PRIVATE_KEY", env("GAME_HELPER_MANIFEST_PRIVATE_KEY", "")),
		QualityPolicy: appreleases.QualityPolicy{
			ResourceActivationFailedCount: envInt("QUALITY_RESOURCE_ACTIVATION_FAILED_COUNT", 0),
			ResourceFailureRate:           envInt("QUALITY_RESOURCE_FAILURE_RATE", 0),
			APKChecksumFailedCount:        envInt("QUALITY_APK_CHECKSUM_FAILED_COUNT", 0),
			APKInstallFailedCount:         envInt("QUALITY_APK_INSTALL_FAILED_COUNT", 0),
			APKInstallFailureRate:         envInt("QUALITY_APK_INSTALL_FAILURE_RATE", 0),
		},
	}

	var store appreleases.Store
	var db *pgxpool.Pool
	if databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); databaseURL != "" {
		var err error
		db, err = pgxpool.New(ctx, databaseURL)
		if err != nil {
			logger.Error("database pool create failed", "error", err)
			os.Exit(1)
		}
		defer db.Close()
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := db.Ping(pingCtx); err != nil {
			cancel()
			logger.Error("database ping failed", "error", err)
			os.Exit(1)
		}
		cancel()
		store = appreleases.NewPostgresStore(db)
	} else {
		logger.Info("database url not configured; using in-memory demo store")
		store = appreleases.NewDemoStore()
	}

	service := appreleases.NewServiceWithStore(cfg, store)
	if err := service.SyncConfiguredRelease(ctx); err != nil {
		logger.Error("configured release sync failed", "error", err)
		os.Exit(1)
	}

	handler := appreleases.NewHandlerWithBlobStore(service, storage.NewLocal(env("FILE_ROOT", "/srv/files")))
	r := chi.NewRouter()
	r.Use(httpx.RequestIDMiddleware)
	r.Use(httpx.AccessLogMiddleware(logger))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				http.Error(w, "database not ready", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
	adminAccounts := adminTokenAccounts(env("ADMIN_TOKEN", ""), env("ADMIN_TOKEN_ACCOUNTS", ""))
	r.Mount("/", handler.RoutesWithOptions(appreleases.RouteOptions{
		AdminMiddleware:           tokenAccountMiddlewares(adminAccounts),
		AdminPermissionMiddleware: appreleases.AdminRBACMiddleware(service, adminAccounts),
		CIMiddleware:              tokenMiddlewares(env("CI_TOKEN", env("GAME_HELPER_CI_TOKEN", ""))),
	}))
	mountWeb(r, logger, env("WEB_DIST", "web/dist"))

	addr := env("ADDR", ":18080")
	logger.Info("release center listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func mountWeb(r chi.Router, logger *slog.Logger, distDir string) {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		logger.Warn("admin web dist unavailable", "dir", distDir, "error", err)
		return
	}
	fileServer := http.FileServer(http.Dir(distDir))
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
	})
	r.Get("/admin", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/", http.StatusFound)
	})
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/admin/api/") {
			http.NotFound(w, req)
			return
		}
		path := strings.TrimPrefix(req.URL.Path, "/")
		if path == "" {
			http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
			return
		}
		cleanPath := strings.TrimPrefix(filepath.Clean("/"+path), "/")
		if info, err := os.Stat(filepath.Join(distDir, cleanPath)); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(distDir, "index.html"))
	})
}

func tokenMiddlewares(token string) []func(http.Handler) http.Handler {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return []func(http.Handler) http.Handler{appreleases.BearerTokenMiddleware(token)}
}

func tokenAccountMiddlewares(accounts map[string]string) []func(http.Handler) http.Handler {
	if len(accounts) == 0 {
		return nil
	}
	tokens := make([]string, 0, len(accounts))
	for token := range accounts {
		tokens = append(tokens, token)
	}
	return []func(http.Handler) http.Handler{appreleases.BearerTokenMiddleware(tokens...)}
}

func adminTokenAccounts(adminToken, mappings string) map[string]string {
	result := map[string]string{}
	if strings.TrimSpace(adminToken) != "" {
		result[strings.TrimSpace(adminToken)] = "system.admin"
	}
	for _, item := range strings.Split(mappings, ",") {
		token, account, ok := strings.Cut(item, ":")
		if !ok {
			continue
		}
		token = strings.TrimSpace(token)
		account = strings.TrimSpace(account)
		if token == "" || account == "" {
			continue
		}
		result[token] = account
	}
	return result
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
