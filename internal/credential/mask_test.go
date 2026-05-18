package credential

import (
	"strings"
	"testing"
)

func TestMaskSecretMasksEmptyValue(t *testing.T) {
	masked := MaskSecret("")
	if masked != "********" {
		t.Fatalf("expected fixed placeholder for empty value, got %q", masked)
	}
}

func TestMaskSecretMasksWhitespaceOnlyValue(t *testing.T) {
	masked := MaskSecret("   ")
	if masked != "********" {
		t.Fatalf("expected fixed placeholder for whitespace value, got %q", masked)
	}
}

func TestMaskSecretFullyMasksShortValue(t *testing.T) {
	masked := MaskSecret("abcd1234")
	if masked != "********" {
		t.Fatalf("expected short value to be fully masked, got %q", masked)
	}
}

func TestMaskSecretRevealsLastFourCharsForLongValue(t *testing.T) {
	masked := MaskSecret("super-secret-token-XYZQ")
	if masked != "********XYZQ" {
		t.Fatalf("expected last four chars revealed, got %q", masked)
	}
}

func TestMaskSecretDoesNotLeakFullValue(t *testing.T) {
	value := "another-long-secret-value-9876"
	masked := MaskSecret(value)
	if strings.Contains(masked, value) {
		t.Fatalf("masked output %q must not contain full secret %q", masked, value)
	}

	if strings.HasPrefix(masked, value[:5]) {
		t.Fatalf("masked output %q must not expose the secret prefix", masked)
	}
}
