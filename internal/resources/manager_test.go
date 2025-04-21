package resources

import (
	"testing"
)

func TestCopyExternalAssets(t *testing.T) {
	// Create temp dir
	out := t.TempDir()

	// Create resource manager
	rm := NewResourceManager("../../", out)

	// Copy external assets
	rm.CopyExternalAssets()

	if rm.Error != nil {
		t.Fatalf("failed to copy external assets: %v", rm.Error)
	}
}

func TestCopyStaticAssets(t *testing.T) {
	// Create temp dir
	out := t.TempDir()

	// Create resource manager
	rm := NewResourceManager("../..", out)

	// Copy static assets
	rm.CopyStaticAssets()

	if rm.Error != nil {
		t.Fatalf("failed to copy static assets: %v", rm.Error)
	}
}
