package overlay

import (
	"claude-squad/theme"

	"github.com/charmbracelet/lipgloss"
)

// Styles shared by every overlay. Like the ui package, they are rebuilt
// whenever the theme changes.
var (
	// Text input overlay.
	tiStyle              lipgloss.Style
	tiTitleStyle         lipgloss.Style
	tiButtonStyle        lipgloss.Style
	tiFocusedButtonStyle lipgloss.Style
	tiDividerStyle       lipgloss.Style

	// Branch picker.
	bpLabelStyle    lipgloss.Style
	bpFilterStyle   lipgloss.Style
	bpSelectedStyle lipgloss.Style
	bpDimStyle      lipgloss.Style

	// Profile picker.
	ppLabelStyle    lipgloss.Style
	ppSelectedStyle lipgloss.Style
	ppDimStyle      lipgloss.Style

	// Plain text overlay and drop shadow.
	textOverlayStyle lipgloss.Style
	shadowStyle      lipgloss.Style

	// Default border color of confirmation dialogs.
	confirmBorderColor lipgloss.TerminalColor
)

func init() {
	theme.OnRefresh(refreshStyles)
}

func refreshStyles() {
	p := theme.Current()

	tiStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.OverlayAccent.Lip()).
		Padding(1, 2)
	tiTitleStyle = lipgloss.NewStyle().
		Foreground(p.OverlayAccent.Lip()).
		Bold(true).
		MarginBottom(1)
	tiButtonStyle = lipgloss.NewStyle().
		Foreground(p.OverlayText.Lip())
	tiFocusedButtonStyle = lipgloss.NewStyle().
		Background(p.OverlayAccent.Lip()).
		Foreground(p.OverlayAccentFg.Lip())
	tiDividerStyle = lipgloss.NewStyle().
		Foreground(p.OverlayDim.Lip())

	bpLabelStyle = lipgloss.NewStyle().
		Foreground(p.OverlayAccent.Lip()).
		Bold(true)
	bpFilterStyle = lipgloss.NewStyle().
		Foreground(p.OverlayText.Lip())
	bpSelectedStyle = lipgloss.NewStyle().
		Background(p.OverlayAccent.Lip()).
		Foreground(p.OverlayAccentFg.Lip())
	bpDimStyle = lipgloss.NewStyle().
		Foreground(p.OverlayDim.Lip())

	ppLabelStyle = lipgloss.NewStyle().
		Foreground(p.OverlayAccent.Lip()).
		Bold(true)
	ppSelectedStyle = lipgloss.NewStyle().
		Background(p.OverlayAccent.Lip()).
		Foreground(p.OverlayAccentFg.Lip())
	ppDimStyle = lipgloss.NewStyle().
		Foreground(p.OverlayDim.Lip())

	textOverlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.OverlayAccent.Lip()).
		Padding(1, 2)
	shadowStyle = lipgloss.NewStyle().
		Foreground(p.OverlayShadow.Lip())

	confirmBorderColor = p.ConfirmBorder.Lip()
}
