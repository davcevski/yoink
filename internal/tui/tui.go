// Package tui renders yoink's clipboard browser with Bubble Tea and Lip Gloss.
// Clips are decrypted once into memory at startup; search filters that in-memory
// set with no further decryption.
//
// Keys reconcile the design's "type to search" with its vim-style bindings via a
// light mode split:
//
//	normal mode: ↑/↓ or j/k move · enter yoink · d delete · / search ·
//	             </> change limit · q/esc quit
//	             (pressing any other letter starts an incremental search)
//	search mode: type to filter · ↑/↓ move · enter yoink · ^d delete · esc clear
package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/davcevski/yoink/internal/crypto"
	"github.com/davcevski/yoink/internal/store"
)

// Item is a decrypted clip held in memory for the session.
type Item struct {
	ID        int64
	Text      string
	CreatedAt time.Time
	Bytes     int
}

// pasteboard is the one clipboard operation the TUI needs: writing a clip back.
type pasteboard interface {
	Write(string) error
}

// deleter is the one store operation the TUI needs: removing a clip.
type deleter interface {
	Delete(int64) error
}

// confirmQuitMsg is delivered shortly after a yoink so the confirmation is
// visible before the program exits.
type confirmQuitMsg struct{}

// Decrypt turns stored ciphertext clips into in-memory Items. Clips that fail to
// decrypt (e.g. after a key change) are skipped rather than aborting the browse.
func Decrypt(clips []store.Clip, c *crypto.Cipher) []Item {
	items := make([]Item, 0, len(clips))
	for _, cl := range clips {
		pt, err := c.Open(cl.Content, cl.Nonce)
		if err != nil {
			continue
		}
		items = append(items, Item{ID: cl.ID, Text: string(pt), CreatedAt: cl.CreatedAt, Bytes: cl.Bytes})
	}
	return items
}

// Model is the Bubble Tea model.
type Model struct {
	items    []Item
	filtered []int // indices into items, best match first
	cursor   int

	searching bool
	input     textinput.Model

	clip   pasteboard
	del    deleter
	limit  int
	limits []int

	th       Theme
	width    int
	status   string
	quitting bool
}

// New builds a Model over the already-decrypted items. limit is the initial
// display limit.
func New(items []Item, clip pasteboard, del deleter, limit int) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type to filter…"

	m := Model{
		items:  items,
		input:  ti,
		clip:   clip,
		del:    del,
		limit:  limit,
		limits: dedupLimits(limit),
		th:     NewTheme(),
		width:  80,
	}
	m.filtered = filter(items, "")
	return m
}

// dedupLimits returns the 5/10/20 cycle, inserting a custom launch limit if it
// is not already one of them.
func dedupLimits(limit int) []int {
	base := []int{5, 10, 20}
	for _, l := range base {
		if l == limit {
			return base
		}
	}
	return append([]int{limit}, base...)
}

