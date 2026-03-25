package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
	"github.com/frostyeti/go/env"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
)

const castGitHubRepo = "frostyeti/cast"

type githubRelease struct {
	TagName string               `json:"tag_name"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

var (
	fetchLatestReleaseFunc    = fetchLatestRelease
	downloadURLBytesFunc      = downloadURLBytes
	resolveExecutablePathFunc = os.Executable
	replaceExecutableFileFunc = replaceExecutableFile
)

var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Manage cast itself",
}

var selfUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cast from latest GitHub release",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSelfUpgrade(cmd)
	},
}

var selfConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage castfile config values",
}

var selfConfigSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value in castfile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		return setProjectConfigValue(projectFile, args[0], args[1])
	},
}

var selfConfigGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value from castfile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		value, found, err := getProjectConfigValue(projectFile, args[0])
		if err != nil {
			return err
		}
		if !found {
			return errors.Newf("config key not found: %s", args[0])
		}

		_, _ = fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	},
}

var selfConfigRmCmd = &cobra.Command{
	Use:   "rm <key>",
	Short: "Remove a config value from castfile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectFile, err := resolveProjectFileFromFlagOrCwd(cmd)
		if err != nil {
			return err
		}

		return removeProjectConfigValue(projectFile, args[0])
	},
}

func init() {
	rootCmd.AddCommand(selfCmd)
	selfCmd.AddCommand(selfUpgradeCmd)
	selfCmd.AddCommand(selfConfigCmd)
	selfConfigCmd.AddCommand(selfConfigSetCmd)
	selfConfigCmd.AddCommand(selfConfigGetCmd)
	selfConfigCmd.AddCommand(selfConfigRmCmd)

	project := env.Get("CAST_PROJECT")
	context := env.Get("CAST_CONTEXT")
	selfCmd.PersistentFlags().StringP("project", "p", project, "Path to the project file (castfile.yaml)")
	selfCmd.PersistentFlags().StringP("context", "c", context, "Context name to use from the project")
}

func runSelfUpgrade(cmd *cobra.Command) error {
	exePath, err := resolveExecutablePathFunc()
	if err != nil {
		return errors.Newf("failed to resolve current executable path: %w", err)
	}

	release, err := fetchLatestReleaseFunc(castGitHubRepo)
	if err != nil {
		return err
	}

	asset, err := findReleaseAsset(release)
	if err != nil {
		return err
	}

	archiveBytes, err := downloadURLBytesFunc(asset.BrowserDownloadURL)
	if err != nil {
		return errors.Newf("failed to download release asset %s: %w", asset.Name, err)
	}

	binaryName := "cast"
	if runtime.GOOS == "windows" {
		binaryName = "cast.exe"
	}

	binaryBytes, fileMode, err := extractBinaryFromReleaseArchive(archiveBytes, asset.Name, binaryName)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(dir, ".cast-upgrade-*")
	if err != nil {
		return errors.Newf("failed to create temp file for upgrade: %w", err)
	}

	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(binaryBytes); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return errors.Newf("failed to write new cast binary: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return errors.Newf("failed to close temp binary file: %w", err)
	}

	if fileMode == 0 {
		fileMode = 0o755
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, fileMode); err != nil {
			_ = os.Remove(tmpPath)
			return errors.Newf("failed to set executable permissions: %w", err)
		}
	}

	if err := replaceExecutableFileFunc(exePath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Upgraded cast to %s\n", release.TagName)
	return nil
}

func fetchLatestRelease(repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cast-cli")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Newf("failed to fetch latest release: status %d (%s)", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	release := &githubRelease{}
	if err := json.NewDecoder(resp.Body).Decode(release); err != nil {
		return nil, errors.Newf("failed to decode GitHub release response: %w", err)
	}

	return release, nil
}

func downloadURLBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Newf("download failed with status %d (%s)", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

func findReleaseAsset(release *githubRelease) (*githubReleaseAsset, error) {
	if release == nil {
		return nil, errors.New("release payload is empty")
	}

	candidates := expectedReleaseAssetNames(release.TagName, runtime.GOOS, runtime.GOARCH)
	for _, candidate := range candidates {
		for _, asset := range release.Assets {
			if asset.Name == candidate {
				selected := asset
				return &selected, nil
			}
		}
	}

	return nil, errors.Newf("no matching release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func expectedReleaseAssetNames(tag, goos, goarch string) []string {
	version := strings.TrimSpace(tag)
	versionNoV := strings.TrimPrefix(version, "v")
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}

	baseWithV := fmt.Sprintf("cast-%s-%s-v%s", goos, goarch, versionNoV)
	baseNoV := fmt.Sprintf("cast-%s-%s-%s", goos, goarch, versionNoV)

	return []string{baseWithV + ext, baseNoV + ext}
}

func extractBinaryFromReleaseArchive(archive []byte, assetName, binaryName string) ([]byte, os.FileMode, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractBinaryFromZip(archive, binaryName)
	}

	if strings.HasSuffix(assetName, ".tar.gz") {
		return extractBinaryFromTarGz(archive, binaryName)
	}

	return nil, 0, errors.Newf("unsupported archive format for asset %s", assetName)
}

func extractBinaryFromZip(archive []byte, binaryName string) ([]byte, os.FileMode, error) {
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, 0, errors.Newf("failed to read zip archive: %w", err)
	}

	for _, file := range r.File {
		if filepath.Base(file.Name) != binaryName {
			continue
		}
		fr, err := file.Open()
		if err != nil {
			return nil, 0, err
		}
		defer func() { _ = fr.Close() }()

		bytes, err := io.ReadAll(fr)
		if err != nil {
			return nil, 0, err
		}
		return bytes, file.Mode(), nil
	}

	return nil, 0, errors.Newf("binary %s not found in zip archive", binaryName)
}

func extractBinaryFromTarGz(archive []byte, binaryName string) ([]byte, os.FileMode, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, 0, errors.Newf("failed to read gzip archive: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		if hdr == nil || hdr.FileInfo() == nil || !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}

		bytes, err := io.ReadAll(tr)
		if err != nil {
			return nil, 0, err
		}
		return bytes, hdr.FileInfo().Mode(), nil
	}

	return nil, 0, errors.Newf("binary %s not found in tar.gz archive", binaryName)
}

func replaceExecutableFile(currentPath, newPath string) error {
	if err := os.Rename(newPath, currentPath); err == nil {
		return nil
	}

	backupPath := currentPath + ".bak"
	_ = os.Remove(backupPath)

	if err := os.Rename(currentPath, backupPath); err != nil {
		return errors.Newf("failed to replace executable: %w", err)
	}

	if err := os.Rename(newPath, currentPath); err != nil {
		_ = os.Rename(backupPath, currentPath)
		return errors.Newf("failed to replace executable: %w", err)
	}

	_ = os.Remove(backupPath)
	return nil
}

func setProjectConfigValue(projectFile, key, rawValue string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("config key cannot be empty")
	}

	doc, rootMap, err := loadProjectYAML(projectFile)
	if err != nil {
		return err
	}

	configMap := ensureMappingField(rootMap, "config")

	valueNode, err := configValueNodeForSet(key, rawValue)
	if err != nil {
		return err
	}

	setMappingField(configMap, key, valueNode)
	return saveProjectYAML(projectFile, doc)
}

func getProjectConfigValue(projectFile, key string) (string, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, errors.New("config key cannot be empty")
	}

	project := &projects.Project{}
	if err := project.LoadFromYaml(projectFile); err != nil {
		return "", false, err
	}

	if project.Schema.Config == nil {
		return "", false, nil
	}

	switch key {
	case "context":
		if project.Schema.Config.Context == nil {
			return "", false, nil
		}
		return strings.TrimSpace(*project.Schema.Config.Context), true, nil
	case "substitution":
		if project.Schema.Config.Substitution == nil {
			return "", false, nil
		}
		if *project.Schema.Config.Substitution {
			return "true", true, nil
		}
		return "false", true, nil
	default:
		if project.Schema.Config.Values == nil {
			return "", false, nil
		}
		v, ok := project.Schema.Config.Values[key]
		if !ok {
			return "", false, nil
		}
		bytes, err := yaml.Marshal(v)
		if err != nil {
			return "", false, err
		}
		return strings.TrimSpace(string(bytes)), true, nil
	}
}

func removeProjectConfigValue(projectFile, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("config key cannot be empty")
	}

	doc, rootMap, err := loadProjectYAML(projectFile)
	if err != nil {
		return err
	}

	configMap, found := getMappingField(rootMap, "config")
	if !found || configMap.Kind != yaml.MappingNode {
		return nil
	}

	_ = removeMappingField(configMap, key)
	if len(configMap.Content) == 0 {
		_ = removeMappingField(rootMap, "config")
	}

	return saveProjectYAML(projectFile, doc)
}

func loadProjectYAML(projectFile string) (*yaml.Node, *yaml.Node, error) {
	bytes, err := os.ReadFile(projectFile)
	if err != nil {
		return nil, nil, err
	}

	doc := &yaml.Node{}
	if len(strings.TrimSpace(string(bytes))) == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
		return doc, doc.Content[0], nil
	}

	if err := yaml.Unmarshal(bytes, doc); err != nil {
		return nil, nil, err
	}
	if doc.Kind != yaml.DocumentNode {
		return nil, nil, errors.New("invalid castfile yaml document")
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil, errors.New("castfile root must be a mapping")
	}

	return doc, doc.Content[0], nil
}

func saveProjectYAML(projectFile string, doc *yaml.Node) error {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return errors.New("invalid yaml document")
	}

	bytes, err := yaml.Marshal(doc.Content[0])
	if err != nil {
		return err
	}

	return os.WriteFile(projectFile, bytes, 0o644)
}

func getMappingField(root *yaml.Node, key string) (*yaml.Node, bool) {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, false
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1], true
		}
	}
	return nil, false
}

func ensureMappingField(root *yaml.Node, key string) *yaml.Node {
	if value, found := getMappingField(root, key); found && value.Kind == yaml.MappingNode {
		return value
	}

	next := &yaml.Node{Kind: yaml.MappingNode}
	setMappingField(root, key, next)
	return next
}

func setMappingField(root *yaml.Node, key string, value *yaml.Node) {
	if root == nil || root.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return
		}
	}
	root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, value)
}

func removeMappingField(root *yaml.Node, key string) bool {
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content = append(root.Content[:i], root.Content[i+2:]...)
			return true
		}
	}
	return false
}

func configValueNodeForSet(key, rawValue string) (*yaml.Node, error) {
	if key == "context" {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: rawValue}, nil
	}

	if key == "substitution" {
		v := strings.TrimSpace(strings.ToLower(rawValue))
		switch v {
		case "true", "yes", "1", "on":
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"}, nil
		case "false", "no", "0", "off":
			return &yaml.Node{Kind: yaml.ScalarNode, Value: "false", Tag: "!!bool"}, nil
		default:
			return nil, errors.New("substitution must be true or false")
		}
	}

	var value any
	if err := yaml.Unmarshal([]byte(rawValue), &value); err != nil {
		value = rawValue
	}

	node := &yaml.Node{}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(encoded, node); err != nil {
		return nil, err
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0], nil
	}

	return node, nil
}
