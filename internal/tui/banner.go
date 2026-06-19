package tui

import "strings"

// bannerHeight is the row count of every glyph below.
const bannerHeight = 5

// glyphs holds a 5-row block-letter rendering of each character in YOINK.
var glyphs = map[rune][]string{
	'Y': {
		"█   █",
		" █ █ ",
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'O': {
		" ███ ",
		"█   █",
		"█   █",
		"█   █",
		" ███ ",
	},
	'I': {
		"█████",
		"  █  ",
		"  █  ",
		"  █  ",
		"█████",
	},
	'N': {
		"█   █",
		"██  █",
		"█ █ █",
		"█  ██",
		"█   █",
	},
	'K': {
		"█   █",
		"█  █ ",
		"███  ",
		"█  █ ",
		"█   █",
	},
}

// Banner renders the YOINK wordmark.
func Banner(th Theme) string {
	const word = "YOINK"
	rows := make([]string, bannerHeight)
	for i, r := range word {
		style := th.BannerColors[i%len(th.BannerColors)]
		letter := glyphs[r]
		for row := 0; row < bannerHeight; row++ {
			rows[row] += style.Render(letter[row]) + "  "
		}
	}
	return strings.Join(rows, "\n")
}
