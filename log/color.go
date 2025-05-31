package log

import "fmt"

// ANSI codes
const (
	// Text colors
	textBlack = "30"
	textRed   = "31"
	textGreen = "32"
	textWhite = "37"
	textGray  = "90"

	// Background colors
	bgBlack = "40"
	bgRed   = "41"
	bgGreen = "42"

	// Text styles
	styleReset     = "0"
	styleBold      = "1"
	styleDim       = "2"
	styleItalic    = "3"
	styleStrikeout = "9"
)

// Public style helpers
var (
	ColorRed   = ansiWrapper(textRed)
	ColorGreen = ansiWrapper(textGreen)
	ColorWhite = ansiWrapper(textWhite)
	ColorGray  = ansiWrapper(textGray)

	BgBlack = ansiWrapper(bgBlack)
	BgRed   = ansiWrapper(bgRed)
	BgGreen = ansiWrapper(bgGreen)

	Reset     = ansiWrapper(styleReset)
	Bold      = ansiWrapper(styleBold)
	Dim       = ansiWrapper(styleDim)
	Italic    = ansiWrapper(styleItalic)
	Strikeout = ansiWrapper(styleStrikeout)
)

func SuccessColor(v any) string {
	return ColorGreen(Bold(v))
}

func ErrorColor(v any) string {
	return ColorRed(Bold(v))
}

func ansiWrapper(code string) func(any) string {
	return func(msg any) string {
		return fmt.Sprintf("\x1b[%sm%v\x1b[0m", code, msg)
	}
}
