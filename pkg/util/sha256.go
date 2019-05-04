package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func DirToSha256(rootPath string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if _, err := io.Copy(h, file); err != nil {
				return err
			}
		} else {
			h.Write([]byte(info.Name()))
		}
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil)), err
}
