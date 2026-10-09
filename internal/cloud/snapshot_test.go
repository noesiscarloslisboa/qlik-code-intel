package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteSnapshotPreservesExactSourceAndManifest(t *testing.T) {
	for _, source := range []string{"", "\ufeff///$tab Main\r\nSET vRoot=lib://Data;\r\nSales: LOAD Amount AS Revenue FROM [$(vRoot)/sales.qvd];"} {
		dest := filepath.Join(physicalTempDir(t), "app snapshot")
		snapshot := originalSnapshot(source)
		result, err := WriteSnapshot(context.Background(), dest, snapshot)
		if err != nil {
			t.Fatal(err)
		}
		files, _ := os.ReadDir(dest)
		if len(files) != 2 || result.SnapshotPath != dest || result.ManifestPath != filepath.Join(dest, "manifest.json") || result.AppID != "app-1" || result.ScriptID != "saved-1" {
			t.Fatalf("incomplete result: %+v files=%v", result, files)
		}
		actual, err := os.ReadFile(filepath.Join(dest, "script.qvs"))
		if err != nil || string(actual) != source {
			t.Fatalf("source changed: %v", err)
		}
		data, _ := os.ReadFile(result.ManifestPath)
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil || manifest != snapshot.Manifest {
			t.Fatalf("manifest changed: %v %+v", err, manifest)
		}
		if strings.Contains(string(data), "private-token") {
			t.Fatal("credential in manifest")
		}
		if runtime.GOOS != "windows" {
			for _, name := range []string{dest, filepath.Join(dest, "script.qvs"), result.ManifestPath} {
				info, _ := os.Stat(name)
				if info.Mode().Perm()&0077 != 0 {
					t.Fatalf("permissions too broad: %s %v", name, info.Mode())
				}
			}
		}
	}
}

func TestSnapshotDoesNotOverwriteExistingDestinations(t *testing.T) {
	root := physicalTempDir(t)
	existing := filepath.Join(root, "existing")
	if err := os.Mkdir(existing, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(existing, "script.qvs")
	os.WriteFile(sentinel, []byte("unrelated original"), 0600)
	for _, path := range []string{existing, sentinel, filepath.Join(root, "missing", "snapshot"), ""} {
		if _, err := WriteSnapshot(context.Background(), path, originalSnapshot("new source")); err == nil {
			t.Fatalf("accepted bad destination %q", path)
		}
	}
	if data, _ := os.ReadFile(sentinel); string(data) != "unrelated original" {
		t.Fatal("overwrote existing source")
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(existing, link); err != nil {
		t.Logf("symlink unavailable: %v", err)
		return
	}
	for _, path := range []string{link, filepath.Join(link, "new")} {
		if _, err := WriteSnapshot(context.Background(), path, originalSnapshot("source")); err == nil {
			t.Fatalf("followed symlink %q", path)
		}
	}
}

func TestSnapshotRejectsTamperedInputsBeforeCreatingDirectory(t *testing.T) {
	for _, mutate := range []func(*Snapshot){
		func(s *Snapshot) { s.Source = []byte("changed") },
		func(s *Snapshot) { s.Manifest.SourceFile = "../outside.qvs" },
		func(s *Snapshot) { s.Manifest.ScriptID = "" },
		func(s *Snapshot) { s.Manifest.SchemaVersion = 2 },
		func(s *Snapshot) { s.Manifest.FetchedAt = time.Time{} },
		func(s *Snapshot) { s.Manifest.SourceBytes++ },
		func(s *Snapshot) { s.Manifest.Tenant = "http://example.com" },
	} {
		dest := filepath.Join(physicalTempDir(t), "snapshot")
		snapshot := originalSnapshot("source")
		mutate(&snapshot)
		if _, err := WriteSnapshot(context.Background(), dest, snapshot); err == nil {
			t.Fatal("accepted invalid snapshot")
		}
		if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("created destination for invalid input: %v", err)
		}
	}
}

func TestSnapshotWriteFailureAndCancellationCleanOwnedFiles(t *testing.T) {
	for _, cancelAfterFirst := range []bool{false, true} {
		dest := filepath.Join(physicalTempDir(t), "snapshot")
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		write := func(w io.Writer, data []byte) error {
			calls++
			if cancelAfterFirst {
				_, err := w.Write(data)
				cancel()
				return err
			}
			if calls == 2 {
				return io.ErrClosedPipe
			}
			_, err := w.Write(data)
			return err
		}
		_, err := writeSnapshot(ctx, dest, originalSnapshot("source"), write)
		cancel()
		if err == nil || cancelAfterFirst && !errors.Is(err, context.Canceled) {
			t.Fatalf("missing write/cancellation error: %v", err)
		}
		if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("left partial snapshot: %v", err)
		}
	}
}

func TestSnapshotCleanupDoesNotRemoveUnownedFiles(t *testing.T) {
	dest := filepath.Join(physicalTempDir(t), "snapshot")
	write := func(w io.Writer, data []byte) error {
		if err := os.WriteFile(filepath.Join(dest, "manifest.json"), []byte("unrelated concurrent file"), 0600); err != nil {
			return err
		}
		_, err := w.Write(data)
		return err
	}
	if _, err := writeSnapshot(context.Background(), dest, originalSnapshot("source"), write); err == nil {
		t.Fatal("overwrote concurrent file")
	}
	data, _ := os.ReadFile(filepath.Join(dest, "manifest.json"))
	if string(data) != "unrelated concurrent file" {
		t.Fatal("cleanup removed unowned file")
	}
	if _, err := os.Lstat(filepath.Join(dest, "script.qvs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("left owned source after failure")
	}
}

func physicalTempDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func originalSnapshot(source string) Snapshot {
	digest := sha256.Sum256([]byte(source))
	return Snapshot{Source: []byte(source), Manifest: Manifest{
		SchemaVersion: 1, Tenant: "https://example.eu.qlikcloud.com", AppID: "app-1", ScriptID: "saved-1",
		FetchedAt: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC), SourceFile: "script.qvs",
		SourceSHA256: hex.EncodeToString(digest[:]), SourceBytes: len([]byte(source)),
	}}
}
