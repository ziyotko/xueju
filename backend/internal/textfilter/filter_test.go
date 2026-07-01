package textfilter

import "testing"

func TestCleanKeepsNormalText(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.Clean("周末去万龙刻滑")
	if got != "周末去万龙刻滑" {
		t.Fatalf("Clean() = %q, want normal text unchanged", got)
	}
}

func TestCleanReplacesSensitiveWords(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.Clean("这里有赌博")
	if got != "这里有**" {
		t.Fatalf("Clean() = %q, want %q", got, "这里有**")
	}
}

func TestCleanReplacesPoliticalSensitiveWords(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.Clean("习近平")
	if got != "***" {
		t.Fatalf("Clean() = %q, want %q", got, "***")
	}
}

func TestCleanReplacesMultipleSensitiveWords(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.Clean("赌博和诈骗都不行")
	if got != "**和**都不行" {
		t.Fatalf("Clean() = %q, want %q", got, "**和**都不行")
	}
}

func TestCleanUsesLongestMatch(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.Clean("赌博网站")
	if got != "****" {
		t.Fatalf("Clean() = %q, want %q", got, "****")
	}
}

func TestCleanSliceKeepsLengthAndOrder(t *testing.T) {
	filter, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := filter.CleanSlice([]string{"刻滑", "赌博", "诈骗"})
	if len(got) != 3 {
		t.Fatalf("CleanSlice() length = %d, want 3", len(got))
	}
	if got[0] != "刻滑" || got[1] != "**" || got[2] != "**" {
		t.Fatalf("CleanSlice() = %#v", got)
	}
}
