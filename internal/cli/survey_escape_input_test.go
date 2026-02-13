package cli

import (
	"io"
	"os"
	"testing"
)

func TestSurveyEscBackInputTransformsStandaloneEscape(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed creating pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	if _, err := writer.Write([]byte{surveyEscapeByte}); err != nil {
		t.Fatalf("failed writing escape byte: %v", err)
	}

	input := newSurveyEscBackInput(reader)
	buffer := make([]byte, 1)

	bytesRead, err := input.Read(buffer)
	if err != nil {
		t.Fatalf("expected read to succeed: %v", err)
	}

	if bytesRead != 1 {
		t.Fatalf("expected 1 byte, got %d", bytesRead)
	}

	if buffer[0] != surveyInterruptByte {
		t.Fatalf("expected standalone escape to map to interrupt byte, got %v", buffer[0])
	}

	if !input.ConsumeBackPressed() {
		t.Fatal("expected back press marker to be set")
	}

	if input.ConsumeBackPressed() {
		t.Fatal("expected back press marker to be reset after consume")
	}
}

func TestSurveyEscBackInputPreservesArrowEscapeSequence(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed creating pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	go func() {
		_, _ = writer.Write([]byte{surveyEscapeByte, '[', 'A'})
		_ = writer.Close()
	}()

	input := newSurveyEscBackInput(reader)
	collected := make([]byte, 0, 3)
	temporaryBuffer := make([]byte, 3)

	for len(collected) < 3 {
		bytesRead, readErr := input.Read(temporaryBuffer)
		if bytesRead > 0 {
			collected = append(collected, temporaryBuffer[:bytesRead]...)
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}

			t.Fatalf("unexpected read error: %v", readErr)
		}
	}

	if len(collected) != 3 {
		t.Fatalf("expected full arrow escape sequence, got %v", collected)
	}

	if collected[0] != surveyEscapeByte || collected[1] != '[' || collected[2] != 'A' {
		t.Fatalf("unexpected escape sequence bytes: %v", collected)
	}

	if input.ConsumeBackPressed() {
		t.Fatal("did not expect back press marker for arrow sequence")
	}
}
