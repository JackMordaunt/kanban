// release is a tool for distributing OS specific packages to Windows, macOS
// and Linux.
package main

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jackmordaunt/icns"
)

// TODO:
// - Release for windows/linux
// - configure things?
// - zip results
// - push to github releases
// - tag a release -> upload to github
func main() {
	if err := func() error {
		binary, err := build("cmd/kanban")
		if err != nil {
			return fmt.Errorf("building: %w", err)
		}
		if err := bundle("dist/Kanban.app", binary, "res/icon.png", "res/darwin/Info.plist"); err != nil {
			return fmt.Errorf("bundling: %w", err)
		}
		return nil
	}(); err != nil {
		fmt.Printf("error: %v", err)
	}
}

// bundle creates a macOS .app bundle on disk rooted at dest.
// All paramaters are filepaths.
// NB: Will clobber destination if it is a directory, or error if it is a file.
func bundle(dest, binary, icon, plist string) error {
	var (
		contents  = filepath.Join(dest, "Contents")
		macos     = filepath.Join(contents, "MacOS")
		resources = filepath.Join(contents, "Resources")
	)
	m, err := os.Stat(dest)
	if os.IsNotExist(err) || m.IsDir() {
		os.RemoveAll(dest)
		if err := os.MkdirAll(dest, 0777); err != nil {
			return fmt.Errorf("preparing destination: %w", err)
		}
	} else if !m.IsDir() {
		return fmt.Errorf("destination %q: not a directory", dest)
	}
	os.MkdirAll(macos, 0777)
	os.MkdirAll(resources, 0777)
	if err := cp(binary, filepath.Join(macos, "kanban")); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}
	if err := cp(plist, filepath.Join(contents, "Info.plist")); err != nil {
		return fmt.Errorf("copying plist: %w", err)
	}
	if err := convertIcon(icon, filepath.Join(resources, "kanban.icns")); err != nil {
		return fmt.Errorf("converting icon to icns: %w", err)
	}
	return nil
}

// convertIcon converts the source png to icon and returns a path to it.
func convertIcon(src, dst string) error {
	srcf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcf.Close()
	img, err := png.Decode(srcf)
	if err != nil {
		return fmt.Errorf("decoding source png: %w", err)
	}
	dstf, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("opening destination file: %w", err)
	}
	defer dstf.Close()
	if err := icns.Encode(dstf, img); err != nil {
		return fmt.Errorf("encoding icns: %w", err)
	}
	return nil
}

// build the Go program rooted at path and returns a path to it.
func build(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}
	if err := run("go", "build", "-o", "dist/kanban", path); err != nil {
		return "", err
	}
	return "dist/kanban", nil
}

// run the specified command and return any error.
func run(cmd string, args ...string) error {
	if out, err := exec.Command(cmd, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("running command %q: %v: %w", cmd, string(out), err)
	}
	return nil
}

// cp copies src file to destination.
// If destination is a directory, the file will be copied into it.
// If destination doesn't exist it will be created as a file.
// If destination is a file an error will be returned.
func cp(src, dst string) error {
	if src == "" || dst == "" {
		return nil
	}
	var err error
	src, err = filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}
	srcf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %q: %w", src, err)
	}
	defer srcf.Close()
	_, err = os.Stat(filepath.Dir(dst))
	if os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(dst), 0777)
	}
	dstf, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("creating %q: %w", dst, err)
	}
	defer dstf.Close()
	if _, err := io.Copy(dstf, srcf); err != nil {
		return fmt.Errorf("copying data: %w", err)
	}
	return nil
}
