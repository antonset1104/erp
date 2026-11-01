package tax_ebupot

import (
	"strings"
)

type terBracket struct {
	maxGross float64
	rate     float64 // percentage, e.g. 0.25 = 0.25%
}

// DetermineTERCategory returns "TER_A", "TER_B", or "TER_C" based on PTKP
func DetermineTERCategory(ptkp string) string {
	clean := strings.ToUpper(strings.TrimSpace(ptkp))
	switch clean {
	case "TK/0", "TK/1", "K/0":
		return "TER_A"
	case "TK/2", "TK/3", "K/1", "K/2":
		return "TER_B"
	case "K/3":
		return "TER_C"
	default:
		// Default single/unspecified is TK/0 -> TER A
		return "TER_A"
	}
}

// PP 58/2023 - TER A brackets
var terABrackets = []terBracket{
	{5400000, 0.0},
	{5650000, 0.25},
	{5950000, 0.50},
	{6300000, 0.75},
	{6750000, 1.00},
	{7500000, 1.25},
	{8550000, 1.50},
	{9650000, 1.75},
	{10050000, 2.00},
	{10350000, 2.25},
	{10700000, 2.50},
	{11050000, 3.00},
	{11600000, 3.50},
	{12500000, 4.00},
	{13750000, 5.00},
	{15100000, 6.00},
	{16950000, 7.00},
	{19750000, 8.00},
	{24150000, 9.00},
	{26450000, 10.00},
	{28000000, 11.00},
	{30050000, 12.00},
	{32400000, 13.00},
	{35400000, 14.00},
	{39100000, 15.00},
	{43850000, 16.00},
	{47800000, 17.00},
	{51400000, 18.00},
	{56300000, 19.00},
	{62200000, 20.00},
	{68600000, 21.00},
	{77500000, 22.00},
	{89000000, 23.00},
	{103000000, 24.00},
	{125000000, 25.00},
	{157000000, 26.00},
	{206000000, 27.00},
	{337000000, 28.00},
	{454000000, 29.00},
	{550000000, 30.00},
	{695000000, 31.00},
	{910000000, 32.00},
	{1400000000, 33.00},
}

// PP 58/2023 - TER B brackets
var terBBrackets = []terBracket{
	{6200000, 0.0},
	{6500000, 0.25},
	{6850000, 0.50},
	{7300000, 0.75},
	{9200000, 1.00},
	{10750000, 1.50},
	{11250000, 2.00},
	{11600000, 2.50},
	{12600000, 3.00},
	{13600000, 4.00},
	{14950000, 5.00},
	{16400000, 6.00},
	{18450000, 7.00},
	{21850000, 8.00},
	{26000000, 9.00},
	{27700000, 10.00},
	{29350000, 11.00},
	{31450000, 12.00},
	{33950000, 13.00},
	{37100000, 14.00},
	{41100000, 15.00},
	{45800000, 16.00},
	{49500000, 17.00},
	{53800000, 18.00},
	{58500000, 19.00},
	{64000000, 20.00},
	{71000000, 21.00},
	{80000000, 22.00},
	{93000000, 23.00},
	{109000000, 24.00},
	{129000000, 25.00},
	{163000000, 26.00},
	{211000000, 27.00},
	{374000000, 28.00},
	{459000000, 29.00},
	{555000000, 30.00},
	{704000000, 31.00},
	{957000000, 32.00},
	{1405000000, 33.00},
}

// PP 58/2023 - TER C brackets
var terCBrackets = []terBracket{
	{6600000, 0.0},
	{6950000, 0.25},
	{7350000, 0.50},
	{7800000, 0.75},
	{8850000, 1.00},
	{9800000, 1.25},
	{10950000, 1.50},
	{11200000, 1.75},
	{12050000, 2.00},
	{12950000, 3.00},
	{14150000, 4.00},
	{15550000, 5.00},
	{17050000, 6.00},
	{19500000, 7.00},
	{22700000, 8.00},
	{26600000, 9.00},
	{28100000, 10.00},
	{30100000, 11.00},
	{32600000, 12.00},
	{35400000, 13.00},
	{38900000, 14.00},
	{43000000, 15.00},
	{47400000, 16.00},
	{51200000, 17.00},
	{55800000, 18.00},
	{60400000, 19.00},
	{66700000, 20.00},
	{74500000, 21.00},
	{83200000, 22.00},
	{95600000, 23.00},
	{110000000, 24.00},
	{134000000, 25.00},
	{169000000, 26.00},
	{221000000, 27.00},
	{390000000, 28.00},
	{463000000, 29.00},
	{561000000, 30.00},
	{709000000, 31.00},
	{965000000, 32.00},
	{1419000000, 33.00},
}

// CalculateTERRate calculates the effective percentage rate for the given TER category and monthly gross income
func CalculateTERRate(terCat string, grossMonthly float64) float64 {
	if grossMonthly <= 0 {
		return 0.0
	}

	var brackets []terBracket
	switch terCat {
	case "TER_B":
		brackets = terBBrackets
	case "TER_C":
		brackets = terCBrackets
	default:
		brackets = terABrackets
	}

	for _, b := range brackets {
		if grossMonthly <= b.maxGross {
			return b.rate
		}
	}

	// Above highest threshold
	return 34.0
}

// CalculatePPh21Monthly calculates TER Category, Rate (%), and Tax Withheld Amount
func CalculatePPh21Monthly(ptkp string, grossMonthly float64) (string, float64, float64) {
	terCat := DetermineTERCategory(ptkp)
	ratePct := CalculateTERRate(terCat, grossMonthly)
	taxAmount := (ratePct / 100.0) * grossMonthly
	return terCat, ratePct, taxAmount
}
