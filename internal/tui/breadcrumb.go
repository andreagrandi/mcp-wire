package tui

import "strings"

// BreadcrumbStep represents one step in the wizard breadcrumb.
type BreadcrumbStep struct {
	Label     string // step name shown when active or future (e.g., "Service")
	Value     string // shown instead of Label when completed (e.g., "sentry")
	Active    bool
	Completed bool
	Visible   bool
}

// RenderBreadcrumb renders the breadcrumb bar from a list of steps.
//
// Completed steps show their Value (or Label if Value is empty) in green
// with a check mark. The active step is bold cyan. Future steps are dim.
// Invisible steps are omitted entirely.
func RenderBreadcrumb(theme Theme, steps []BreadcrumbStep) string {
	var parts []string

	for _, step := range steps {
		if !step.Visible {
			continue
		}

		if step.Completed {
			display := step.Label
			if step.Value != "" {
				display = step.Value
			}

			parts = append(parts, theme.Completed.Render(display+" \u2713"))
		} else if step.Active {
			parts = append(parts, theme.Active.Render(step.Label))
		} else {
			parts = append(parts, theme.Dim.Render(step.Label))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	sep := theme.BreadSep.Render(" \u203a ")
	return strings.Join(parts, sep)
}
