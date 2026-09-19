package node_test

import (
	"bytes"
	"testing"

	"github.com/M0rfes/go-db/inmemory"
	"github.com/M0rfes/go-db/node"
)

func TestBTree_Insert(t *testing.T) {
	t.Run("insert into empty tree and get", func(t *testing.T) {
		mem := inmemory.New()
		key := []byte("hello")
		val := []byte("world")

		err := mem.Tree.Insert(key, val)
		if err != nil {
			t.Fatalf("unexpected error on Insert: %v", err)
		}

		got, ok := mem.Tree.Get(key)
		if !ok {
			t.Fatalf("expected key %q to be found", key)
		}
		if !bytes.Equal(got, val) {
			t.Fatalf("expected val %q, got %q", val, got)
		}
	})

	t.Run("insert key too large", func(t *testing.T) {
		mem := inmemory.New()
		key := make([]byte, node.BTREE_MAX_KEY_SIZE+1)
		val := []byte("val")

		err := mem.Tree.Insert(key, val)
		if err == nil {
			t.Fatalf("expected error for key exceeding BTREE_MAX_KEY_SIZE, got nil")
		}
	})

	t.Run("insert val too large", func(t *testing.T) {
		mem := inmemory.New()
		key := []byte("key")
		val := make([]byte, node.BTREE_MAX_VAL_SIZE+1)

		err := mem.Tree.Insert(key, val)
		if err == nil {
			t.Fatalf("expected error for val exceeding BTREE_MAX_VAL_SIZE, got nil")
		}
	})

	t.Run("update existing key", func(t *testing.T) {
		mem := inmemory.New()
		key := []byte("foo")
		val1 := []byte("bar1")
		val2 := []byte("bar2")

		if err := mem.Tree.Insert(key, val1); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}
		if err := mem.Tree.Insert(key, val2); err != nil {
			t.Fatalf("Insert update failed: %v", err)
		}

		got, ok := mem.Tree.Get(key)
		if !ok {
			t.Fatalf("expected key %q to be found", key)
		}
		if !bytes.Equal(got, val2) {
			t.Fatalf("expected val %q, got %q", val2, got)
		}
	})

	t.Run("insert multiple keys in ascending order", func(t *testing.T) {
		mem := inmemory.New()
		keys := [][]byte{
			[]byte("apple"),
			[]byte("banana"),
			[]byte("cherry"),
			[]byte("date"),
			[]byte("fig"),
		}

		for _, k := range keys {
			val := []byte("val_" + string(k))
			if err := mem.Tree.Insert(k, val); err != nil {
				t.Fatalf("Insert(%s) failed: %v", k, err)
			}
		}

		for _, k := range keys {
			expected := []byte("val_" + string(k))
			got, ok := mem.Tree.Get(k)
			if !ok {
				t.Fatalf("expected key %s to be found", k)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("for key %s expected %s, got %s", k, expected, got)
			}
		}
	})

	t.Run("insert multiple keys in descending order", func(t *testing.T) {
		mem := inmemory.New()
		keys := [][]byte{
			[]byte("zebra"),
			[]byte("yak"),
			[]byte("walrus"),
			[]byte("tiger"),
			[]byte("snake"),
		}

		for _, k := range keys {
			val := []byte("val_" + string(k))
			if err := mem.Tree.Insert(k, val); err != nil {
				t.Fatalf("Insert(%s) failed: %v", k, err)
			}
		}

		for _, k := range keys {
			expected := []byte("val_" + string(k))
			got, ok := mem.Tree.Get(k)
			if !ok {
				t.Fatalf("expected key %s to be found", k)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("for key %s expected %s, got %s", k, expected, got)
			}
		}
	})
}

func TestBTree_Get(t *testing.T) {
	t.Run("get on empty tree", func(t *testing.T) {
		mem := inmemory.New()
		got, ok := mem.Tree.Get([]byte("nonexistent"))
		if ok || got != nil {
			t.Fatalf("expected (nil, false), got (%v, %v)", got, ok)
		}
	})

	t.Run("get nonexistent key from non-empty tree", func(t *testing.T) {
		mem := inmemory.New()
		if err := mem.Tree.Insert([]byte("k2"), []byte("v2")); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		// Key smaller than existing key
		if got, ok := mem.Tree.Get([]byte("k1")); ok || got != nil {
			t.Fatalf("expected (nil, false) for smaller key, got (%v, %v)", got, ok)
		}

		// Key larger than existing key
		if got, ok := mem.Tree.Get([]byte("k3")); ok || got != nil {
			t.Fatalf("expected (nil, false) for larger key, got (%v, %v)", got, ok)
		}
	})

	t.Run("get keys with common prefixes", func(t *testing.T) {
		mem := inmemory.New()
		kvs := []struct {
			k string
			v string
		}{
			{"test", "val_test"},
			{"testing", "val_testing"},
			{"tester", "val_tester"},
		}

		for _, item := range kvs {
			if err := mem.Tree.Insert([]byte(item.k), []byte(item.v)); err != nil {
				t.Fatalf("Insert(%s) failed: %v", item.k, err)
			}
		}

		for _, item := range kvs {
			got, ok := mem.Tree.Get([]byte(item.k))
			if !ok {
				t.Fatalf("expected key %s to be found", item.k)
			}
			if string(got) != item.v {
				t.Fatalf("key %s: expected %s, got %s", item.k, item.v, string(got))
			}
		}
	})
}

