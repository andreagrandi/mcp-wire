package credential

import "strings"

const (
	maskedPlaceholder = "********"
	maskedSuffixLen   = 4
)

// MaskSecret returns a display-safe representation of a secret value.
//
// Short or empty values are fully replaced with a fixed-length mask so the
// real length is not disclosed. Longer values keep the last four characters
// behind a fixed-length mask so the user can recognise a stored credential
// without seeing it in full.
func MaskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return maskedPlaceholder
	}

	if len(value) <= maskedSuffixLen*2 {
		return maskedPlaceholder
	}

	return maskedPlaceholder + value[len(value)-maskedSuffixLen:]
}
