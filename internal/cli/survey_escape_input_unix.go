//go:build darwin || linux

package cli

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

func surveyInputHasBufferedSequenceData(file *os.File, timeout time.Duration) bool {
	if file == nil {
		return false
	}

	timeoutMilliseconds := int(timeout / time.Millisecond)
	if timeoutMilliseconds < 0 {
		timeoutMilliseconds = 0
	}

	pollDescriptors := []unix.PollFd{{
		Fd:     int32(file.Fd()),
		Events: unix.POLLIN,
	}}

	ready, err := unix.Poll(pollDescriptors, timeoutMilliseconds)
	if err != nil || ready <= 0 {
		return false
	}

	return pollDescriptors[0].Revents&unix.POLLIN != 0
}
