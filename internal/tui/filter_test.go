package tui

import (
	"testing"
	"time"
)

func items(texts ...string) []Item {
	out := make([]Item, len(texts))
	for i, t := range texts {
		out[i] = Item{ID: int64(i), Text: t, CreatedAt: time.Now(), Bytes: len(t)}
	}
	return out
}

func TestScoreEmptyQueryMatchesAll(t *testing.T) {
	if s, ok := score("", "anything"); !ok || s != 0 {
		t.Fatalf("empty query: got (%d,%v), want (0,true)", s, ok)
	}
}

func TestScoreSubstringBeatsFuzzy(t *testing.T) {
	sub, okSub := score("cat", "the cat sat")
	fuz, okFuz := score("cat", "c a t aaa")
	if !okSub || !okFuz {
		t.Fatalf("expected both to match: sub=%v fuz=%v", okSub, okFuz)
	}
	if sub <= fuz {
		t.Errorf("substring score %d should beat fuzzy score %d", sub, fuz)
	}
}

func TestScoreNoMatch(t *testing.T) {
	if _, ok := score("xyz", "the cat sat"); ok {
		t.Error("expected no match")
	}
}

func TestScoreCaseInsensitive(t *testing.T) {
	if _, ok := score("CAT", "a cat"); !ok {
		t.Error("expected case-insensitive match")
	}
}

func TestFilterOrdersByScoreThenRecency(t *testing.T) {
	its := items(
		"foobar",     // index 0: substring "foo"
		"f o o test", // index 1: fuzzy
		"nope",       // index 2: no match
		"food",       // index 3: substring "foo", shorter than index 0
	)
	got := filter(its, "foo")

	// "nope" must be excluded.
	for _, idx := range got {
		if idx == 2 {
			t.Fatal("non-matching item included")
		}
	}
	if len(got) != 3 {
		t.Fatalf("got %d matches, want 3", len(got))
	}
	// "food" (3) is a tighter substring than "foobar" (0), so it ranks first;
	// the fuzzy match (1) ranks last.
	if got[0] != 3 {
		t.Errorf("expected tightest substring first, got index %d", got[0])
	}
	if got[len(got)-1] != 1 {
		t.Errorf("expected fuzzy match last, got index %d", got[len(got)-1])
	}
}

func TestSearchReturnsRankedItemsWithLimit(t *testing.T) {
	its := items("food", "foobar", "nope", "f_o_o")
	got := Search(its, "foo", 2)
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2 (limit)", len(got))
	}
	if got[0].Text != "food" {
		t.Errorf("best match = %q, want food", got[0].Text)
	}
}

func TestSearchZeroLimitReturnsAll(t *testing.T) {
	its := items("a", "ab", "abc")
	if got := Search(its, "a", 0); len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
}

func TestFilterEmptyQueryPreservesOrder(t *testing.T) {
	its := items("a", "b", "c")
	got := filter(its, "")
	want := []int{0, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order not preserved: got %v, want %v", got, want)
		}
	}
}
