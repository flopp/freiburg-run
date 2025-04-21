package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExternalResourcePaths(t *testing.T) {
	// Create temp dir
	out := t.TempDir()

	// Create resource manager
	rm := NewResourceManager(out)

	// cd into base dir of the repo
	// This is needed to get the correct relative path (e.g. "static/parkrun-track.js")
	// and to get the correct path for the external assets
	// Get the base directory of the repo dynamically
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	// Get the base directory of the repo (two up from currentDir)
	baseDir := filepath.Join(currentDir, "..", "..")

	// Change directory to the base directory of the repo
	if err := os.Chdir(baseDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Copy external assets (this might panic!)
	rm.CopyExternalAssets()
}
