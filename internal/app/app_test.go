package app

import "testing"

func TestAppVersion(t *testing.T) {
	app := New()

	if app.Version != Version {
		t.Errorf("Expected version %s, got %s", Version, app.Version)
	}

	if app.Name != Name {
		t.Errorf("Expected name %s, got %s", Name, app.Name)
	}

	expectedFullVersion := Name + " version " + Version
	if app.GetFullVersion() != expectedFullVersion {
		t.Errorf("Expected full version '%s', got '%s'", expectedFullVersion, app.GetFullVersion())
	}
}

func TestAppConstants(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}

	if Name == "" {
		t.Error("Name constant should not be empty")
	}

	if Author == "" {
		t.Error("Author constant should not be empty")
	}

	if License == "" {
		t.Error("License constant should not be empty")
	}
}
