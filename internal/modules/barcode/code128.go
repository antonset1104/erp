package barcode

import (
	"fmt"
	"html"
	"strings"
)

// code128Patterns represents widths of bars and spaces for symbols 0 through 106.
// Each symbol has 6 digits representing alternating bar and space widths (summing to 11 modules),
// except symbol 106 (Stop character) which has 7 digits (summing to 13 modules).
var code128Patterns = []string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213", // 0-9
	"221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132", // 10-19
	"221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211", // 20-29
	"212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313", // 30-39
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331", // 40-49
	"231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111", // 50-59
	"314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214", // 60-69
	"112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111", // 70-79
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141", // 80-89
	"214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141", // 90-99
	"114131", "311141", "411131", "211412", "211214", "211232", "2331112", // 100-106 (104=Start B, 105=Start C, 106=Stop)
}

// GenerateCode128SVG encodes string into vector SVG Code 128 format
func GenerateCode128SVG(text string, title string, subtitle string, priceText string) string {
	if text == "" {
		text = "00000000"
	}

	// Subset B: supports ASCII 32 to 126
	var values []int
	for _, ch := range text {
		val := int(ch) - 32
		if val < 0 || val > 95 {
			val = 0 // fallback for non-printable characters
		}
		values = append(values, val)
	}

	// Start Code B is index 104
	startSymbol := 104
	checksum := startSymbol
	for i, val := range values {
		checksum += (i + 1) * val
	}
	checksumSymbol := checksum % 103
	stopSymbol := 106

	// Compile sequence of symbols
	var sequence []int
	sequence = append(sequence, startSymbol)
	sequence = append(sequence, values...)
	sequence = append(sequence, checksumSymbol)
	sequence = append(sequence, stopSymbol)

	// Build alternating bar/space modules
	var modules []bool // true = bar, false = space
	for _, symIdx := range sequence {
		pattern := code128Patterns[symIdx]
		isBar := true
		for _, charWidth := range pattern {
			width := int(charWidth - '0')
			for w := 0; w < width; w++ {
				modules = append(modules, isBar)
			}
			isBar = !isBar
		}
	}

	// Layout parameters
	moduleWidth := 2
	quietZone := 20
	barcodeWidth := len(modules) * moduleWidth
	totalWidth := barcodeWidth + (quietZone * 2)
	if totalWidth < 280 {
		totalWidth = 280
	}
	offsetX := (totalWidth - barcodeWidth) / 2

	barcodeHeight := 65
	headerHeight := 45
	if title == "" && subtitle == "" {
		headerHeight = 15
	}
	totalHeight := headerHeight + barcodeHeight + 45

	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" style="background: #ffffff; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;">`,
		totalWidth, totalHeight, totalWidth, totalHeight))

	// Border & Background
	svg.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#ffffff" rx="8" stroke="#e2e8f0" stroke-width="1.5"/>`, totalWidth, totalHeight))

	currY := 20
	if title != "" {
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="13" font-weight="700" fill="#0f172a">%s</text>`,
			totalWidth/2, currY, html.EscapeString(title)))
		currY += 16
	}
	if subtitle != "" || priceText != "" {
		subStr := subtitle
		if priceText != "" {
			if subStr != "" {
				subStr += "  |  " + priceText
			} else {
				subStr = priceText
			}
		}
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="11" font-weight="600" fill="#2563eb">%s</text>`,
			totalWidth/2, currY, html.EscapeString(subStr)))
		currY += 14
	}

	// Barcode Bars
	barStartY := currY + 5
	currX := offsetX
	for _, isBar := range modules {
		if isBar {
			svg.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="#000000"/>`,
				currX, barStartY, moduleWidth, barcodeHeight))
		}
		currX += moduleWidth
	}

	// Human-readable code text
	textY := barStartY + barcodeHeight + 18
	svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="12" font-weight="700" letter-spacing="3" fill="#1e293b">%s</text>`,
		totalWidth/2, textY, html.EscapeString(text)))

	svg.WriteString(`</svg>`)
	return svg.String()
}
