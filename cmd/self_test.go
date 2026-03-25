package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExpectedReleaseAssetNames(t *testing.T) {
	names := expectedReleaseAssetNames("v0.2.0-alpha.4", "linux", "amd64")
	if len(names) == 0 {
		t.Fatalf("expected release asset names")
	}
	if !strings.HasPrefix(names[0], "cast-linux-amd64-") {
		t.Fatalf("unexpected asset name: %s", names[0])
	}
}

func TestFindReleaseAsset(t *testing.T) {
	release := &githubRelease{
		TagName: "v1.2.3",
		Assets: []githubReleaseAsset{
			{Name: "not-it", BrowserDownloadURL: "https://example.com/nope"},
			{Name: expectedReleaseAssetNames("v1.2.3", runtime.GOOS, runtime.GOARCH)[0], BrowserDownloadURL: "https://example.com/ok"},
		},
	}

	asset, err := findReleaseAsset(release)
	if err != nil {
		t.Fatalf("findReleaseAsset returned error: %v", err)
	}
	if asset == nil || asset.BrowserDownloadURL != "https://example.com/ok" {
		t.Fatalf("unexpected asset selected: %#v", asset)
	}
}

func TestSetGetRemoveProjectConfigValues(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: demo\n"), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	if err := setProjectConfigValue(projectFile, "context", "prod"); err != nil {
		t.Fatalf("set context failed: %v", err)
	}
	if err := setProjectConfigValue(projectFile, "feature_flags", "[\"a\",\"b\"]"); err != nil {
		t.Fatalf("set custom value failed: %v", err)
	}

	v, found, err := getProjectConfigValue(projectFile, "context")
	if err != nil || !found || v != "prod" {
		t.Fatalf("unexpected context value: v=%q found=%v err=%v", v, found, err)
	}

	v, found, err = getProjectConfigValue(projectFile, "feature_flags")
	if err != nil || !found || !strings.Contains(v, "- a") {
		t.Fatalf("unexpected custom config value: v=%q found=%v err=%v", v, found, err)
	}

	if err := removeProjectConfigValue(projectFile, "feature_flags"); err != nil {
		t.Fatalf("remove custom value failed: %v", err)
	}

	_, found, err = getProjectConfigValue(projectFile, "feature_flags")
	if err != nil {
		t.Fatalf("get removed key failed: %v", err)
	}
	if found {
		t.Fatalf("expected removed key to be missing")
	}
}

func TestResolveDefaultContextNameFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: demo\nconfig:\n  context: prod\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("context", "c", "", "")
	if got := resolveDefaultContextName(cmd, projectFile); got != "prod" {
		t.Fatalf("resolveDefaultContextName returned %q, want prod", got)
	}
}

func TestRunSelfUpgrade_UsesDownloadedArchive(t *testing.T) {
	oldFetch := fetchLatestReleaseFunc
	oldDownload := downloadURLBytesFunc
	oldExe := resolveExecutablePathFunc
	oldReplace := replaceExecutableFileFunc
	defer func() {
		fetchLatestReleaseFunc = oldFetch
		downloadURLBytesFunc = oldDownload
		resolveExecutablePathFunc = oldExe
		replaceExecutableFileFunc = oldReplace
	}()

	assetName := expectedReleaseAssetNames("v1.2.3", runtime.GOOS, runtime.GOARCH)[0]

	fetchLatestReleaseFunc = func(repo string) (*githubRelease, error) {
		return &githubRelease{
			TagName: "v1.2.3",
			Assets:  []githubReleaseAsset{{Name: assetName, BrowserDownloadURL: "https://example.com/cast"}},
		}, nil
	}

	binaryName := "cast"
	if runtime.GOOS == "windows" {
		binaryName = "cast.exe"
	}

	archive, err := makeTestArchive(assetName, binaryName, []byte("new-binary"))
	if err != nil {
		t.Fatalf("make archive: %v", err)
	}

	downloadURLBytesFunc = func(url string) ([]byte, error) {
		return archive, nil
	}
	resolveExecutablePathFunc = func() (string, error) {
		return filepath.Join(t.TempDir(), binaryName), nil
	}

	replaced := false
	replaceExecutableFileFunc = func(currentPath, newPath string) error {
		replaced = true
		bytes, err := os.ReadFile(newPath)
		if err != nil {
			return err
		}
		if string(bytes) != "new-binary" {
			return errors.New("unexpected binary contents")
		}
		return nil
	}

	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := runSelfUpgrade(cmd); err != nil {
		t.Fatalf("runSelfUpgrade failed: %v", err)
	}
	if !replaced {
		t.Fatalf("expected executable replacement to run")
	}
	if !strings.Contains(buf.String(), "Upgraded cast to v1.2.3") {
		t.Fatalf("expected upgrade output, got: %s", buf.String())
	}
}

func makeTestArchive(assetName, binaryName string, payload []byte) ([]byte, error) {
	if strings.HasSuffix(assetName, ".zip") {
		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		f, err := zw.Create(binaryName)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(payload); err != nil {
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	buf := &bytes.Buffer{}
	gzw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gzw)
	hdr := &tar.Header{Name: binaryName, Mode: 0o755, Size: int64(len(payload))}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(payload); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
