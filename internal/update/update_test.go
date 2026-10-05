package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func makeArchive(t *testing.T, goos, name, contents string, symlink bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	if goos == "windows" {
		zw := zip.NewWriter(&buf)
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0755)
		if symlink {
			header.SetMode(os.ModeSymlink | 0755)
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		header := &tar.Header{Name: name, Size: int64(len(contents)), Mode: 0755, Typeflag: tar.TypeReg}
		if symlink {
			header.Typeflag, header.Linkname, header.Size = tar.TypeSymlink, "elsewhere", 0
			contents = ""
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

type updateFixture struct {
	u         updater
	release   releaseInfo
	archive   []byte
	checksums string
	status    map[string]int
	downloads atomic.Int32
}

func newUpdateFixture(t *testing.T, goos, goarch string) *updateFixture {
	t.Helper()
	name, ext := "./gmr", ".tar.gz"
	if goos == "windows" {
		name, ext = "gmr.exe", ".zip"
	}
	f := &updateFixture{
		archive: makeArchive(t, goos, name, "new executable", false),
		status:  make(map[string]int),
	}
	archiveName := "gmr-v1.2.3-" + goos + "-" + goarch + ext
	f.checksums = fmt.Sprintf("%x  %s\n", sha256.Sum256(f.archive), archiveName)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status := f.status[r.URL.Path]; status != 0 {
			w.WriteHeader(status)
			return
		}
		switch r.URL.Path {
		case "/latest":
			if r.Header.Get("Accept") != "application/vnd.github+json" || r.Header.Get("User-Agent") != "gmr-updater" {
				t.Error("missing API headers")
			}
			if err := json.NewEncoder(w).Encode(f.release); err != nil {
				t.Error(err)
			}
		case "/checksums":
			f.downloads.Add(1)
			fmt.Fprint(w, f.checksums)
		case "/archive":
			f.downloads.Add(1)
			if _, err := w.Write(f.archive); err != nil {
				t.Error(err)
			}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	f.release = releaseInfo{Tag: "v1.2.3", Assets: []releaseAsset{
		{Name: archiveName, URL: server.URL + "/archive"},
		{Name: "checksums.txt", URL: server.URL + "/checksums"},
	}}
	f.u = updater{client: server.Client(), releaseURL: server.URL + "/latest", goos: goos, goarch: goarch}
	return f
}

func installedBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gmr")
	if err := os.WriteFile(path, []byte("old executable"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertBinary(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("executable = %q, want %q", data, want)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0755 {
			t.Errorf("permissions = %v, want 0755", info.Mode().Perm())
		}
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".gmr-update-*"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v, error: %v", leftovers, err)
	}
}

func TestUpdatePlatforms(t *testing.T) {
	t.Parallel()
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			t.Run(goos+"/"+goarch, func(t *testing.T) {
				t.Parallel()
				f := newUpdateFixture(t, goos, goarch)
				path := installedBinary(t)
				result, err := f.u.run(t.Context(), "0.13.0", path)
				if err != nil {
					t.Fatal(err)
				}
				if result != (Result{Version: "v1.2.3", Updated: true}) {
					t.Fatalf("result = %+v", result)
				}
				assertBinary(t, path, "new executable")
			})
		}
	}
}

func TestUpdateVersions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		current string
		update  bool
	}{
		{"v1.2.3", false}, {"1.2.3", false}, {"1.10.0", false},
		{"2.0.0", false}, {"1.2.2", true}, {"dev-abcdef", true},
	} {
		t.Run(tc.current, func(t *testing.T) {
			t.Parallel()
			f := newUpdateFixture(t, "linux", "amd64")
			path := installedBinary(t)
			result, err := f.u.run(t.Context(), tc.current, path)
			if err != nil || result.Updated != tc.update {
				t.Fatalf("result = %+v, error = %v", result, err)
			}
			if !tc.update {
				assertBinary(t, path, "old executable")
				if f.downloads.Load() != 0 {
					t.Fatal("up-to-date version downloaded release assets")
				}
			}
		})
	}
}

func TestUpdateFailuresPreserveExecutable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*updateFixture)
		want   string
	}{
		{"API failure", func(f *updateFixture) { f.status["/latest"] = 403 }, "HTTP 403"},
		{"checksum download fails", func(f *updateFixture) { f.status["/checksums"] = 500 }, "HTTP 500"},
		{"archive download fails", func(f *updateFixture) { f.status["/archive"] = 404 }, "HTTP 404"},
		{"checksum mismatch", func(f *updateFixture) { f.archive = []byte("tampered") }, "SHA-256 mismatch"},
		{"checksum missing", func(f *updateFixture) { f.checksums = "" }, "no SHA-256 checksum"},
		{"checksum invalid", func(f *updateFixture) { f.checksums = "invalid  " + f.release.Assets[0].Name }, "invalid SHA-256"},
		{"checksum duplicate", func(f *updateFixture) { f.checksums += f.checksums }, "duplicate checksum"},
		{"asset missing", func(f *updateFixture) { f.release.Assets = f.release.Assets[1:] }, "missing gmr-"},
		{"checksums asset missing", func(f *updateFixture) { f.release.Assets = f.release.Assets[:1] }, "missing checksums.txt"},
		{"invalid tag", func(f *updateFixture) { f.release.Tag = "../../other" }, "invalid stable release"},
		{"prerelease", func(f *updateFixture) { f.release.Prerelease = true }, "invalid stable release"},
		{"draft", func(f *updateFixture) { f.release.Draft = true }, "invalid stable release"},
		{"unsupported OS", func(f *updateFixture) { f.u.goos = "plan9" }, "not supported"},
		{"unsupported arch", func(f *updateFixture) { f.u.goarch = "386" }, "not supported"},
		{"invalid archive", func(f *updateFixture) {
			f.archive = []byte("not a gzip archive")
			f.checksums = fmt.Sprintf("%x  %s", sha256.Sum256(f.archive), f.release.Assets[0].Name)
		}, "extract"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newUpdateFixture(t, "linux", "amd64")
			tc.change(f)
			path := installedBinary(t)
			result, err := f.u.run(t.Context(), "1.0.0", path)
			if err == nil || !strings.Contains(err.Error(), tc.want) || result.Updated {
				t.Fatalf("result = %+v, error = %v, want %q", result, err, tc.want)
			}
			assertBinary(t, path, "old executable")
		})
	}
}

