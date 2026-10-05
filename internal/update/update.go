// Package update installs verified gmr binaries from GitHub Releases.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/slucheninov/gmr/internal/release"
)

const (
	latestReleaseURL = "https://api.github.com/repos/slucheninov/gmr/releases/latest"
	maxMetadataSize  = 1 << 20
	maxArchiveSize   = 64 << 20
	maxBinarySize    = 128 << 20
)

// Result describes the latest release and whether it was installed.
type Result struct {
	Version string
	Updated bool
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type releaseInfo struct {
	Tag        string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []releaseAsset `json:"assets"`
}

type updater struct {
	client     *http.Client
	releaseURL string
	goos       string
	goarch     string
}

// Run updates executable to the latest stable release. Known versions are
// never downgraded; development builds are replaced by the latest release.
// Symlinks are followed so that the installed executable, not its link, changes.
func Run(ctx context.Context, current, executable string) (Result, error) {
	u := updater{
		client:     &http.Client{Timeout: 2 * time.Minute},
		releaseURL: latestReleaseURL,
		goos:       runtime.GOOS,
		goarch:     runtime.GOARCH,
	}
	return u.run(ctx, current, executable)
}

func (u updater) run(ctx context.Context, current, executable string) (Result, error) {
	if (u.goos != "linux" && u.goos != "darwin" && u.goos != "windows") ||
		(u.goarch != "amd64" && u.goarch != "arm64") {
		return Result{}, fmt.Errorf("automatic updates are not supported on %s/%s", u.goos, u.goarch)
	}
	data, err := u.download(ctx, u.releaseURL, maxMetadataSize)
	if err != nil {
		return Result{}, fmt.Errorf("check latest release: %w", err)
	}
	var latest releaseInfo
	if err := json.Unmarshal(data, &latest); err != nil {
		return Result{}, fmt.Errorf("read latest release: %w", err)
	}
	prefix, _, valid := release.ParseTag(latest.Tag)
	if !valid || prefix != "v" || latest.Draft || latest.Prerelease {
		return Result{}, fmt.Errorf("invalid stable release tag %q", latest.Tag)
	}
	result := Result{Version: latest.Tag}
	// Latest chooses the first tag on a version tie, accepting both vX.Y.Z
	// (release binaries) and X.Y.Z (go install/source builds).
	if newest, _, _, ok := release.Latest([]string{current, latest.Tag}); ok && newest == current {
		return result, nil
	}

	ext := ".tar.gz"
	if u.goos == "windows" {
		ext = ".zip"
	}
	archiveName := fmt.Sprintf("gmr-%s-%s-%s%s", latest.Tag, u.goos, u.goarch, ext)
	archiveURL, err := latest.assetURL(archiveName)
	if err != nil {
		return Result{}, err
	}
	checksumURL, err := latest.assetURL("checksums.txt")
	if err != nil {
		return Result{}, err
	}
	checksums, err := u.download(ctx, checksumURL, maxMetadataSize)
	if err != nil {
		return Result{}, fmt.Errorf("download checksums: %w", err)
	}
	want, err := archiveChecksum(string(checksums), archiveName)
	if err != nil {
		return Result{}, err
	}
	archive, err := u.download(ctx, archiveURL, maxArchiveSize)
	if err != nil {
		return Result{}, fmt.Errorf("download %s: %w", archiveName, err)
	}
	if got := sha256.Sum256(archive); got != want {
		return Result{}, fmt.Errorf("SHA-256 mismatch for %s; update aborted", archiveName)
	}
	binary, err := extractBinary(archive, u.goos)
	if err != nil {
		return Result{}, fmt.Errorf("extract %s: %w", archiveName, err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := installBinary(executable, binary); err != nil {
		return Result{}, fmt.Errorf("install update: %w", err)
	}
	result.Updated = true
	return result, nil
}

func (r releaseInfo) assetURL(name string) (string, error) {
	for _, asset := range r.Assets {
		if asset.Name == name && asset.URL != "" {
			return asset.URL, nil
		}
	}
	return "", fmt.Errorf("release %s is missing %s", r.Tag, name)
}

func (u updater) download(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gmr-updater")
	if url == u.releaseURL {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return readLimited(resp.Body, limit)
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download or archive entry exceeds %d bytes", limit)
	}
	return data, nil
}

func archiveChecksum(checksums, name string) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	found := false
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != name {
			continue
		}
		if found {
			return digest, fmt.Errorf("duplicate checksum for %s", name)
		}
		decoded, err := hex.DecodeString(fields[0])
		if err != nil || len(decoded) != sha256.Size {
			return digest, fmt.Errorf("invalid SHA-256 checksum for %s", name)
		}
		copy(digest[:], decoded)
		found = true
	}
	if !found {
		return digest, fmt.Errorf("no SHA-256 checksum for %s", name)
	}
	return digest, nil
}
