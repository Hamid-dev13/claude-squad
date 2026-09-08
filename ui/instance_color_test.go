package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"claude-squad/session"
	"claude-squad/theme"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain forces a color-capable profile. `go test` runs without a TTY, so
// lipgloss would otherwise strip every escape sequence and the color
// assertions below would pass vacuously against identical plain text.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	os.Exit(m.Run())
}

func newTestInstance(t *testing.T, title, color string) *session.Instance {
	t.Helper()
	inst, err := session.NewInstance(session.InstanceOptions{
		Title:   title,
		Path:    ".",
		Program: "echo",
	})
	require.NoError(t, err)
	inst.Color = color
	return inst
}

func newTestRenderer() *InstanceRenderer {
	s := spinner.New()
	r := &InstanceRenderer{spinner: &s}
	r.setWidth(80)
	return r
}

// renderedWidths returns the printable width of each line, ignoring the ANSI
// sequences that carry the colors.
func renderedWidths(rendered string) []int {
	lines := strings.Split(rendered, "\n")
	widths := make([]int, len(lines))
	for i, line := range lines {
		widths[i] = ansi.PrintableRuneWidth(line)
	}
	return widths
}

// The marker takes over the prefix's leading space rather than being prepended,
// precisely so the row keeps its width. If that ever regresses the list will
// overflow its pane, which is hard to spot in a screenshot but obvious here.
func TestInstanceColorDoesNotChangeRowWidth(t *testing.T) {
	r := newTestRenderer()

	for _, selected := range []bool{false, true} {
		plain := r.Render(newTestInstance(t, "session", ""), 1, selected, false)
		tagged := r.Render(newTestInstance(t, "session", "amber"), 1, selected, false)

		assert.Equal(t, renderedWidths(plain), renderedWidths(tagged),
			"a colored row must occupy exactly the same width (selected=%v)", selected)
	}
}

func TestInstanceColorRendersMarker(t *testing.T) {
	r := newTestRenderer()

	plain := r.Render(newTestInstance(t, "session", ""), 1, false, false)
	tagged := r.Render(newTestInstance(t, "session", "amber"), 1, false, false)

	assert.NotContains(t, plain, instanceMarker, "an untagged row has no marker")
	assert.Equal(t, 2, strings.Count(tagged, instanceMarker),
		"the marker should appear once per line of the row")
}

func TestUnknownInstanceColorFallsBackToTheme(t *testing.T) {
	r := newTestRenderer()

	plain := r.Render(newTestInstance(t, "session", ""), 1, false, false)
	// A name the palette does not define, e.g. left over from a theme that
	// used to declare it.
	unknown := r.Render(newTestInstance(t, "session", "chartreuse"), 1, false, false)

	assert.Equal(t, plain, unknown,
		"an unresolvable color name must render exactly like an untagged instance")
}

func TestInstanceColorLookup(t *testing.T) {
	c, ok := instanceColor(newTestInstance(t, "session", "amber"))
	require.True(t, ok)
	expected, found := theme.Default().InstanceColor("amber")
	require.True(t, found)
	assert.Equal(t, expected.Lip(), c)

	_, ok = instanceColor(newTestInstance(t, "session", ""))
	assert.False(t, ok, "an untagged instance has no color")

	_, ok = instanceColor(nil)
	assert.False(t, ok, "a nil instance must not panic")
}

// The tab bar takes the active instance's color; an untagged instance leaves
// the theme colors alone.
func TestTabbedWindowUsesInstanceColor(t *testing.T) {
	newWindow := func(inst *session.Instance) string {
		w := NewTabbedWindow(NewPreviewPane(), NewDiffPane(), NewTerminalPane())
		w.SetSize(60, 20)
		w.SetInstance(inst)
		return w.String()
	}

	plain := newWindow(newTestInstance(t, "session", ""))
	tagged := newWindow(newTestInstance(t, "session", "amber"))

	require.NotEmpty(t, plain)
	assert.NotEqual(t, plain, tagged, "a tagged instance must recolor the tab bar")
	assert.Equal(t, renderedWidths(plain), renderedWidths(tagged),
		"recoloring must not change the pane's geometry")
}

// The active tab is filled with the color rather than merely outlined, and the
// label switches to a text color readable on that fill.
func TestActiveTabIsFilledWithInstanceColor(t *testing.T) {
	amber, ok := theme.Default().InstanceColor("amber")
	require.True(t, ok)

	w := NewTabbedWindow(NewPreviewPane(), NewDiffPane(), NewTerminalPane())
	w.SetSize(60, 20)
	w.SetInstance(newTestInstance(t, "session", "amber"))
	rendered := w.String()

	// termenv emits truecolor backgrounds as "48;2;R;G;B".
	r, g, b, ok := hexToRGB(amber.Dark)
	require.True(t, ok)
	assert.Contains(t, rendered, fmt.Sprintf("48;2;%d;%d;%d", r, g, b),
		"the active tab should carry the color as a background, not just a border")

	tr, tg, tb, ok := hexToRGB(amber.Contrast().Dark)
	require.True(t, ok)
	assert.Contains(t, rendered, fmt.Sprintf("38;2;%d;%d;%d", tr, tg, tb),
		"the label should use the contrasting text color")
}

// Inactive tabs must not be filled, otherwise there is nothing left to tell
// the active tab apart.
func TestInactiveTabsKeepThemeColor(t *testing.T) {
	themed, ok := theme.Default().InstanceColor("amber")
	require.True(t, ok)
	r, g, b, _ := hexToRGB(themed.Dark)
	fill := fmt.Sprintf("48;2;%d;%d;%d", r, g, b)

	w := NewTabbedWindow(NewPreviewPane(), NewDiffPane(), NewTerminalPane())
	w.SetSize(60, 20)
	w.SetInstance(newTestInstance(t, "session", "amber"))

	tabRow := strings.Split(w.String(), "\n")[3]
	assert.Equal(t, 1, strings.Count(tabRow, "Preview"))
	assert.NotContains(t,
		tabRow[strings.Index(tabRow, "Diff"):],
		fill,
		"the fill must stop at the active tab")
}

func hexToRGB(v string) (r, g, b uint8, ok bool) {
	if len(v) != 7 || v[0] != '#' {
		return 0, 0, 0, false
	}
	var parsed [3]uint64
	for i := 0; i < 3; i++ {
		n, err := strconv.ParseUint(v[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return 0, 0, 0, false
		}
		parsed[i] = n
	}
	return uint8(parsed[0]), uint8(parsed[1]), uint8(parsed[2]), true
}
