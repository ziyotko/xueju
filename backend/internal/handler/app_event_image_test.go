package handler

import "testing"

func TestEventUploadStorageKeyIgnoresSchemeAndHost(t *testing.T) {
	for _, imageURL := range []string{
		"http://www.xueju.xyz/uploads/events/12-345.jpg",
		"https://www.xueju.xyz/uploads/events/12-345.jpg",
	} {
		got, ok := eventUploadStorageKey(imageURL)
		if !ok || got != "events/12-345.jpg" {
			t.Fatalf("eventUploadStorageKey(%q) = %q, %v", imageURL, got, ok)
		}
	}
}

func TestEventUploadStorageKeyRejectsUnrelatedPaths(t *testing.T) {
	for _, imageURL := range []string{
		"https://www.xueju.xyz/uploads/avatars/12-345.jpg",
		"https://www.xueju.xyz/uploads/events/",
		"https://www.xueju.xyz/uploads/events/../avatars/12-345.jpg",
		"https://www.xueju.xyz/uploads/events/folder/12-345.jpg",
	} {
		if got, ok := eventUploadStorageKey(imageURL); ok {
			t.Fatalf("eventUploadStorageKey(%q) unexpectedly accepted %q", imageURL, got)
		}
	}
}
