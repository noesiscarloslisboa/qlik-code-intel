package qlik

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var skippedDirectories = map[string]bool{
	".git": true, ".hg": true, ".svn": true, "node_modules": true,
	"vendor": true, "bin": true, "dist": true, "build": true, ".cache": true,
}

// Scan indexes QVS files in lexical path order. Symlinks are never followed.
// Unsupported syntax is a diagnostic; filesystem errors are returned to callers.
func Scan(ctx context.Context, root string) (*Index, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("scan root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root must be a directory: %s", root)
	}
	idx := &Index{Root: abs, Files: []File{}}
	err = filepath.WalkDir(abs, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if name != abs && skippedDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(name), ".qvs") {
			return nil
		}
		rel, err := filepath.Rel(abs, name)
		if err != nil {
			return err
		}
		source, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		file, err := Parse(ctx, filepath.ToSlash(rel), source)
		if err != nil {
			return err
		}
		idx.Files = append(idx.Files, file)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}
	return idx, nil
}
