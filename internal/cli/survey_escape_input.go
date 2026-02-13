package cli

import (
	"os"
	"time"
)

const (
	surveyEscapeByte            = byte(0x1b)
	surveyInterruptByte         = byte(0x03)
	surveyEscapeSequenceTimeout = 25 * time.Millisecond
)

type surveyEscBackInput struct {
	file        *os.File
	backPressed bool
}

func newSurveyEscBackInput(file *os.File) *surveyEscBackInput {
	return &surveyEscBackInput{file: file}
}

func (i *surveyEscBackInput) Read(p []byte) (int, error) {
	n, err := i.file.Read(p)
	if n <= 0 {
		return n, err
	}

	for index := 0; index < n; index++ {
		if p[index] != surveyEscapeByte {
			continue
		}

		if index < n-1 {
			continue
		}

		if surveyInputHasBufferedSequenceData(i.file, surveyEscapeSequenceTimeout) {
			continue
		}

		p[index] = surveyInterruptByte
		i.backPressed = true
	}

	return n, err
}

func (i *surveyEscBackInput) Fd() uintptr {
	return i.file.Fd()
}

func (i *surveyEscBackInput) ConsumeBackPressed() bool {
	wasPressed := i.backPressed
	i.backPressed = false

	return wasPressed
}
