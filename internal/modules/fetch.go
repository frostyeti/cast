package modules

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/go/exec"
)

func FetchModule(projectDir string, uri string) (string, error) {
	// Create cache directory .cast/modules
	modulesDir := filepath.Join(projectDir, ".cast", "modules")
	os.MkdirAll(modulesDir, 0755)

	// Key directory by hash of URI to ensure unique version keys
	hash := sha256.Sum256([]byte(uri))
	hashStr := hex.EncodeToString(hash[:])
	
	// Create human readable suffix based on uri if possible
	parts := strings.Split(uri, "/")
	namePart := parts[len(parts)-1]
	namePart = strings.ReplaceAll(namePart, ".tar.gz", "")
	namePart = strings.ReplaceAll(namePart, ".git", "")

	targetDir := filepath.Join(modulesDir, fmt.Sprintf("%s-%s", namePart, hashStr[:8]))

	if _, err := os.Stat(targetDir); err == nil {
		// Already cached
		return targetDir, nil
	}

	if strings.HasSuffix(uri, ".tar.gz") || strings.HasPrefix(uri, "http") && strings.Contains(uri, "tar.gz") {
		err := fetchTarball(uri, targetDir)
		if err != nil {
			return "", err
		}
		return targetDir, nil
	}

	if strings.HasPrefix(uri, "github.com/") || strings.HasPrefix(uri, "git@") || strings.HasSuffix(uri, ".git") {
		err := fetchGit(uri, targetDir)
		if err != nil {
			return "", err
		}
		return targetDir, nil
	}

	return "", errors.Newf("unsupported module URI: %s", uri)
}

func fetchGit(uri, targetDir string) error {
	// Handle github.com/user/repo@v1.0.0
	parts := strings.Split(uri, "@")
	repoPath := parts[0]
	version := "main"
	if len(parts) > 1 {
		version = parts[1]
	}

	repoURL := repoPath
	if strings.HasPrefix(repoPath, "github.com/") {
		repoURL = "https://" + repoPath
		if !strings.HasSuffix(repoURL, ".git") {
			repoURL += ".git"
		}
	}

	cmd := exec.New("git", "clone", "--depth", "1", "--branch", version, repoURL, targetDir)
	out, err := cmd.Run()
	if err != nil || out.Code != 0 {
		return errors.Newf("failed to clone module %s: %v\n%s", uri, err, out.Stdout)
	}

	return nil
}

func fetchTarball(uri, targetDir string) error {
	resp, err := http.Get(uri)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Newf("failed to download %s: status %d", uri, resp.StatusCode)
	}

	os.MkdirAll(targetDir, 0755)

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Some tarballs contain a root directory, we might want to strip it,
		// but for simplicity we just extract as is.
		targetPath := filepath.Join(targetDir, header.Name)
		
		// Protect against directory traversal
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) && targetPath != filepath.Clean(targetDir) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(targetPath), 0755)
			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			os.Chmod(targetPath, os.FileMode(header.Mode))
		}
	}

	return nil
}
