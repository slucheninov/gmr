package update

import (
	"errors"
	"fmt"
	"os"
)

func replaceExecutable(source, target string) error {
	// Windows allows renaming a running executable but not overwriting it.
	// Keep it as .old until the next update, when the old process has exited.
	backup := target + ".old"
	if err := os.Remove(backup); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(target, backup); err != nil {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return fmt.Errorf("replace executable: %w; restore failed: %v; previous binary is at %s", err, rollbackErr, backup)
		}
		return err
	}
	return nil
}
