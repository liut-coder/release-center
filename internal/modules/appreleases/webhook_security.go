package appreleases

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
)

const webhookSignatureBodyLimit = 2 << 20

func WebhookSignatureMiddleware(githubSecret, giteaSecret string) func(http.Handler) http.Handler {
	githubSecret = strings.TrimSpace(githubSecret)
	giteaSecret = strings.TrimSpace(giteaSecret)
	return func(next http.Handler) http.Handler {
		if githubSecret == "" && giteaSecret == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provider := webhookProviderFromPath(r.URL.Path)
			secret := webhookSecretForProvider(provider, githubSecret, giteaSecret)
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(io.LimitReader(r.Body, webhookSignatureBodyLimit+1))
			if err != nil {
				http.Error(w, "webhook body read failed", http.StatusBadRequest)
				return
			}
			if len(body) > webhookSignatureBodyLimit {
				http.Error(w, "webhook body too large", http.StatusRequestEntityTooLarge)
				return
			}
			if err := verifyWebhookSignature(provider, r.Header, body, secret); err != nil {
				http.Error(w, "webhook signature invalid", http.StatusUnauthorized)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func webhookProviderFromPath(path string) string {
	path = strings.TrimSpace(path)
	switch {
	case strings.HasSuffix(path, "/github"):
		return "github"
	case strings.HasSuffix(path, "/gitea"):
		return "gitea"
	default:
		return ""
	}
}

func webhookSecretForProvider(provider, githubSecret, giteaSecret string) string {
	switch provider {
	case "github":
		return githubSecret
	case "gitea":
		return giteaSecret
	default:
		return ""
	}
}

func verifyWebhookSignature(provider string, header http.Header, body []byte, secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}
	switch provider {
	case "github":
		if signature := strings.TrimSpace(header.Get("X-Hub-Signature-256")); signature != "" {
			if verifyWebhookHMAC(secret, body, signature, "sha256", sha256.New) {
				return nil
			}
			return fmt.Errorf("github sha256 signature mismatch")
		}
		if signature := strings.TrimSpace(header.Get("X-Hub-Signature")); signature != "" {
			if verifyWebhookHMAC(secret, body, signature, "sha1", sha1.New) {
				return nil
			}
			return fmt.Errorf("github sha1 signature mismatch")
		}
		return fmt.Errorf("github signature header missing")
	case "gitea":
		if signature := strings.TrimSpace(header.Get("X-Gitea-Signature")); signature != "" {
			if verifyWebhookHMAC(secret, body, signature, "sha256", sha256.New) {
				return nil
			}
			return fmt.Errorf("gitea signature mismatch")
		}
		if signature := strings.TrimSpace(header.Get("X-Hub-Signature-256")); signature != "" {
			if verifyWebhookHMAC(secret, body, signature, "sha256", sha256.New) {
				return nil
			}
			return fmt.Errorf("gitea github-compatible signature mismatch")
		}
		return fmt.Errorf("gitea signature header missing")
	default:
		return nil
	}
}

func verifyWebhookHMAC(secret string, body []byte, signature, prefix string, newHash func() hash.Hash) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	mac := hmac.New(newHash, []byte(secret))
	_, _ = mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))
	expected := expectedHex
	if strings.Contains(signature, "=") {
		expected = prefix + "=" + expectedHex
	}
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}
