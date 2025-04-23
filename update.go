package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
)

func update(ctx context.Context, version string) error {
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("denizsincar29/go_yandex_music"))
	if err != nil {
		return fmt.Errorf("error occurred while detecting version: %w", err)
	}
	if !found {
		return fmt.Errorf("latest version for %s/%s could not be found from github repository", runtime.GOOS, runtime.GOARCH)
	}

	if latest.LessOrEqual(version) {
		fmt.Printf("Current version (%s) is the latest\n", version)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(ctx, latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	fmt.Printf("Successfully updated to version %s\n", latest.Version())
	return nil
}
