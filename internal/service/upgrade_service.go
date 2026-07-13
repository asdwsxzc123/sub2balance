package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrUpgradeInProgress   = errors.New("升级正在进行中")
	ErrContainerDeployment = errors.New("容器部署请通过拉取新镜像升级")
	ErrAlreadyUpToDate     = errors.New("当前已是最新版本，无需升级")
	ErrReleaseNotFound     = errors.New("找不到指定版本的发行版")
	ErrAssetNotFound       = errors.New("找不到当前平台对应的升级资产")
)

// DefaultRepoURL is the release source; override with SUB2BALANCE_REPO_URL
// (same variable deploy.sh uses).
const DefaultRepoURL = "https://github.com/asdwsxzc123/sub2balance"

const (
	upgradeHTTPTimeout = 60 * time.Second
	selfCheckTimeout   = 10 * time.Second
	restartDelay       = time.Second
	checksumsAssetName = "checksums.txt"
	// Generous caps for a Go binary release — anything larger is a corrupt or
	// malicious asset and must not exhaust the production host's disk.
	maxArchiveBytes = int64(500 << 20)
	maxBinaryBytes  = int64(500 << 20)
)

type UpgradeService struct {
	auditService *AuditService
	httpClient   *http.Client
	version      string
	owner        string
	repo         string

	// mu guards the whole upgrade flow. It is intentionally never unlocked on
	// the success path: a successful upgrade ends in a process restart, so the
	// lock only needs to block concurrent attempts until then.
	mu sync.Mutex
}

func NewUpgradeService(version string, auditService *AuditService) *UpgradeService {
	owner, repo := resolveRepo()
	return &UpgradeService{
		auditService: auditService,
		httpClient:   &http.Client{Timeout: upgradeHTTPTimeout},
		version:      version,
		owner:        owner,
		repo:         repo,
	}
}

// resolveRepo parses owner/repo from SUB2BALANCE_REPO_URL (or DefaultRepoURL).
// Falls back to the default repository when the override is unparsable.
func resolveRepo() (owner, repo string) {
	raw := os.Getenv("SUB2BALANCE_REPO_URL")
	if raw == "" {
		raw = DefaultRepoURL
	}
	if o, r, err := parseRepoURL(raw); err == nil {
		return o, r
	} else if raw != DefaultRepoURL {
		log.Printf("upgrade: invalid SUB2BALANCE_REPO_URL %q (%v), falling back to %s", raw, err, DefaultRepoURL)
	}
	o, r, _ := parseRepoURL(DefaultRepoURL)
	return o, r
}

func parseRepoURL(raw string) (owner, repo string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse repo url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo url %q must contain owner/repo path", raw)
	}
	return parts[0], parts[1], nil
}

