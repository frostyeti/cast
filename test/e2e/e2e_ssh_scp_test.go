//go:build integration
// +build integration

package e2e_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const sshdImage = "lscr.io/linuxserver/openssh-server:latest"

func TestE2E_SSHTask(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	tmpDir := t.TempDir()
	binPath := buildCastBinary(t, tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container := startSSHServer(t, ctx)
	defer testcontainers.CleanupContainer(t, container)
	prepareRemoteConfig(t, ctx, container, tmpDir)

	host, port := containerHostPort(t, ctx, container, "2222/tcp")
	projectDir := filepath.Join(tmpDir, "ssh-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	castfile := "" +
		"name: ssh-e2e\n" +
		"inventory:\n" +
		"  hosts:\n" +
		"    sshbox:\n" +
		"      host: " + host + "\n" +
		"      port: " + port + "\n" +
		"      user: cast\n" +
		"      password: cast-pass\n" +
		"tasks:\n" +
		"  remote-shell:\n" +
		"    uses: ssh\n" +
		"    hosts: [sshbox]\n" +
		"    run: |\n" +
		"      echo SSH_OK\n" +
		"      test -f /config/outgoing/from-server-one.txt\n" +
		"      test -f /config/outgoing/from-server-two.txt\n"

	writeFile(t, filepath.Join(projectDir, "castfile"), []byte(castfile), 0o644)

	out := runCast(t, projectDir, binPath, "remote-shell")
	require.Contains(t, out, "SSH_OK")
}

func TestE2E_SSHRootSubcommand(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	tmpDir := t.TempDir()
	binPath := buildCastBinary(t, tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container := startSSHServer(t, ctx)
	defer testcontainers.CleanupContainer(t, container)
	prepareRemoteConfig(t, ctx, container, tmpDir)

	host, port := containerHostPort(t, ctx, container, "2222/tcp")
	projectDir := filepath.Join(tmpDir, "ssh-root-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	castfile := "" +
		"name: ssh-root-e2e\n" +
		"inventory:\n" +
		"  hosts:\n" +
		"    sshbox:\n" +
		"      host: " + host + "\n" +
		"      port: " + port + "\n" +
		"      user: cast\n" +
		"      password: cast-pass\n"

	writeFile(t, filepath.Join(projectDir, "castfile"), []byte(castfile), 0o644)

	scriptPath := filepath.Join(projectDir, "remote-check.sh")
	writeFile(t, scriptPath, []byte("echo ROOT_SSH_OK\ntest -f /config/outgoing/from-server-one.txt\n"), 0o755)

	out := runCast(t, projectDir, binPath, "ssh", "sshbox", "--script", scriptPath)
	require.Contains(t, out, "ROOT_SSH_OK")
}

func TestE2E_SCPTasks_MultiFile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	tmpDir := t.TempDir()
	binPath := buildCastBinary(t, tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container := startSSHServer(t, ctx)
	defer testcontainers.CleanupContainer(t, container)
	prepareRemoteConfig(t, ctx, container, tmpDir)

	host, port := containerHostPort(t, ctx, container, "2222/tcp")
	projectDir := filepath.Join(tmpDir, "scp-project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "dir"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "relative-src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "relative-downloads"), 0o755))

	writeFile(t, filepath.Join(projectDir, "single.txt"), []byte("single"), 0o644)
	writeFile(t, filepath.Join(projectDir, "dir", "one.txt"), []byte("one"), 0o644)
	writeFile(t, filepath.Join(projectDir, "dir", "two.txt"), []byte("two"), 0o644)
	writeFile(t, filepath.Join(projectDir, "relative-src", "one.txt"), []byte("relative-one"), 0o644)
	writeFile(t, filepath.Join(projectDir, "relative-src", "two.txt"), []byte("relative-two"), 0o644)

	absDownloadOne := filepath.ToSlash(filepath.Join(projectDir, "downloaded-one.txt"))
	absDownloadTwo := filepath.ToSlash(filepath.Join(projectDir, "downloaded-two.txt"))

	castfile := "" +
		"name: scp-e2e\n" +
		"env:\n" +
		"  REL_SOURCE_DIR: ./relative-src\n" +
		"  REL_DOWNLOAD_DIR: ./relative-downloads\n" +
		"  REMOTE_OUT_DIR: /config/outgoing\n" +
		"inventory:\n" +
		"  hosts:\n" +
		"    sshbox:\n" +
		"      host: " + host + "\n" +
		"      port: " + port + "\n" +
		"      user: cast\n" +
		"      password: cast-pass\n" +
		"tasks:\n" +
		"  upload-absolute:\n" +
		"    uses: scp\n" +
		"    hosts: [sshbox]\n" +
		"    with:\n" +
		"      files:\n" +
		"        - " + filepath.ToSlash(filepath.Join(projectDir, "single.txt")) + ":/config/incoming/single.txt\n" +
		"        - " + filepath.ToSlash(filepath.Join(projectDir, "dir", "one.txt")) + ":/config/incoming/nested/one.txt\n" +
		"        - " + filepath.ToSlash(filepath.Join(projectDir, "dir", "two.txt")) + ":/config/incoming/nested/two.txt\n" +
		"\n" +
		"  upload-relative:\n" +
		"    uses: scp\n" +
		"    hosts: [sshbox]\n" +
		"    with:\n" +
		"      files:\n" +
		"        - ./relative-src/one.txt:/config/incoming/relative/one.txt\n" +
		"        - $REL_SOURCE_DIR/two.txt:/config/incoming/relative/two.txt\n" +
		"        - ./relative-src/missing.txt?:/config/incoming/relative/missing.txt\n" +
		"\n" +
		"  verify-upload:\n" +
		"    uses: ssh\n" +
		"    hosts: [sshbox]\n" +
		"    run: |\n" +
		"      test -f /config/incoming/single.txt\n" +
		"      test -f /config/incoming/nested/one.txt\n" +
		"      test -f /config/incoming/nested/two.txt\n" +
		"      test -f /config/incoming/relative/one.txt\n" +
		"      test -f /config/incoming/relative/two.txt\n" +
		"      test ! -f /config/incoming/relative/missing.txt\n" +
		"\n" +
		"  download-absolute:\n" +
		"    uses: scp\n" +
		"    hosts: [sshbox]\n" +
		"    with:\n" +
		"      direction: download\n" +
		"      files:\n" +
		"        - /config/outgoing/from-server-one.txt:" + absDownloadOne + "\n" +
		"        - /config/outgoing/from-server-two.txt:" + absDownloadTwo + "\n" +
		"\n" +
		"  download-relative:\n" +
		"    uses: scp\n" +
		"    hosts: [sshbox]\n" +
		"    with:\n" +
		"      direction: download\n" +
		"      files:\n" +
		"        - $REMOTE_OUT_DIR/from-server-one.txt:$REL_DOWNLOAD_DIR/one.txt\n" +
		"        - /config/outgoing/from-server-two.txt:./relative-downloads/two.txt\n" +
		"        - /config/outgoing/missing.txt?:$REL_DOWNLOAD_DIR/missing.txt\n"

	writeFile(t, filepath.Join(projectDir, "castfile"), []byte(castfile), 0o644)

	uploadOut := runCast(t, projectDir, binPath, "upload-absolute")
	require.Contains(t, uploadOut, "Transfer complete")

	relativeUploadOut := runCast(t, projectDir, binPath, "upload-relative")
	require.Contains(t, relativeUploadOut, "Transfer complete")

	verifyOut := runCast(t, projectDir, binPath, "verify-upload")
	require.NotContains(t, verifyOut, "failed")

	downloadOut := runCast(t, projectDir, binPath, "download-absolute")
	require.Contains(t, downloadOut, "Transfer complete")

	relativeDownloadOut := runCast(t, projectDir, binPath, "download-relative")
	require.Contains(t, relativeDownloadOut, "Transfer complete")

	first := readFile(t, absDownloadOne)
	require.Equal(t, "server-one", strings.TrimSpace(first))

	second := readFile(t, absDownloadTwo)
	require.Equal(t, "server-two", strings.TrimSpace(second))

	relativeFirst := readFile(t, filepath.Join(projectDir, "relative-downloads", "one.txt"))
	require.Equal(t, "server-one", strings.TrimSpace(relativeFirst))

	relativeSecond := readFile(t, filepath.Join(projectDir, "relative-downloads", "two.txt"))
	require.Equal(t, "server-two", strings.TrimSpace(relativeSecond))

	_, statErr := os.Stat(filepath.Join(projectDir, "relative-downloads", "missing.txt"))
	require.Error(t, statErr)
	require.True(t, os.IsNotExist(statErr))
}

func TestE2E_SCPRootSubcommand(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)

	tmpDir := t.TempDir()
	binPath := buildCastBinary(t, tmpDir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container := startSSHServer(t, ctx)
	defer testcontainers.CleanupContainer(t, container)
	prepareRemoteConfig(t, ctx, container, tmpDir)

	host, port := containerHostPort(t, ctx, container, "2222/tcp")
	projectDir := filepath.Join(tmpDir, "scp-root-project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	castfile := "" +
		"name: scp-root-e2e\n" +
		"inventory:\n" +
		"  hosts:\n" +
		"    sshbox:\n" +
		"      host: " + host + "\n" +
		"      port: " + port + "\n" +
		"      user: cast\n" +
		"      password: cast-pass\n"

	writeFile(t, filepath.Join(projectDir, "castfile"), []byte(castfile), 0o644)

	localUpload := filepath.Join(projectDir, "root-upload.txt")
	localDownload := filepath.Join(projectDir, "root-download.txt")
	verifyScript := filepath.Join(projectDir, "verify-upload.sh")
	writeFile(t, localUpload, []byte("root-scp-upload"), 0o644)
	writeFile(t, verifyScript, []byte("test -f /config/incoming/root-upload.txt\n"), 0o755)

	uploadOut := runCast(t, projectDir, binPath, "scp", "-t", "sshbox", localUpload, "/config/incoming/root-upload.txt")
	require.Contains(t, uploadOut, "Success for sshbox")

	_ = runCast(t, projectDir, binPath, "ssh", "sshbox", "--script", verifyScript)

	downloadOut := runCast(t, projectDir, binPath, "scp", "--pull", "-t", "sshbox", "/config/outgoing/from-server-two.txt", localDownload)
	require.Contains(t, downloadOut, "Success for sshbox")

	content := readFile(t, localDownload)
	require.Equal(t, "server-two", strings.TrimSpace(content))
}

func buildCastBinary(t *testing.T, tmpDir string) string {
	t.Helper()
	binPath := filepath.Join(tmpDir, "cast")
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}
	return binPath
}

func runCast(t *testing.T, dir, binPath string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cast %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func startSSHServer(t *testing.T, ctx context.Context) testcontainers.Container {
	t.Helper()
	container, err := testcontainers.Run(ctx, sshdImage,
		testcontainers.WithEnv(map[string]string{
			"PUID":            "1000",
			"PGID":            "1000",
			"TZ":              "Etc/UTC",
			"PASSWORD_ACCESS": "true",
			"USER_PASSWORD":   "cast-pass",
			"SUDO_ACCESS":     "false",
			"USER_NAME":       "cast",
		}),
		testcontainers.WithExposedPorts("2222/tcp"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("2222/tcp").WithStartupTimeout(2*time.Minute)),
	)
	require.NoError(t, err)
	return container
}

func prepareRemoteConfig(t *testing.T, ctx context.Context, container testcontainers.Container, tmpDir string) {
	t.Helper()
	remoteConfig := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(filepath.Join(remoteConfig, "incoming", "nested"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(remoteConfig, "incoming", "relative"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(remoteConfig, "outgoing"), 0o755))
	writeFile(t, filepath.Join(remoteConfig, "outgoing", "from-server-one.txt"), []byte("server-one"), 0o644)
	writeFile(t, filepath.Join(remoteConfig, "outgoing", "from-server-two.txt"), []byte("server-two"), 0o644)
	require.NoError(t, container.CopyDirToContainer(ctx, remoteConfig, "/", 0o755))
	_, _, err := container.Exec(ctx, []string{"chmod", "-R", "777", "/config/incoming"})
	require.NoError(t, err)
}

func containerHostPort(t *testing.T, ctx context.Context, container testcontainers.Container, port nat.Port) (string, string) {
	t.Helper()
	host, err := container.Host(ctx)
	require.NoError(t, err)
	mapped, err := container.MappedPort(ctx, port)
	require.NoError(t, err)
	return host, mapped.Port()
}

func writeFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, data, mode))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
