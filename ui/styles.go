package ui

import (
	"claude-squad/session"
	"claude-squad/theme"

	"github.com/charmbracelet/lipgloss"
)

// Every style used by the ui package lives here so a theme change only has to
// rebuild one file. refreshStyles runs once at init with the default palette,
// then again each time theme.Apply is called.
var (
	// Tab bar.
	inactiveTabStyle lipgloss.Style
	activeTabStyle   lipgloss.Style
	windowStyle      lipgloss.Style

	// Session list.
	readyStyle         lipgloss.Style
	addedLinesStyle    lipgloss.Style
	removedLinesStyle  lipgloss.Style
	pausedStyle        lipgloss.Style
	titleStyle         lipgloss.Style
	listDescStyle      lipgloss.Style
	selectedTitleStyle lipgloss.Style
	selectedDescStyle  lipgloss.Style
	mainTitle          lipgloss.Style
	autoYesStyle       lipgloss.Style

	// Menu bar.
	keyStyle         lipgloss.Style
	descStyle        lipgloss.Style
	sepStyle         lipgloss.Style
	actionGroupStyle lipgloss.Style
	menuStyle        lipgloss.Style

	// Diff tab. Exported because app renders diff stats too.
	AdditionStyle lipgloss.Style
	DeletionStyle lipgloss.Style
	HunkStyle     lipgloss.Style

	// Preview and terminal tabs.
	previewPaneStyle    lipgloss.Style
	pausedHintStyle     lipgloss.Style
	scrollFooterStyle   lipgloss.Style
	terminalPaneStyle   lipgloss.Style
	terminalFooterStyle lipgloss.Style

	// Error bar.
	errStyle lipgloss.Style

	// borders is the glyph set the tab strip and the pane frame share.
	borders tabBorders
)

// instanceColor resolves the color an instance is tagged with. The second
// return value is false for a nil or untagged instance, and for a tag naming a
// color the current theme no longer defines — callers then fall back to the
// regular theme colors rather than rendering something arbitrary.
func instanceColor(i *session.Instance) (lipgloss.TerminalColor, bool) {
	c, ok := instanceThemeColor(i)
	if !ok {
		return nil, false
	}
	return c.Lip(), true
}

// instanceThemeColor is instanceColor without the conversion, for callers that
// also need the matching text color.
func instanceThemeColor(i *session.Instance) (theme.Color, bool) {
	if i == nil {
		return theme.Color{}, false
	}
	return theme.Current().InstanceColor(i.Color)
}

func init() {
	theme.OnRefresh(refreshStyles)
}

func refreshStyles() {
	p := theme.Current()

	borders = roundedBorders()
	if p.BorderStyle == theme.BorderThick {
		borders = thickBorders()
	}

	inactiveTabStyle = lipgloss.NewStyle().
		Border(borders.inactive, true).
		BorderForeground(p.TabInactive.Lip()).
		AlignHorizontal(lipgloss.Center)
	activeTabStyle = lipgloss.NewStyle().
		Border(borders.active, true).
		BorderForeground(p.TabActive.Lip()).
		AlignHorizontal(lipgloss.Center)
	windowStyle = lipgloss.NewStyle().
		BorderForeground(p.TabBorder.Lip()).
		Border(borders.window, false, true, true, true)

	readyStyle = lipgloss.NewStyle().Foreground(p.StatusReady.Lip())
	addedLinesStyle = lipgloss.NewStyle().Foreground(p.StatAdded.Lip())
	removedLinesStyle = lipgloss.NewStyle().Foreground(p.StatRemoved.Lip())
	pausedStyle = lipgloss.NewStyle().Foreground(p.StatusPaused.Lip())
	// Horizontal padding is deliberately absent: the list renderer supplies the
	// left and right columns as separate blocks. Padding here would sit after
	// the nested resets of the status icon and diff stats, and so would render
	// unpainted — a gap at the row's edge.
	titleStyle = lipgloss.NewStyle().
		Padding(1, 0, 0, 0).
		Foreground(p.TextPrimary.Lip())
	listDescStyle = lipgloss.NewStyle().
		Padding(0, 0, 1, 0).
		Foreground(p.TextMuted.Lip())
	selectedTitleStyle = lipgloss.NewStyle().
		Padding(1, 0, 0, 0).
		Background(p.SelectionBg.Lip()).
		Foreground(p.SelectionFg.Lip())
	selectedDescStyle = lipgloss.NewStyle().
		Padding(0, 0, 1, 0).
		Background(p.SelectionBg.Lip()).
		Foreground(p.SelectionFg.Lip())
	mainTitle = lipgloss.NewStyle().
		Background(p.AppTitleBg.Lip()).
		Foreground(p.AppTitleFg.Lip())
	autoYesStyle = lipgloss.NewStyle().
		Background(p.SelectionBg.Lip()).
		Foreground(p.SelectionFg.Lip())

	keyStyle = lipgloss.NewStyle().Foreground(p.MenuKey.Lip())
	descStyle = lipgloss.NewStyle().Foreground(p.MenuDesc.Lip())
	sepStyle = lipgloss.NewStyle().Foreground(p.MenuSeparator.Lip())
	actionGroupStyle = lipgloss.NewStyle().Foreground(p.MenuActionGroup.Lip())
	menuStyle = lipgloss.NewStyle().Foreground(p.MenuText.Lip())

	AdditionStyle = lipgloss.NewStyle().Foreground(p.DiffAdded.Lip())
	DeletionStyle = lipgloss.NewStyle().Foreground(p.DiffRemoved.Lip())
	HunkStyle = lipgloss.NewStyle().Foreground(p.DiffHunk.Lip())

	previewPaneStyle = lipgloss.NewStyle().Foreground(p.TextPrimary.Lip())
	pausedHintStyle = lipgloss.NewStyle().Foreground(p.PausedHint.Lip())
	scrollFooterStyle = lipgloss.NewStyle().Foreground(p.TextFooter.Lip())
	terminalPaneStyle = lipgloss.NewStyle().Foreground(p.TextPrimary.Lip())
	terminalFooterStyle = lipgloss.NewStyle().Foreground(p.TextFooter.Lip())

	errStyle = lipgloss.NewStyle().Foreground(p.Error.Lip())
}