type SystemInfo struct {
	Version     string `json:"version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	InContainer bool   `json:"in_container"`
}

type LatestInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	HasUpdate      bool   `json:"has_update"`
	PublishedAt    string `json:"published_at"`
	ReleaseNotes   string `json:"release_notes"`
	AssetReady     bool   `json:"asset_ready"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// CurrentInfo reports the running binary's version and platform.
func (s *UpgradeService) CurrentInfo() SystemInfo {
	return SystemInfo{
		Version:     s.version,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InContainer: inContainer(),
	}
}

// inContainer detects docker (/.dockerenv) and kubernetes
// (KUBERNETES_SERVICE_HOST) environments.
func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// CheckLatest queries GitHub for the latest release and compares it with the
// running version.
func (s *UpgradeService) CheckLatest(ctx context.Context) (*LatestInfo, error) {
	release, err := s.fetchRelease(ctx, "releases/latest")
	if err != nil {
		return nil, err
	}
	return &LatestInfo{
		CurrentVersion: s.version,
		LatestVersion:  release.TagName,
		HasUpdate:      hasNewerVersion(s.version, release.TagName),
		PublishedAt:    release.PublishedAt,
		ReleaseNotes:   release.Body,
		AssetReady:     findAsset(release.Assets, platformAssetName()) != nil,
	}, nil
}

// Upgrade downloads the target release, verifies it, swaps the binary in place
// and records an audit entry. On success the caller is expected to trigger
// Restart after the HTTP response is written.
func (s *UpgradeService) Upgrade(ctx context.Context, operatorID uint, targetVersion string) (from, to string, err error) {
	if !s.mu.TryLock() {
		return "", "", ErrUpgradeInProgress
	}
	// Keep the lock forever on success: the process restarts right after.
	// A panic must also release the lock — gin's Recovery keeps the process
	// alive, and a stuck lock would make every future upgrade return 409.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("upgrade panicked: %v", r)
		}
		if err != nil {
			s.mu.Unlock()
		}
	}()

	if inContainer() {
		return "", "", ErrContainerDeployment
	}

	if targetVersion == "" {
		latest, err := s.fetchRelease(ctx, "releases/latest")
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve latest release: %w", err)
		}
		targetVersion = latest.TagName
	}
	if targetVersion == s.version {
		return "", "", ErrAlreadyUpToDate
	}

	release, err := s.fetchRelease(ctx, "releases/tags/"+url.PathEscape(targetVersion))
	if err != nil {
		if errors.Is(err, errGitHubNotFound) {
			return "", "", ErrReleaseNotFound
		}
		return "", "", fmt.Errorf("failed to fetch release %s: %w", targetVersion, err)
	}

	assetName := platformAssetName()
	binaryAsset := findAsset(release.Assets, assetName)
	checksumsAsset := findAsset(release.Assets, checksumsAssetName)
	if binaryAsset == nil || checksumsAsset == nil {
		return "", "", ErrAssetNotFound
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", "", fmt.Errorf("failed to locate current executable: %w", err)
	}

	// Download the archive next to the executable so the final rename stays on
	// the same filesystem and is atomic.
	archivePath, archiveSum, err := s.downloadToDir(ctx, binaryAsset.BrowserDownloadURL, filepath.Dir(exePath))
	if err != nil {
		return "", "", fmt.Errorf("failed to download %s: %w", assetName, err)
	}
	defer os.Remove(archivePath)

	expectedSum, err := s.fetchChecksum(ctx, checksumsAsset.BrowserDownloadURL, assetName)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch checksums: %w", err)
	}
	if !strings.EqualFold(archiveSum, expectedSum) {
		return "", "", fmt.Errorf("checksum mismatch for %s: got %s, want %s", assetName, archiveSum, expectedSum)
	}

	newPath := exePath + ".new"
	binaryName := fmt.Sprintf("sub2balance-%s-%s", runtime.GOOS, runtime.GOARCH)
	if err := extractBinary(archivePath, binaryName, newPath); err != nil {
		os.Remove(newPath)
		return "", "", fmt.Errorf("failed to extract binary: %w", err)
	}

	if err := verifyNewBinary(ctx, newPath, targetVersion); err != nil {
		os.Remove(newPath)
		return "", "", fmt.Errorf("new binary self-check failed: %w", err)
	}

	// Back up via hard link, then swap with a single atomic rename. The exe
	// path is never absent from disk, so systemd's ExecStart target stays valid
	// no matter where this fails. The .bak link allows manual rollback.
	bakPath := exePath + ".bak"
	if err := os.Remove(bakPath); err != nil && !os.IsNotExist(err) {
		os.Remove(newPath)
		return "", "", fmt.Errorf("failed to clear previous backup: %w", err)
	}
	if err := os.Link(exePath, bakPath); err != nil {
		os.Remove(newPath)
		return "", "", fmt.Errorf("failed to back up current binary: %w", err)
	}
	if err := os.Rename(newPath, exePath); err != nil {
		os.Remove(newPath)
		return "", "", fmt.Errorf("failed to activate new binary: %w", err)
	}

	if auditErr := s.auditService.Log(ctx, operatorID, "system_upgrade", map[string]any{
		"from": s.version,
		"to":   targetVersion,
	}); auditErr != nil {
		log.Printf("upgrade: failed to write audit log: %v", auditErr)
	}

	log.Printf("upgrade: binary replaced %s -> %s, restarting shortly", s.version, targetVersion)
	return s.version, targetVersion, nil
}

