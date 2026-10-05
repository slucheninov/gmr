package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
)

// Only the expected root-level regular file is read. Archive paths are never
// used as filesystem destinations, so other entries cannot escape the archive.
func extractBinary(archive []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, file := range zr.File {
			if file.Name != "gmr.exe" && file.Name != "./gmr.exe" {
				continue
			}
			if !file.Mode().IsRegular() {
				return nil, errors.New("gmr.exe is not a regular file")
			}
			r, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			return readBinary(r)
		}
		return nil, errors.New("gmr.exe not found in archive")
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer func() { _ = gz.Close() }()
	// Also bound decompression while tar skips unrelated entries.
	tr := tar.NewReader(io.LimitReader(gz, maxBinarySize+maxMetadataSize))
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("gmr not found in archive")
		}
		if err != nil {
			return nil, err
		}
		if header.Name != "gmr" && header.Name != "./gmr" {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return nil, errors.New("gmr is not a regular file")
		}
		return readBinary(tr)
	}
}

func readBinary(r io.Reader) ([]byte, error) {
	data, err := readLimited(r, maxBinarySize)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("empty executable in archive")
	}
	return data, nil
}
