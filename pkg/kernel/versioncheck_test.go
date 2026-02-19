//go:build integration

// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"testing"
)

func TestCheckVersion_LatestStable(t *testing.T) {
	result, err := CheckVersion("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version == "" {
		t.Fatal("expected resolved version, got empty string")
	}
	if !result.Available {
		t.Errorf("expected latest stable to be available")
	}
	// Latest stable should almost always have checksums ready
	if !result.Buildable {
		t.Logf("latest stable %s not yet buildable (checksums pending): %s", result.Version, result.Message)
	}
}

func TestCheckVersion_NonExistent(t *testing.T) {
	result, err := CheckVersion("99.99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Available {
		t.Error("expected bogus version to be unavailable")
	}
	if result.Buildable {
		t.Error("expected bogus version to not be buildable")
	}
	if result.Message == "" {
		t.Error("expected descriptive message for unavailable version")
	}
}

func TestCheckVersion_LatestKeyword(t *testing.T) {
	result, err := CheckVersion("latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version == "" || result.Version == "latest" {
		t.Fatalf("expected resolved version, got %q", result.Version)
	}
	if !result.Available {
		t.Errorf("expected resolved latest to be available")
	}
}