// Restart re-execs the current process (now backed by the new binary) after a
// short delay so the HTTP response can reach the client first. If exec fails,
// exit non-zero and let systemd (Restart=on-failure) bring up the new binary.
func (s *UpgradeService) Restart() {
	go func() {
		time.Sleep(restartDelay)
		exePath, err := os.Executable()
		if err != nil {
			log.Printf("upgrade restart: failed to locate executable: %v", err)
			os.Exit(1)
		}
		if err := syscall.Exec(exePath, os.Args, os.Environ()); err != nil {
			log.Printf("upgrade restart: exec failed: %v", err)
			os.Exit(1)
		}
	}()
}

var errGitHubNotFound = errors.New("github resource not found")

// fetchRelease calls the GitHub releases API, e.g. path "releases/latest" or
// "releases/tags/v1.2.3".
func (s *UpgradeService) fetchRelease(ctx context.Context, apiPath string) (*githubRelease, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/%s", s.owner, s.repo, apiPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errGitHubNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github responded with status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode github response: %w", err)
	}
	return &release, nil
}

// downloadToDir streams the URL into a temp file inside dir and returns the
// file path plus its hex-encoded sha256.
func (s *UpgradeService) downloadToDir(ctx context.Context, downloadURL, dir string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to build download request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("download responded with status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(dir, ".sub2balance-upgrade-*.tar.gz")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp file: %w", err)
	}
	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", "", fmt.Errorf("failed to write download: %w", err)
	}
	if n > maxArchiveBytes {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", "", fmt.Errorf("download exceeds size limit of %d bytes", int64(maxArchiveBytes))
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", "", fmt.Errorf("failed to close temp file: %w", err)
	}
	return tmp.Name(), hex.EncodeToString(hasher.Sum(nil)), nil
}

// fetchChecksum downloads checksums.txt and returns the sha256 recorded for
// assetName. Lines look like "<hex>  <filename>".
func (s *UpgradeService) fetchChecksum(ctx context.Context, downloadURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build checksums request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("checksums request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums responded with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if name := strings.TrimPrefix(fields[1], "*"); name == assetName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum entry for %s", assetName)
}

// extractBinary pulls the exact member named memberName out of a tar.gz
// archive into destPath (0755). Matching by exact cleaned name guards against
// path traversal — no member path is ever used to build the destination.
func extractBinary(archivePath, memberName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read archive entry: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg || path.Clean(hdr.Name) != memberName {
			continue
		}

		out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", destPath, err)
		}
		n, err := io.Copy(out, io.LimitReader(tr, maxBinaryBytes+1))
		if err != nil {
			out.Close()
			return fmt.Errorf("failed to write %s: %w", destPath, err)
		}
		if n > maxBinaryBytes {
			out.Close()
			return fmt.Errorf("extracted binary exceeds size limit of %d bytes", int64(maxBinaryBytes))
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("failed to close %s: %w", destPath, err)
		}
		return nil
	}
	return fmt.Errorf("archive does not contain %s", memberName)
}

// verifyNewBinary runs "{binary} --version" and requires the output to match
// the target tag exactly, so a corrupt or wrong binary never replaces the
// running one.
func verifyNewBinary(ctx context.Context, binaryPath, expectedVersion string) error {
	checkCtx, cancel := context.WithTimeout(ctx, selfCheckTimeout)
	defer cancel()

	out, err := exec.CommandContext(checkCtx, binaryPath, "--version").Output()
	if err != nil {
		return fmt.Errorf("failed to run --version: %w", err)
	}
	if got := strings.TrimSpace(string(out)); got != expectedVersion {
		return fmt.Errorf("reported version %q does not match target %q", got, expectedVersion)
	}
	return nil
}

func platformAssetName() string {
	return fmt.Sprintf("sub2balance-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

func findAsset(assets []githubAsset, name string) *githubAsset {
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i]
		}
	}
	return nil
}

// hasNewerVersion compares versions semantically ("v" prefix stripped,
// major.minor.patch numeric compare). A "dev" build is always upgradable;
// unparsable versions fall back to string inequality.
func hasNewerVersion(current, latest string) bool {
	if current == "dev" {
		return true
	}
	cur, curOK := parseSemver(current)
	lat, latOK := parseSemver(latest)
	if !curOK || !latOK {
		return current != latest
	}
	for i := 0; i < 3; i++ {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

func parseSemver(v string) ([3]int, bool) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		nums[i] = n
	}
	return nums, true
}
