package inmemory_test

import (
	"testing"

	"github.com/M0rfes/go-db/inmemory"
)

func TestInMemory(t *testing.T) {
	mem := inmemory.New()

	// Initial Get
	if val, ok := mem.Get("k1"); ok || val != "" {
		t.Fatalf("expected empty, false for initial Get; got %q, %v", val, ok)
	}

	// Initial Delete
	if mem.Delete("k1") {
		t.Fatalf("expected Delete on missing key to return false")
	}

	// Add
	if err := mem.Add("k1", "v1"); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Check Ref map
	if mem.Ref["k1"] != "v1" {
		t.Fatalf("expected Ref[k1] == v1, got %q", mem.Ref["k1"])
	}

	// Get
	val, ok := mem.Get("k1")
	if !ok || val != "v1" {
		t.Fatalf("Get(k1) = %q, %v; want %q, true", val, ok, "v1")
	}

	// Delete
	if !mem.Delete("k1") {
		t.Fatalf("Delete(k1) failed")
	}

	// Ref map updated
	if _, ok := mem.Ref["k1"]; ok {
		t.Fatalf("expected k1 deleted from Ref")
	}

	// Get after Delete
	if val, ok := mem.Get("k1"); ok || val != "" {
		t.Fatalf("expected not found after delete, got %q, %v", val, ok)
	}
}
