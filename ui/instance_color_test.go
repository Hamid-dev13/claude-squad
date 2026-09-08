package ui

import (
	"os"
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

// The color bar is a column the row reserves whether or not it is tagged, so
// tagging cannot change the row's width. If that regresses the list overflows
// its pane, which is hard to spot in a screenshot but obvious here.
func TestInstanceColorDoesNotChangeRowWidth(t *testing.T) {
	r := newTestRenderer()

	for _, selected := range []bool{false, true} {
		plain := r.Render(newTestInstance(t, "session", ""), 1, selected, false)
		tagged := r.Render(newTestInstance(t, "session", "amber"), 1, selected, false)

		assert.Equal(t, renderedWidths(plain), renderedWidths(tagged),
			"a colored row must occupy exactly the same width (selected=%v)", selected)
	}
}

// sgrColor extracts the first truecolor directive with the given introducer
// ("48;2;" for a background, "38;2;" for a foreground) from a rendered string.
func sgrColor(rendered, introducer string) string {
	idx := strings.Index(rendered, introducer)
	if idx < 0 {
		return ""
	}
	// Stop at the terminator, or the trailing reset ends up inside the fields.
	rest := rendered[idx:]
	if end := strings.IndexByte(rest, 'm'); end >= 0 {
		rest = rest[:end]
	}
	fields := strings.Split(rest, ";")
	if len(fields) < 5 {
		return ""
	}
	return strings.Join(fields[:5], ";")
}

// bgSequence asks lipgloss what it actually emits for a color, rather than
// formatting the hex ourselves: its conversion is not byte-exact, and #dde4f0
// comes out as 221;227;240 rather than the 221;228;240 the hex implies.
func bgSequence(t *testing.T, c theme.Color) string {
	t.Helper()
	seq := sgrColor(lipgloss.NewStyle().Background(c.Lip()).Render(" "), "48;2;")
	require.NotEmpty(t, seq, "the probe should carry a background")
	return seq
}

func fgSequence(t *testing.T, c theme.Color) string {
	t.Helper()
	seq := sgrColor(lipgloss.NewStyle().Foreground(c.Lip()).Render("x"), "38;2;")
	require.NotEmpty(t, seq, "the probe should carry a foreground")
	return seq
}

// cellBackgrounds walks a rendered line and returns the background in effect
// for each printable cell, as the SGR fragment that set it. An empty string
// means the terminal's own background.
//
// This exists because a styled block nested inside a styled row ends with a
// reset that silently drops the row's background for everything after it. The
// symptom is a black gap mid-row, which no amount of substring matching on the
// rendered output reliably catches.
func cellBackgrounds(line string) []string {
	var (
		cells   []string
		current string
		i       int
	)
	runes := []rune(line)
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			end := i + 2
			for end < len(runes) && runes[end] != 'm' {
				end++
			}
			if end >= len(runes) {
				break
			}
			params := string(runes[i+2 : end])
			switch {
			case params == "" || params == "0":
				current = ""
			case strings.Contains(params, "49"):
				current = ""
			default:
				if idx := strings.Index(params, "48;2;"); idx >= 0 {
					rest := params[idx:]
					// Keep only the five fields of the background directive.
					fields := strings.Split(rest, ";")
					if len(fields) >= 5 {
						current = strings.Join(fields[:5], ";")
					}
				}
			}
			i = end + 1
			continue
		}
		cells = append(cells, current)
		i++
	}
	return cells
}

// The selected row's highlight band takes the session's color, dimmed, and it
// must run edge to edge. The raw color is reserved for the leftmost column.
func TestSelectedRowBandIsUnbroken(t *testing.T) {
	amber, ok := theme.Default().InstanceColor("amber")
	require.True(t, ok)

	raw := bgSequence(t, amber)
	muted := bgSequence(t, amber.Muted())

	r := newTestRenderer()
	rendered := r.Render(newTestInstance(t, "session", "amber"), 1, true, false)

	for n, line := range strings.Split(rendered, "\n") {
		cells := cellBackgrounds(line)
		require.NotEmpty(t, cells, "line %d is empty", n)

		assert.Equal(t, raw, cells[0], "line %d: the first column is the color bar", n)
		for col, bg := range cells[1:] {
			assert.Equal(t, muted, bg,
				"line %d, column %d: the band must not break — this is the black gap bug", n, col+1)
		}
	}
}

// An untagged but selected row must be just as unbroken, on the theme's band.
func TestSelectedUntaggedRowBandIsUnbroken(t *testing.T) {
	band := bgSequence(t, theme.Default().SelectionBg)

	r := newTestRenderer()
	rendered := r.Render(newTestInstance(t, "session", ""), 1, true, false)

	for n, line := range strings.Split(rendered, "\n") {
		for col, bg := range cellBackgrounds(line) {
			assert.Equal(t, band, bg, "line %d, column %d", n, col)
		}
	}
}

// An unselected tagged row carries the color bar and nothing else: the band is
// what distinguishes the selected row, and tinting every row would lose it.
func TestUnselectedTaggedRowShowsOnlyTheColorBar(t *testing.T) {
	amber, ok := theme.Default().InstanceColor("amber")
	require.True(t, ok)
	raw := bgSequence(t, amber)

	r := newTestRenderer()
	rendered := r.Render(newTestInstance(t, "session", "amber"), 1, false, false)

	for n, line := range strings.Split(rendered, "\n") {
		cells := cellBackgrounds(line)
		require.NotEmpty(t, cells, "line %d is empty", n)

		assert.Equal(t, raw, cells[0], "line %d: the bar runs the full height", n)
		for col, bg := range cells[1:] {
			assert.Empty(t, bg,
				"line %d, column %d: the rest of an unselected row is unpainted", n, col+1)
		}
	}
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

	assert.Contains(t, rendered, bgSequence(t, amber),
		"the active tab should carry the color as a background, not just a border")
	assert.Contains(t, rendered, fgSequence(t, amber.Contrast()),
		"the label should use the contrasting text color")
}

// Inactive tabs must not be filled, otherwise there is nothing left to tell
// the active tab apart.
func TestInactiveTabsKeepThemeColor(t *testing.T) {
	themed, ok := theme.Default().InstanceColor("amber")
	require.True(t, ok)
	fill := bgSequence(t, themed)

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
