package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderBreadcrumb_Empty(t *testing.T) {
	theme := NewTheme()
	assert.Equal(t, "", RenderBreadcrumb(theme, nil))
	assert.Equal(t, "", RenderBreadcrumb(theme, []BreadcrumbStep{}))
}

func TestRenderBreadcrumb_InvisibleStepsOmitted(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Source", Visible: false},
		{Label: "Service", Visible: false},
	})

	assert.Equal(t, "", result)
}

func TestRenderBreadcrumb_ActiveStep(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Service", Active: true, Visible: true},
	})

	assert.Contains(t, result, "Service")
}

func TestRenderBreadcrumb_CompletedWithValue(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Source", Value: "curated", Completed: true, Visible: true},
	})

	assert.Contains(t, result, "curated")
	assert.Contains(t, result, "\u2713")
	assert.NotContains(t, result, "Source")
}

func TestRenderBreadcrumb_CompletedWithoutValue(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Action", Completed: true, Visible: true},
	})

	assert.Contains(t, result, "Action")
	assert.Contains(t, result, "\u2713")
}

func TestRenderBreadcrumb_MixedStates(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Source", Value: "curated", Completed: true, Visible: true},
		{Label: "Service", Active: true, Visible: true},
		{Label: "Target", Visible: true},
	})

	assert.Contains(t, result, "curated")
	assert.Contains(t, result, "\u2713")
	assert.Contains(t, result, "Service")
	assert.Contains(t, result, "Target")
	assert.Contains(t, result, "\u203a")
}

func TestRenderBreadcrumb_ConditionalStepsHidden(t *testing.T) {
	theme := NewTheme()

	result := RenderBreadcrumb(theme, []BreadcrumbStep{
		{Label: "Source", Visible: false},
		{Label: "Service", Active: true, Visible: true},
		{Label: "Scope", Visible: false},
		{Label: "Target", Visible: true},
	})

	assert.NotContains(t, result, "Source")
	assert.Contains(t, result, "Service")
	assert.NotContains(t, result, "Scope")
	assert.Contains(t, result, "Target")
}
