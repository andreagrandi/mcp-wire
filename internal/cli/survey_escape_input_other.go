//go:build !darwin && !linux

package cli

import (
	"os"
	"time"
)

func surveyInputHasBufferedSequenceData(_ *os.File, _ time.Duration) bool {
	return false
}
