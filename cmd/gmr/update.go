package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/slucheninov/gmr/internal/ui"
	"github.com/slucheninov/gmr/internal/update"
	"github.com/slucheninov/gmr/internal/version"
)

func runUpdate() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	ui.Log("Checking for gmr updates (current: %s)...", version.Version)
	result, err := update.Run(ctx, version.Version, executable)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("%w; updating %s requires write access to its installation directory", err, executable)
		}
		return err
	}
	if !result.Updated {
		ui.OK("gmr %s is already up to date (latest release: %s)", version.Version, result.Version)
		return nil
	}
	ui.OK("Updated gmr %s → %s", version.Version, result.Version)
	return nil
}
