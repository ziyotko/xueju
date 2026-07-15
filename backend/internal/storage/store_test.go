package storage

import (
	"context"
	"os"
	"testing"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	root, err := os.MkdirTemp("", "xueju-storage-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	store := &localStore{root: root}
	want := []byte("image payload")
	if err := store.Put(context.Background(), "avatars/a.jpg", want, "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	got, _, err := store.Get(context.Background(), "avatars/a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q want %q", got, want)
	}
	if err := store.Delete(context.Background(), "avatars/a.jpg"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalStoreRejectsTraversal(t *testing.T) {
	store := &localStore{root: t.TempDir()}
	if err := store.Put(context.Background(), "../escape", []byte("x"), "text/plain"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}