// Status returns the last status line (e.g. the yoink confirmation), letting the
// caller echo it to the normal screen after the alt-screen program exits.
func (m Model) Status() string { return m.status }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case confirmQuitMsg:
		m.quitting = true
		return m, tea.Quit
	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Controls available in either mode.
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quitting = true
		return tea.Quit
	case tea.KeyCtrlD:
		return m.deleteSelected()
	case tea.KeyUp:
		m.moveUp()
		return nil
	case tea.KeyDown:
		m.moveDown()
		return nil
	case tea.KeyEnter:
		return m.yoink()
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	return m.handleNormalKey(msg)
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "k":
		m.moveUp()
	case "j":
		m.moveDown()
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = m.visibleCount() - 1
		m.clampCursor()
	case "d":
		return m.deleteSelected()
	case "q", "esc":
		m.quitting = true
		return tea.Quit
	case "/":
		m.enterSearch("")
		return textinput.Blink
	case ">", "tab", "+":
		m.cycleLimit(1)
	case "<", "shift+tab", "-":
		m.cycleLimit(-1)
	default:
		// Any other printable key starts an incremental search.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
			m.enterSearch(string(msg.Runes))
			return textinput.Blink
		}
	}
	return nil
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) tea.Cmd {
	if msg.Type == tea.KeyEsc {
		m.exitSearch()
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.reFilter()
	return cmd
}

func (m *Model) enterSearch(seed string) {
	m.searching = true
	m.input.SetValue(seed)
	m.input.CursorEnd()
	m.input.Focus()
	m.reFilter()
}

func (m *Model) exitSearch() {
	m.searching = false
	m.input.SetValue("")
	m.input.Blur()
	m.reFilter()
}

func (m *Model) reFilter() {
	m.filtered = filter(m.items, m.input.Value())
	m.clampCursor()
}

func (m *Model) yoink() tea.Cmd {
	it, ok := m.selected()
	if !ok {
		return nil
	}
	if err := m.clip.Write(it.Text); err != nil {
		m.status = "✗ write failed: " + err.Error()
		return nil
	}
	m.status = "✓ yoinked " + humanizeBytes(it.Bytes) + " back to the clipboard"
	return tea.Tick(700*time.Millisecond, func(time.Time) tea.Msg { return confirmQuitMsg{} })
}

func (m *Model) deleteSelected() tea.Cmd {
	it, ok := m.selected()
	if !ok {
		return nil
	}
	if err := m.del.Delete(it.ID); err != nil {
		m.status = "✗ delete failed: " + err.Error()
		return nil
	}
	for i := range m.items {
		if m.items[i].ID == it.ID {
			m.items = append(m.items[:i], m.items[i+1:]...)
			break
		}
	}
	m.status = "✗ deleted clip"
	m.reFilter()
	return nil
}

func (m *Model) cycleLimit(dir int) {
	idx := 0
	for i, l := range m.limits {
		if l == m.limit {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(m.limits)) % len(m.limits)
	m.limit = m.limits[idx]
	m.clampCursor()
}

func (m *Model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) moveDown() {
	if m.cursor < m.visibleCount()-1 {
		m.cursor++
	}
}

func (m *Model) clampCursor() {
	if max := m.visibleCount(); m.cursor >= max {
		m.cursor = max - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) visibleCount() int {
	if len(m.filtered) < m.limit {
		return len(m.filtered)
	}
	return m.limit
}

func (m *Model) selected() (Item, bool) {
	vc := m.visibleCount()
	if vc == 0 || m.cursor < 0 || m.cursor >= vc {
		return Item{}, false
	}
	return m.items[m.filtered[m.cursor]], true
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		if m.status != "" {
			return m.th.Status.Render(m.status) + "\n"
		}
		return ""
	}

	var b strings.Builder
	b.WriteString(Banner(m.th))
	b.WriteByte('\n')
	b.WriteString(m.th.Subtitle.Render("  ▚▚ clipboard history ▞▞"))
	b.WriteString("\n\n")
	b.WriteString(m.searchLine())
	b.WriteString("\n\n")
	b.WriteString(m.listView())
	b.WriteByte('\n')
	if m.status != "" {
		b.WriteByte('\n')
		b.WriteString(m.th.Status.Render(m.status))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(m.helpView())

	return m.th.Frame.Render(b.String())
}

func (m Model) searchLine() string {
	label := m.th.SearchLabel.Render("search ▶ ")
	if m.searching {
		return label + m.input.View()
	}
	return label + m.th.QueryHint.Render("type to filter…")
}

func (m Model) listView() string {
	if len(m.items) == 0 {
		return m.th.Empty.Render("  (no clips yet — copy something and it shows up here)")
	}
	vc := m.visibleCount()
	if vc == 0 {
		return m.th.Empty.Render("  (no matches)")
	}
	width := m.contentWidth()
	rows := make([]string, 0, vc)
	for i := 0; i < vc; i++ {
		rows = append(rows, m.rowView(i, m.items[m.filtered[i]], width, i == m.cursor))
	}
	return strings.Join(rows, "\n")
}

func (m Model) rowView(i int, it Item, width int, selected bool) string {
	idx := fmt.Sprintf(" %2d ", i+1)
	meta := fmt.Sprintf("%s · %s", humanizeTime(it.CreatedAt), humanizeBytes(it.Bytes))
	previewWidth := width - lipgloss.Width(idx) - lipgloss.Width(meta) - 3
	if previewWidth < 8 {
		previewWidth = 8
	}
	preview := padRight(Preview(it.Text, previewWidth), previewWidth)

	if selected {
		return m.th.SelBar.Render("▌") +
			m.th.SelIndex.Render(idx) +
			m.th.SelPreview.Render(preview) + " " +
			m.th.SelMeta.Render(meta)
	}
	return " " +
		m.th.Index.Render(idx) +
		m.th.Preview.Render(preview) + " " +
		m.th.Meta.Render(meta)
}

func (m Model) helpView() string {
	var keys []string
	if m.searching {
		keys = []string{"↑/↓ move", "enter yoink", "^d delete", "esc clear"}
	} else {
		keys = []string{
			"↑/↓·j/k move",
			"enter yoink",
			"d delete",
			"/ search",
			"<> limit(" + strconv.Itoa(m.limit) + ")",
			"q quit",
		}
	}
	return m.th.Help.Render(strings.Join(keys, "   "))
}

func (m Model) contentWidth() int {
	w := m.width - 8 // frame border + padding
	switch {
	case w < 40:
		return 40
	case w > 100:
		return 100
	default:
		return w
	}
}

// Preview collapses whitespace in s and truncates it to max runes with an
// ellipsis, producing the one-line summary shown in the list and by `yoink
// search`.
func Preview(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func humanizeBytes(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}

func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func padRight(s string, w int) string {
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}
