package main

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed zig-common.zip
var zigCommonZip []byte

//go:embed zig-linux-amd64.zip
var zigLinuxAmd64Zip []byte

func myUserCacheDir() (string, error) {
	const appname = "go-zig"
	const appauthor = "jcbhmr"
	const version = "0.13.0"

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		return filepath.Join(userCacheDir, appauthor, appname, "Cache", version), nil
	} else if runtime.GOOS == "darwin" {
		return filepath.Join(userCacheDir, appname, version), nil
	} else {
		return filepath.Join(userCacheDir, appname, version), nil
	}
}

func main() {
	log.SetFlags(0)

	cacheDir, err := myUserCacheDir()
	if err != nil {
		log.Fatalf("failed to get cache directory: %v", err)
	}

	cacheDirExists := true
	_, err = os.Stat(cacheDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			cacheDirExists = false
		} else {
			log.Fatalf("failed to stat cache directory %v: %v", cacheDir, err)
		}
	}

	if !cacheDirExists {
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			log.Fatalf("failed to create cache directory %v: %v", cacheDir, err)
		}

		err = func() error {
			reader := bytes.NewReader(zigCommonZip)
			zipReader, err := zip.NewReader(reader, int64(reader.Len()))
			if err != nil {
				return fmt.Errorf("failed to create zip reader for zig-common.zip: %w", err)
			}
			err = os.CopyFS(cacheDir, zipReader)
			if err != nil {
				return fmt.Errorf("failed to copy zig-common.zip to cache directory %v: %w", cacheDir, err)
			}

			reader = bytes.NewReader(zigLinuxAmd64Zip)
			zipReader, err = zip.NewReader(reader, int64(reader.Len()))
			if err != nil {
				return fmt.Errorf("failed to create zip reader for zig-linux-amd64.zip: %w", err)
			}
			err = os.CopyFS(cacheDir, zipReader)
			if err != nil {
				return fmt.Errorf("failed to copy zig-linux-amd64.zip to cache directory %v: %w", cacheDir, err)
			}

			return nil
		}()
		if err != nil {
			err2 := os.RemoveAll(cacheDir)
			if err2 != nil {
				log.Fatalf("failed to remove cache directory %v after fatal error %v: %v", cacheDir, err, err2)
			}
			log.Fatal(err)
		}
	}

	var zigExe string
	if runtime.GOOS == "windows" {
		zigExe = "zig.exe"
	} else {
		zigExe = "zig"
	}
	cmd := exec.Command(filepath.Join(cacheDir, zigExe))
	cmd.Args = os.Args
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	exitCode := 0
	err = cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			log.Fatalf("failed to run %v: %v", cmd, err)
		}
	}
	os.Exit(exitCode)
}
