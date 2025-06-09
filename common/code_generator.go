package common

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// RandomCodeOption lets you customize the type of random code generated.
type RandomCodeOption int

const (
	RandomCodeAlpha        RandomCodeOption = iota // Letters only (A-Z)
	RandomCodeNumeric                              // Numbers only (0-9)
	RandomCodeAlphaNumeric                         // Letters (A-Z) + numbers (0-9)
)

// RandomCode generates a random code with flexible separator/grouping.
// n: total character count (not including separators)
// option: character set type
// separator: []any{string separator, int groupSize} (optional; groupSize default 4)
func RandomCode(n int, option RandomCodeOption, separator ...any) string {
	var charset string
	switch option {
	case RandomCodeAlpha:
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case RandomCodeNumeric:
		charset = "0123456789"
	case RandomCodeAlphaNumeric:
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	default:
		charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	chars := make([]byte, n)
	for i := range chars {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return ""
		}
		chars[i] = charset[num.Int64()]
	}

	// Separator options
	sep := ""
	group := 4
	if len(separator) > 0 {
		if s, ok := separator[0].(string); ok {
			sep = s
		}
		if len(separator) > 1 {
			if g, ok := separator[1].(int); ok && g > 0 {
				group = g
			}
		}
	}

	if sep != "" && group > 0 && n > group {
		var sb strings.Builder
		for i, c := range chars {
			if i > 0 && i%group == 0 {
				sb.WriteString(sep)
			}
			sb.WriteByte(c)
		}
		return sb.String()
	}

	return string(chars)
}
