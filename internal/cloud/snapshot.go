package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"
)

// Manifest associates exact local source bytes with a saved version in one app.
// API timestamps describe separate observations; they do not prove reload validity.
type Manifest struct {
	SchemaVersion        int       `json:"schema_version"`
	Tenant               string    `json:"tenant"`
	AppID                string    `json:"app_id"`
	AppName              string    `json:"app_name,omitempty"`
	ScriptID             string    `json:"script_id"`
	ScriptModifiedTime   string    `json:"script_modified_time,omitempty"`
	ScriptVersionMessage string    `json:"script_version_message,omitempty"`
	FetchedAt            time.Time `json:"fetched_at"`
	LastReloadTime       string    `json:"last_reload_time,omitempty"`
	SourceFile           string    `json:"source_file"`
	SourceSHA256         string    `json:"source_sha256"`
	SourceBytes          int       `json:"source_bytes"`
}

// Snapshot contains unevaluated source, not a persisted symbol index.
type Snapshot struct {
	Manifest Manifest
	Source   []byte
}

// Result is the CLI summary; source text and authentication are excluded.
type Result struct {
	SnapshotPath string `json:"snapshot_path"`
	ManifestPath string `json:"manifest_path"`
	AppID        string `json:"app_id"`
	ScriptID     string `json:"script_id"`
}

// ValidateDestination requires a new directory under existing, non-symlink parents.
// Checking again at creation time prevents an existing snapshot from being replaced.
func ValidateDestination(destination string) error {
	if destination == "" {
		return errors.New("snapshot destination is required")
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("snapshot destination: %w", err)
	}
	if _, err := os.Lstat(abs); err == nil {
		return errors.New("snapshot destination already exists; choose a new directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("snapshot destination: %w", err)
	}
	for parent := filepath.Dir(abs); ; parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err != nil {
			return fmt.Errorf("snapshot parent must already exist: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("snapshot parents must be directories without symlinks; use a physical directory path")
		}
		if filepath.Dir(parent) == parent {
			break
		}
	}
	return nil
}

// WriteSnapshot writes source and metadata only to a new destination. On failure
// it removes files created by this call, never recursively removing other content.
func WriteSnapshot(ctx context.Context, destination string, snapshot Snapshot) (Result, error) {
	return writeSnapshot(ctx, destination, snapshot, func(w io.Writer, data []byte) error {
		n, err := w.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
		return err
	})
}

func writeSnapshot(ctx context.Context, destination string, snapshot Snapshot, write func(io.Writer, []byte) error) (result Result, err error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	m := snapshot.Manifest
	digest := sha256.Sum256(snapshot.Source)
	if m.SchemaVersion != 1 || m.SourceFile != "script.qvs" || m.FetchedAt.IsZero() || ValidateTenant(m.Tenant) != nil || ValidateAppID(m.AppID) != nil || ValidateAppID(m.ScriptID) != nil || m.ScriptID == "current" || !utf8.Valid(snapshot.Source) || m.SourceBytes != len(snapshot.Source) || m.SourceSHA256 != hex.EncodeToString(digest[:]) {
		return Result{}, errors.New("snapshot source or manifest identity is invalid")
	}
	metadata, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Result{}, errors.New("could not encode snapshot manifest")
	}
	metadata = append(metadata, '\n')
	if err := ValidateDestination(destination); err != nil {
		return Result{}, err
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return Result{}, err
	}
	if err := os.Mkdir(abs, 0700); err != nil {
		return Result{}, fmt.Errorf("create snapshot: %w", err)
	}
	type ownedFile struct {
		path string
		info os.FileInfo
	}
	var owned []ownedFile
	directory, statErr := os.Lstat(abs)
	defer func() {
		if err == nil {
			return
		}
		current, checkErr := os.Lstat(abs)
		if statErr != nil || checkErr != nil || !os.SameFile(directory, current) {
			err = errors.Join(err, errors.New("snapshot directory changed; cannot safely clean up"))
			return
		}
		for _, file := range owned {
			current, checkErr := os.Lstat(file.path)
			if checkErr == nil && os.SameFile(file.info, current) {
				err = errors.Join(err, os.Remove(file.path))
			} else if !errors.Is(checkErr, os.ErrNotExist) {
				err = errors.Join(err, errors.New("snapshot file changed; cannot safely clean up"))
			}
		}
		err = errors.Join(err, os.Remove(abs)) // Fails rather than deleting unrelated files.
	}()
	if statErr != nil {
		return Result{}, fmt.Errorf("inspect new snapshot: %w", statErr)
	}
	for _, file := range []struct {
		name string
		data []byte
	}{{"script.qvs", snapshot.Source}, {"manifest.json", metadata}} {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		path := filepath.Join(abs, file.name)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return Result{}, fmt.Errorf("create snapshot file: %w", err)
		}
		info, statErr := f.Stat()
		if statErr != nil {
			return Result{}, errors.Join(statErr, f.Close())
		}
		owned = append(owned, ownedFile{path, info})
		writeErr := write(f, file.data)
		closeErr := f.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return Result{}, fmt.Errorf("write snapshot file: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return Result{SnapshotPath: abs, ManifestPath: filepath.Join(abs, "manifest.json"), AppID: m.AppID, ScriptID: m.ScriptID}, nil
}
