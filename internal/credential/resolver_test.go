package credential

import (
	"errors"
	"testing"
)

type fakeSource struct {
	name   string
	values map[string]string
}

func (f fakeSource) Name() string {
	return f.name
}

func (f fakeSource) Get(envName string) (string, bool) {
	value, ok := f.values[envName]
	return value, ok
}

func (f fakeSource) Store(_ string, _ string) error {
	return ErrNotSupported
}

func TestResolveReturnsFirstMatchInOrder(t *testing.T) {
	first := fakeSource{
		name: "source-first",
		values: map[string]string{
			"DEMO_TOKEN": "first-value",
		},
	}

	second := fakeSource{
		name: "source-second",
		values: map[string]string{
			"DEMO_TOKEN": "second-value",
		},
	}

	resolver := NewResolver(first, second)

	value, sourceName, found := resolver.Resolve("DEMO_TOKEN")
	if !found {
		t.Fatal("expected value to be found")
	}

	if value != "first-value" {
		t.Fatalf("expected first value, got %q", value)
	}

	if sourceName != "source-first" {
		t.Fatalf("expected source-first, got %q", sourceName)
	}
}

func TestResolveFallsBackToLaterSource(t *testing.T) {
	first := fakeSource{name: "source-first", values: map[string]string{}}
	second := fakeSource{
		name: "source-second",
		values: map[string]string{
			"DEMO_TOKEN": "second-value",
		},
	}

	resolver := NewResolver(first, second)

	value, sourceName, found := resolver.Resolve("DEMO_TOKEN")
	if !found {
		t.Fatal("expected value to be found")
	}

	if value != "second-value" {
		t.Fatalf("expected second value, got %q", value)
	}

	if sourceName != "source-second" {
		t.Fatalf("expected source-second, got %q", sourceName)
	}
}

func TestResolveReturnsNotFoundWhenMissingEverywhere(t *testing.T) {
	resolver := NewResolver(
		fakeSource{name: "source-a", values: map[string]string{}},
		fakeSource{name: "source-b", values: map[string]string{}},
	)

	value, sourceName, found := resolver.Resolve("DEMO_TOKEN")
	if found {
		t.Fatal("expected value to be missing")
	}

	if value != "" {
		t.Fatalf("expected empty value, got %q", value)
	}

	if sourceName != "" {
		t.Fatalf("expected empty source name, got %q", sourceName)
	}
}

func TestResolveSkipsNilSource(t *testing.T) {
	second := fakeSource{
		name: "source-second",
		values: map[string]string{
			"DEMO_TOKEN": "second-value",
		},
	}

	resolver := NewResolver(nil, second)

	value, sourceName, found := resolver.Resolve("DEMO_TOKEN")
	if !found {
		t.Fatal("expected value to be found")
	}

	if value != "second-value" {
		t.Fatalf("expected second value, got %q", value)
	}

	if sourceName != "source-second" {
		t.Fatalf("expected source-second, got %q", sourceName)
	}
}

func TestResolveReturnsNotFoundForEmptyEnvName(t *testing.T) {
	resolver := NewResolver(fakeSource{name: "source", values: map[string]string{"DEMO_TOKEN": "value"}})

	_, _, found := resolver.Resolve("   ")
	if found {
		t.Fatal("expected empty env name to be ignored")
	}
}

func TestResolveWorksWithNilResolver(t *testing.T) {
	var resolver *Resolver

	_, _, found := resolver.Resolve("DEMO_TOKEN")
	if found {
		t.Fatal("expected nil resolver to return not found")
	}
}

func TestStoreCanReturnErrNotSupported(t *testing.T) {
	source := fakeSource{name: "source", values: map[string]string{}}
	err := source.Store("DEMO_TOKEN", "value")

	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("expected ErrNotSupported, got %v", err)
	}
}
