package appreleases

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
)

const ManifestSignatureAlgorithmEd25519 = "ed25519"

func SignResourceManifest(manifest []byte, privateKeyText string) ([]byte, error) {
	privateKeyText = strings.TrimSpace(privateKeyText)
	if privateKeyText == "" {
		return manifest, nil
	}
	privateKey, err := parseEd25519PrivateKey(privateKeyText)
	if err != nil {
		return nil, err
	}
	payload, err := canonicalManifestSigningPayload(manifest)
	if err != nil {
		return nil, err
	}
	signature := ed25519.Sign(privateKey, payload)
	var decoded map[string]any
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		return nil, err
	}
	decoded["signatureAlgorithm"] = ManifestSignatureAlgorithmEd25519
	decoded["signature"] = base64.StdEncoding.EncodeToString(signature)
	return json.MarshalIndent(decoded, "", "  ")
}

func VerifyResourceManifestSignature(manifest []byte, publicKeyText string) error {
	publicKey, err := parseEd25519PublicKey(publicKeyText)
	if err != nil {
		return err
	}
	var decoded map[string]any
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		return err
	}
	algorithm, _ := decoded["signatureAlgorithm"].(string)
	if !strings.EqualFold(strings.TrimSpace(algorithm), ManifestSignatureAlgorithmEd25519) {
		return fmt.Errorf("manifest signature algorithm %q is not supported", algorithm)
	}
	signatureText, _ := decoded["signature"].(string)
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureText))
	if err != nil {
		return fmt.Errorf("manifest signature is not valid base64: %w", err)
	}
	payload, err := canonicalManifestSigningPayload(manifest)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return fmt.Errorf("manifest signature verification failed")
	}
	return nil
}

func ManifestPublicKeyFromPrivateKey(privateKeyText string) (string, error) {
	privateKey, err := parseEd25519PrivateKey(privateKeyText)
	if err != nil {
		return "", err
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return "", fmt.Errorf("private key did not expose an ed25519 public key")
	}
	return base64.StdEncoding.EncodeToString(publicKey), nil
}

func canonicalManifestSigningPayload(manifest []byte) ([]byte, error) {
	var decoded map[string]any
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		return nil, err
	}
	delete(decoded, "signatureAlgorithm")
	delete(decoded, "signature")
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(decoded); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

func parseEd25519PrivateKey(text string) (ed25519.PrivateKey, error) {
	if block, _ := pem.Decode([]byte(text)); block != nil {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse ed25519 private key pem: %w", err)
		}
		privateKey, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key pem is not ed25519")
		}
		return privateKey, nil
	}
	raw, err := decodeRawKey(text)
	if err != nil {
		return nil, fmt.Errorf("parse ed25519 private key: %w", err)
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	default:
		return nil, fmt.Errorf("expected 32 byte seed or 64 byte private key, got %d bytes", len(raw))
	}
}

func parseEd25519PublicKey(text string) (ed25519.PublicKey, error) {
	if block, _ := pem.Decode([]byte(strings.TrimSpace(text))); block != nil {
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse ed25519 public key pem: %w", err)
		}
		publicKey, ok := key.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key pem is not ed25519")
		}
		return publicKey, nil
	}
	raw, err := decodeRawKey(text)
	if err != nil {
		return nil, fmt.Errorf("parse ed25519 public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("expected 32 byte public key, got %d bytes", len(raw))
	}
	return ed25519.PublicKey(raw), nil
}

func decodeRawKey(text string) ([]byte, error) {
	compact := strings.TrimSpace(text)
	if strings.HasPrefix(compact, "base64:") {
		return base64.StdEncoding.DecodeString(strings.TrimPrefix(compact, "base64:"))
	}
	if strings.HasPrefix(compact, "hex:") {
		return hex.DecodeString(strings.TrimPrefix(compact, "hex:"))
	}
	if decoded, err := base64.StdEncoding.DecodeString(compact); err == nil && (len(decoded) == ed25519.SeedSize || len(decoded) == ed25519.PublicKeySize || len(decoded) == ed25519.PrivateKeySize) {
		return decoded, nil
	}
	compact = strings.ReplaceAll(compact, " ", "")
	compact = strings.ReplaceAll(compact, "\n", "")
	return hex.DecodeString(compact)
}
