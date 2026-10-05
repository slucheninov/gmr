package update

import (
	"fmt"
	"os"
	"path/filepath"
)

func installBinary(executable string, data []byte) error {
	target, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", target)
	}
	// Use the same directory to keep replacement on the same filesystem.
	file, err := os.CreateTemp(filepath.Dir(target), ".gmr-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replaceExecutable(file.Name(), target)
}
