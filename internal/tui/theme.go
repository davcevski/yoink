package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorCyan   = lipgloss.Color("#6aece1") // primary text / borders
	colorTeal   = lipgloss.Color("#26ccc2") // secondary accent / metadata
	colorYellow = lipgloss.Color("#fff57e") // selection / highlight
	colorOrange = lipgloss.Color("#ffb76c") // confirmations / warnings
	colorDim    = lipgloss.Color("#2f6f6b") // dim text
	colorInk    = lipgloss.Color("#08221f") // text on highlighted rows
)

// pixelBorder is a half-block frame.
var pixelBorder = lipgloss.Border{
	Top:         "▀",
	Bottom:      "▄",
	Left:        "█",
	Right:       "█",
	TopLeft:     "█",
	TopRight:    "█",
	BottomLeft:  "█",
	BottomRight: "█",
}

// Theme bundles every Lip Gloss style the view uses. Grouping them keeps the
// render code declarative and the styles in one place.
type Theme struct {
	// Banner colors, applied per letter of YOINK.
	BannerColors []lipgloss.Style

	Frame    lipgloss.Style
	Subtitle lipgloss.Style

	SearchLabel lipgloss.Style
	Query       lipgloss.Style
	QueryHint   lipgloss.Style

	Index   lipgloss.Style
	Preview lipgloss.Style
	Meta    lipgloss.Style

	SelBar     lipgloss.Style
	SelIndex   lipgloss.Style
	SelPreview lipgloss.Style
	SelMeta    lipgloss.Style

	Status lipgloss.Style
	Help   lipgloss.Style
	Empty  lipgloss.Style
}

// NewTheme builds the default theme.
func NewTheme() Theme {
	return Theme{
		BannerColors: []lipgloss.Style{
			lipgloss.NewStyle().Foreground(colorCyan).Bold(true),
			lipgloss.NewStyle().Foreground(colorTeal).Bold(true),
			lipgloss.NewStyle().Foreground(colorYellow).Bold(true),
			lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
			lipgloss.NewStyle().Foreground(colorCyan).Bold(true),
		},
		Frame:       lipgloss.NewStyle().Border(pixelBorder).BorderForeground(colorTeal).Padding(0, 2),
		Subtitle:    lipgloss.NewStyle().Foreground(colorTeal),
		SearchLabel: lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
		Query:       lipgloss.NewStyle().Foreground(colorCyan),
		QueryHint:   lipgloss.NewStyle().Foreground(colorDim).Italic(true),
		Index:       lipgloss.NewStyle().Foreground(colorTeal),
		Preview:     lipgloss.NewStyle().Foreground(colorCyan),
		Meta:        lipgloss.NewStyle().Foreground(colorDim),
		SelBar:      lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
		SelIndex:    lipgloss.NewStyle().Foreground(colorInk).Background(colorYellow).Bold(true),
		SelPreview:  lipgloss.NewStyle().Foreground(colorInk).Background(colorYellow).Bold(true),
		SelMeta:     lipgloss.NewStyle().Foreground(colorInk).Background(colorYellow),
		Status:      lipgloss.NewStyle().Foreground(colorOrange).Bold(true),
		Help:        lipgloss.NewStyle().Foreground(colorDim),
		Empty:       lipgloss.NewStyle().Foreground(colorDim).Italic(true),
	}
}
