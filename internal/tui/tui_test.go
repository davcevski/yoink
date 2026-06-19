package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/davcevski/yoink/internal/clipboard"
)

type fakeDeleter struct{ deleted []int64 }

func (f *fakeDeleter) Delete(id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func update(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newTestModel(texts ...string) (Model, *clipboard.Fake, *fakeDeleter) {
	clip := &clipboard.Fake{}
	del := &fakeDeleter{}
	return New(items(texts...), clip, del, 20), clip, del
}

func TestNavigationDownUp(t *testing.T) {
	m, _, _ := newTestModel("a", "b", "c")
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	// Cannot move past the last item.
	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Fatalf("cursor moved past end: %d", m.cursor)
	}
	m = update(t, m, key("k")) // vim up
	if m.cursor != 1 {
		t.Fatalf("cursor after k = %d, want 1", m.cursor)
	}
}

func TestVimJMovesDown(t *testing.T) {
	m, _, _ := newTestModel("a", "b")
	m = update(t, m, key("j"))
	if m.cursor != 1 {
		t.Fatalf("cursor after j = %d, want 1", m.cursor)
	}
	if m.searching {
		t.Error("j should navigate in normal mode, not start a search")
	}
}

func TestTypingStartsIncrementalSearch(t *testing.T) {
	m, _, _ := newTestModel("apple", "banana", "cherry")
	m = update(t, m, key("b")) // 'b' is not a command → search
	if !m.searching {
		t.Fatal("typing did not start search mode")
	}
	if got := m.input.Value(); got != "b" {
		t.Fatalf("query = %q, want \"b\"", got)
	}
	if vc := m.visibleCount(); vc != 1 {
		t.Fatalf("filtered to %d items, want 1 (banana)", vc)
	}
	it, ok := m.selected()
	if !ok || it.Text != "banana" {
		t.Fatalf("selected = %+v ok=%v, want banana", it, ok)
	}
}

func TestEnterYoinksSelected(t *testing.T) {
	m, clip, _ := newTestModel("first", "second")
	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown}) // select "second"
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(clip.Written) != 1 || clip.Written[0] != "second" {
		t.Fatalf("clipboard writes = %v, want [second]", clip.Written)
	}
	if m.status == "" {
		t.Error("expected a confirmation status after yoink")
	}
}

func TestCtrlDDeletesSelected(t *testing.T) {
	m, _, del := newTestModel("keep", "remove-me", "alsokeep")
	m = update(t, m, tea.KeyMsg{Type: tea.KeyDown}) // select index 1
	target := m.items[m.filtered[m.cursor]]

	m = update(t, m, tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(del.deleted) != 1 || del.deleted[0] != target.ID {
		t.Fatalf("deleted = %v, want [%d]", del.deleted, target.ID)
	}
	if len(m.items) != 2 {
		t.Fatalf("items after delete = %d, want 2", len(m.items))
	}
	for _, it := range m.items {
		if it.ID == target.ID {
			t.Fatal("deleted item still present in model")
		}
	}
}

func TestDInNormalModeDeletes(t *testing.T) {
	m, _, del := newTestModel("one", "two")
	m = update(t, m, key("d"))
	if len(del.deleted) != 1 {
		t.Fatalf("d did not delete in normal mode: %v", del.deleted)
	}
}

func TestQQuits(t *testing.T) {
	m, _, _ := newTestModel("a")
	m = update(t, m, key("q"))
	if !m.quitting {
		t.Fatal("q did not set quitting")
	}
}

func TestEscClearsSearchThenStaysOpen(t *testing.T) {
	m, _, _ := newTestModel("apple", "banana")
	m = update(t, m, key("a")) // enter search
	if !m.searching {
		t.Fatal("not searching after typing")
	}
	m = update(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searching {
		t.Error("esc did not exit search mode")
	}
	if m.quitting {
		t.Error("esc from search should not quit")
	}
	if vc := m.visibleCount(); vc != 2 {
		t.Errorf("filter not reset after esc: %d visible", vc)
	}
}

func TestCycleLimit(t *testing.T) {
	m, _, _ := newTestModel("a", "b", "c", "d", "e", "f")
	if m.limit != 20 {
		t.Fatalf("initial limit = %d, want 20", m.limit)
	}
	m = update(t, m, key(">")) // 20 -> 5 (wraps)
	if m.limit != 5 {
		t.Fatalf("limit after > = %d, want 5", m.limit)
	}
	m = update(t, m, key("<")) // back to 20
	if m.limit != 20 {
		t.Fatalf("limit after < = %d, want 20", m.limit)
	}
}

func TestViewRendersWithoutPanic(t *testing.T) {
	m, _, _ := newTestModel("hello world", "another clip")
	if out := m.View(); out == "" {
		t.Fatal("View returned empty output")
	}
	empty := New(nil, &clipboard.Fake{}, &fakeDeleter{}, 20)
	if out := empty.View(); out == "" {
		t.Fatal("empty View returned no output")
	}
}