func TestBTree_Delete(t *testing.T) {
	t.Run("delete from empty tree", func(t *testing.T) {
		mem := inmemory.New()
		ok := mem.Tree.Delete([]byte("nonexistent"))
		if ok {
			t.Fatalf("expected Delete on empty tree to return false")
		}
	})

	t.Run("delete nonexistent key from non-empty tree", func(t *testing.T) {
		mem := inmemory.New()
		if err := mem.Tree.Insert([]byte("key1"), []byte("val1")); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		if mem.Tree.Delete([]byte("missing")) {
			t.Fatalf("expected Delete of missing key to return false")
		}

		// Verify original key is still intact
		got, ok := mem.Tree.Get([]byte("key1"))
		if !ok || !bytes.Equal(got, []byte("val1")) {
			t.Fatalf("key1 should still exist with val1, got (%s, %v)", got, ok)
		}
	})

	t.Run("delete until empty", func(t *testing.T) {
		mem := inmemory.New()
		k1 := []byte("key1")
		v1 := []byte("val1")

		if err := mem.Tree.Insert(k1, v1); err != nil {
			t.Fatalf("Insert failed: %v", err)
		}

		deleted := mem.Tree.Delete(k1)
		if !deleted {
			t.Fatalf("expected Delete to return true")
		}

		// Tree should be empty now
		got, ok := mem.Tree.Get(k1)
		if ok || got != nil {
			t.Fatalf("expected (nil, false) after deleting only key, got (%v, %v)", got, ok)
		}

		// Deleting again should return false
		if mem.Tree.Delete(k1) {
			t.Fatalf("expected second Delete to return false")
		}
	})

	t.Run("delete one of multiple keys", func(t *testing.T) {
		mem := inmemory.New()
		keys := []string{"k1", "k2", "k3", "k4"}
		for _, k := range keys {
			if err := mem.Tree.Insert([]byte(k), []byte("v_"+k)); err != nil {
				t.Fatalf("Insert(%s) failed: %v", k, err)
			}
		}

		// Delete k2
		if !mem.Tree.Delete([]byte("k2")) {
			t.Fatalf("Delete(k2) returned false")
		}

		// Verify k2 is gone
		if _, ok := mem.Tree.Get([]byte("k2")); ok {
			t.Fatalf("k2 should have been deleted")
		}

		// Verify other keys are still present
		for _, k := range []string{"k1", "k3", "k4"} {
			got, ok := mem.Tree.Get([]byte(k))
			if !ok || string(got) != "v_"+k {
				t.Fatalf("expected key %s to exist with value v_%s, got (%s, %v)", k, k, string(got), ok)
			}
		}

		// Delete k1 (the first key)
		if !mem.Tree.Delete([]byte("k1")) {
			t.Fatalf("Delete(k1) returned false")
		}
		if _, ok := mem.Tree.Get([]byte("k1")); ok {
			t.Fatalf("k1 should have been deleted")
		}

		// Delete k4 (the last key)
		if !mem.Tree.Delete([]byte("k4")) {
			t.Fatalf("Delete(k4) returned false")
		}
		if _, ok := mem.Tree.Get([]byte("k4")); ok {
			t.Fatalf("k4 should have been deleted")
		}

		// Delete k3 (remaining key)
		if !mem.Tree.Delete([]byte("k3")) {
			t.Fatalf("Delete(k3) returned false")
		}
		if _, ok := mem.Tree.Get([]byte("k3")); ok {
			t.Fatalf("k3 should have been deleted")
		}

		// Tree should now be empty
		if _, ok := mem.Tree.Get([]byte("k3")); ok {
			t.Fatalf("tree should be empty")
		}
	})

	t.Run("delete in reverse order", func(t *testing.T) {
		mem := inmemory.New()
		keys := []string{"k1", "k2", "k3", "k4", "k5"}
		for _, k := range keys {
			if err := mem.Tree.Insert([]byte(k), []byte("v_"+k)); err != nil {
				t.Fatalf("Insert(%s) failed: %v", k, err)
			}
		}

		for i := len(keys) - 1; i >= 0; i-- {
			k := keys[i]
			if !mem.Tree.Delete([]byte(k)) {
				t.Fatalf("Delete(%s) failed", k)
			}
			if _, ok := mem.Tree.Get([]byte(k)); ok {
				t.Fatalf("key %s still found after delete", k)
			}
		}

		// Verify all deleted
		for _, k := range keys {
			if _, ok := mem.Tree.Get([]byte(k)); ok {
				t.Fatalf("key %s should not exist", k)
			}
		}
	})
}

