package content

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrUploadUnavailable = errors.New("upload store unavailable")
	ErrUploadTooLarge      = errors.New("upload file too large")
	ErrUploadInvalidType   = errors.New("upload file type not allowed")
)

const MaxCMSUploadBytes = 5 << 20 // 5MB

type UploadStore struct {
	Dir string
}

func NewUploadStore(dir string) (*UploadStore, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = "./data/uploads/cms"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &UploadStore{Dir: abs}, nil
}

func (s *UploadStore) SaveImage(r io.Reader, maxBytes int64) (filename string, err error) {
	if s == nil || s.Dir == "" {
		return "", ErrUploadUnavailable
	}
	if maxBytes <= 0 {
		maxBytes = MaxCMSUploadBytes
	}
	limited := io.LimitReader(r, maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if int64(len(buf)) > maxBytes {
		return "", ErrUploadTooLarge
	}
	if len(buf) == 0 {
		return "", ErrUploadInvalidType
	}

	ext, err := imageExtFromBytes(buf)
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomHex(8), ext)
	path := filepath.Join(s.Dir, name)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func imageExtFromBytes(buf []byte) (string, error) {
	if len(buf) >= 3 && buf[0] == 0xFF && buf[1] == 0xD8 && buf[2] == 0xFF {
		return ".jpg", nil
	}
	if len(buf) >= 8 && string(buf[:8]) == "\x89PNG\r\n\x1a\n" {
		return ".png", nil
	}
	if len(buf) >= 6 && (string(buf[:6]) == "GIF87a" || string(buf[:6]) == "GIF89a") {
		return ".gif", nil
	}
	if len(buf) >= 12 && string(buf[:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
		return ".webp", nil
	}
	return "", ErrUploadInvalidType
}

func randomHex(n int) string {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)[:n]
}

func (s *UploadStore) FilePath(name string) (string, error) {
	if s == nil || s.Dir == "" {
		return "", ErrUploadUnavailable
	}
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || strings.Contains(name, "..") {
		return "", ErrUploadInvalidType
	}
	path := filepath.Join(s.Dir, name)
	absDir, err := filepath.Abs(s.Dir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absDir+string(os.PathSeparator)) && absPath != absDir {
		return "", ErrUploadInvalidType
	}
	return absPath, nil
}