func TestUpdateCanceled(t *testing.T) {
	t.Parallel()
	f := newUpdateFixture(t, "linux", "amd64")
	path := installedBinary(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := f.u.run(ctx, "1.0.0", path); err == nil {
		t.Fatal("expected cancellation error")
	}
	assertBinary(t, path, "old executable")
}

func TestExtractBinary(t *testing.T) {
	t.Parallel()
	for _, goos := range []string{"linux", "windows"} {
		name := "gmr"
		if goos == "windows" {
			name += ".exe"
		}
		for _, tc := range []struct {
			name     string
			entry    string
			contents string
			symlink  bool
			wantErr  bool
		}{
			{"root", name, "binary", false, false},
			{"dot root", "./" + name, "binary", false, false},
			{"traversal", "../" + name, "binary", false, true},
			{"absolute", "/" + name, "binary", false, true},
			{"wrong binary", "other", "binary", false, true},
			{"empty binary", name, "", false, true},
			{"symlink", name, "elsewhere", true, true},
		} {
			t.Run(goos+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				archive := makeArchive(t, goos, tc.entry, tc.contents, tc.symlink)
				data, err := extractBinary(archive, goos)
				if (err != nil) != tc.wantErr {
					t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
				}
				if !tc.wantErr && string(data) != tc.contents {
					t.Fatalf("binary = %q, want %q", data, tc.contents)
				}
			})
		}
	}
}

func TestReadLimited(t *testing.T) {
	t.Parallel()
	if _, err := readLimited(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected size limit error")
	}
	if data, err := readLimited(strings.NewReader("1234"), 4); err != nil || string(data) != "1234" {
		t.Fatalf("exact limit: data = %q, error = %v", data, err)
	}
}

func TestInstallThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks may require additional privileges on Windows")
	}
	t.Parallel()
	path := installedBinary(t)
	link := filepath.Join(t.TempDir(), "gmr")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := installBinary(link, []byte("new executable")); err != nil {
		t.Fatal(err)
	}
	assertBinary(t, path, "new executable")
	if target, err := os.Readlink(link); err != nil || target != path {
		t.Fatalf("symlink replaced: target = %q, error = %v", target, err)
	}
}

func TestInstallRejectsDirectory(t *testing.T) {
	t.Parallel()
	path := t.TempDir()
	if err := installBinary(path, []byte("binary")); err == nil {
		t.Fatal("expected error for directory target")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatal("target directory changed")
	}
}

func TestReplaceFailurePreservesExecutable(t *testing.T) {
	t.Parallel()
	target := installedBinary(t)
	source := filepath.Join(t.TempDir(), "missing")
	if err := replaceExecutable(source, target); err == nil {
		t.Fatal("expected replacement failure")
	}
	assertBinary(t, target, "old executable")
}
