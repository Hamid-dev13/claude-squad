package ui

import (
	"strings"
	"testing"

	"claude-squad/theme"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderPane(t *testing.T, borderStyle string) string {
	t.Helper()
	t.Cleanup(func() { theme.Apply(nil) })

	require.Empty(t, theme.Apply(&theme.Palette{BorderStyle: borderStyle}))

	w := NewTabbedWindow(NewPreviewPane(), NewDiffPane(), NewTerminalPane())
	w.SetSize(58, 12)
	w.SetInstance(newTestInstance(t, "session", ""))
	return w.String()
}

// A mixed-weight frame is the failure mode worth guarding: a light ┴ landing on
// a heavy ┃ leaves a visible notch at the seam. Each set must be self-contained.
func TestBorderStyleGlyphsAreConsistent(t *testing.T) {
	light := "─│╭╮╰╯┴├┤┘└"
	heavy := "━┃┏┓┗┛┻┣┫"

	t.Run("rounded uses only light glyphs", func(t *testing.T) {
		rendered := renderPane(t, theme.BorderRounded)
		for _, r := range heavy {
			assert.NotContains(t, rendered, string(r),
				"a heavy glyph has no matching corner in the rounded set")
		}
		assert.True(t, strings.ContainsAny(rendered, light))
	})

	t.Run("thick uses only heavy glyphs", func(t *testing.T) {
		rendered := renderPane(t, theme.BorderThick)
		for _, r := range light {
			assert.NotContains(t, rendered, string(r),
				"a light glyph would leave a notch against the heavy frame")
		}
		assert.True(t, strings.ContainsAny(rendered, heavy))
	})
}

func TestBorderStyleChangesTheFrame(t *testing.T) {
	rounded := renderPane(t, theme.BorderRounded)
	thick := renderPane(t, theme.BorderThick)

	assert.NotEqual(t, rounded, thick)
	assert.Equal(t, renderedWidths(rounded), renderedWidths(thick),
		"the weight must not change the pane's geometry")
}

// Rounded corners only survive if the border background is left unpainted.
func TestRoundedCornersSurviveATaggedInstance(t *testing.T) {
	t.Cleanup(func() { theme.Apply(nil) })
	require.Empty(t, theme.Apply(&theme.Palette{BorderStyle: theme.BorderRounded}))

	w := NewTabbedWindow(NewPreviewPane(), NewDiffPane(), NewTerminalPane())
	w.SetSize(58, 12)
	w.SetInstance(newTestInstance(t, "session", "amber"))

	assert.Contains(t, w.String(), "╭", "the tag must not square off the tab")
}
