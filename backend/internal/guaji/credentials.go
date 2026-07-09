package guaji

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrCredentialsKeyMissing = errors.New("GUAJI_CREDENTIALS_KEY 未配置")

// CredentialsKey derives a 32-byte AES key from env (hex/base64/raw) or JWT fallback for dev.
func CredentialsKey(primary, fallback string) ([]byte, error) {
	raw := strings.TrimSpace(primary)
	if raw == "" {
		raw = strings.TrimSpace(fallback)
	}
	if raw == "" {
		return nil, ErrCredentialsKeyMissing
	}
	if key, err := decodeKeyMaterial(raw); err == nil && len(key) == 32 {
		return key, nil
	}
	sum := sha256.Sum256([]byte(raw))
	return sum[:], nil
}

func decodeKeyMaterial(raw string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(raw); err == nil && len(b) > 0 {
		if len(b) == 32 {
			return b, nil
		}
		sum := sha256.Sum256(b)
		return sum[:], nil
	}
	if len(raw) == 64 {
		out := make([]byte, 32)
		for i := 0; i < 32; i++ {
			var v byte
			_, err := fmt.Sscanf(raw[i*2:i*2+2], "%02x", &v)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	return nil, fmt.Errorf("invalid key material")
}

func EncryptSecret(key []byte, plaintext string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("credentials key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func DecryptSecret(key []byte, ciphertext string) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("credentials key must be 32 bytes")
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