func TestBTree_EdgeCases(t *testing.T) {
	t.Run("boundary key size", func(t *testing.T) {
		mem := inmemory.New()
		maxKey := bytes.Repeat([]byte("k"), node.BTREE_MAX_KEY_SIZE)
		val := []byte("val")

		if err := mem.Tree.Insert(maxKey, val); err != nil {
			t.Fatalf("Insert with key of exactly BTREE_MAX_KEY_SIZE failed: %v", err)
		}

		got, ok := mem.Tree.Get(maxKey)
		if !ok || !bytes.Equal(got, val) {
			t.Fatalf("Get with max-length key failed, got (%s, %v)", got, ok)
		}

		if !mem.Tree.Delete(maxKey) {
			t.Fatalf("Delete with max-length key failed")
		}
	})

	t.Run("boundary val size", func(t *testing.T) {
		mem := inmemory.New()
		key := []byte("key")
		maxVal := bytes.Repeat([]byte("v"), node.BTREE_MAX_VAL_SIZE)

		if err := mem.Tree.Insert(key, maxVal); err != nil {
			t.Fatalf("Insert with val of exactly BTREE_MAX_VAL_SIZE failed: %v", err)
		}

		got, ok := mem.Tree.Get(key)
		if !ok || !bytes.Equal(got, maxVal) {
			t.Fatalf("Get with max-length val failed")
		}

		if !mem.Tree.Delete(key) {
			t.Fatalf("Delete with max-length val failed")
		}
	})

	t.Run("empty val", func(t *testing.T) {
		mem := inmemory.New()
		key := []byte("key_with_empty_val")
		emptyVal := []byte{}

		if err := mem.Tree.Insert(key, emptyVal); err != nil {
			t.Fatalf("Insert with empty val failed: %v", err)
		}

		got, ok := mem.Tree.Get(key)
		if !ok {
			t.Fatalf("expected key to be found")
		}
		if len(got) != 0 {
			t.Fatalf("expected empty val, got length %d", len(got))
		}
	})

	t.Run("binary keys and values", func(t *testing.T) {
		mem := inmemory.New()
		k1 := []byte{0x00, 0x01, 0x02, 0x00}
		v1 := []byte{0xFF, 0xFE, 0x00, 0xFF}
		k2 := []byte{0x00, 0x01, 0x02, 0x01}
		v2 := []byte{0xAA, 0xBB, 0xCC}

		if err := mem.Tree.Insert(k1, v1); err != nil {
			t.Fatalf("Insert(k1) failed: %v", err)
		}
		if err := mem.Tree.Insert(k2, v2); err != nil {
			t.Fatalf("Insert(k2) failed: %v", err)
		}

		got1, ok1 := mem.Tree.Get(k1)
		if !ok1 || !bytes.Equal(got1, v1) {
			t.Fatalf("Get(k1) = (%v, %v); want (%v, true)", got1, ok1, v1)
		}

		got2, ok2 := mem.Tree.Get(k2)
		if !ok2 || !bytes.Equal(got2, v2) {
			t.Fatalf("Get(k2) = (%v, %v); want (%v, true)", got2, ok2, v2)
		}

		if !mem.Tree.Delete(k1) {
			t.Fatalf("Delete(k1) failed")
		}
		if _, ok := mem.Tree.Get(k1); ok {
			t.Fatalf("k1 should be deleted")
		}
		if got, ok := mem.Tree.Get(k2); !ok || !bytes.Equal(got, v2) {
			t.Fatalf("k2 should still exist")
		}
	})
}

func TestBTree_Lifecycle(t *testing.T) {
	mem := inmemory.New()
	key := []byte("user_1")
	val1 := []byte("alice")
	val2 := []byte("alice_updated")

	// 1. Initial Get on non-existent key
	if _, ok := mem.Tree.Get(key); ok {
		t.Fatalf("key should not exist initially")
	}

	// 2. Initial Delete on non-existent key
	if mem.Tree.Delete(key) {
		t.Fatalf("Delete should return false on non-existent key")
	}

	// 3. Insert key
	if err := mem.Tree.Insert(key, val1); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// 4. Get inserted key
	got, ok := mem.Tree.Get(key)
	if !ok || !bytes.Equal(got, val1) {
		t.Fatalf("Get = (%s, %v), want (%s, true)", got, ok, val1)
	}

	// 5. Update key
	if err := mem.Tree.Insert(key, val2); err != nil {
		t.Fatalf("Insert update failed: %v", err)
	}

	// 6. Get updated key
	got, ok = mem.Tree.Get(key)
	if !ok || !bytes.Equal(got, val2) {
		t.Fatalf("Get after update = (%s, %v), want (%s, true)", got, ok, val2)
	}

	// 7. Delete key
	if !mem.Tree.Delete(key) {
		t.Fatalf("Delete failed")
	}

	// 8. Get deleted key
	if _, ok := mem.Tree.Get(key); ok {
		t.Fatalf("Get after delete should return false")
	}

	// 9. Re-insert key
	val3 := []byte("alice_reinserted")
	if err := mem.Tree.Insert(key, val3); err != nil {
		t.Fatalf("Re-insert failed: %v", err)
	}

	// 10. Get re-inserted key
	got, ok = mem.Tree.Get(key)
	if !ok || !bytes.Equal(got, val3) {
		t.Fatalf("Get after re-insert = (%s, %v), want (%s, true)", got, ok, val3)
	}
}
