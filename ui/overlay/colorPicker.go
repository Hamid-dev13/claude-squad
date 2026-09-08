package overlay

import (
	"fmt"
	"strings"

	"claude-squad/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// noColorOption is the entry that clears a session's color tag.
const noColorOption = "No color"

// ColorPickerOverlay lets the user tag the selected session with one of the
// palette colors. It is a standalone modal, unlike the branch and profile
// pickers which are embedded in the new-session overlay.
type ColorPickerOverlay struct {
	// Title of the session being tagged, shown in the header.
	instanceTitle string
	// colors is the palette snapshot taken when the overlay opened.
	colors []theme.NamedColor
	// cursor indexes into the option list, where 0 is noColorOption.
	cursor int
	// Selected is the color name chosen on submit, empty for no color.
	Selected string
	// Submitted is true when the user confirmed, Canceled when they escaped.
	Submitted bool
	Canceled  bool
}

// NewColorPickerOverlay builds a picker for one session. current is the color
// name the session already carries, which the cursor starts on.
func NewColorPickerOverlay(instanceTitle, current string) *ColorPickerOverlay {
	colors := theme.Current().InstanceColors

	cursor := 0
	for i, c := range colors {
		if c.Name == current {
			cursor = i + 1 // offset by the "No color" entry
			break
		}
	}

	return &ColorPickerOverlay{
		instanceTitle: instanceTitle,
		colors:        colors,
		cursor:        cursor,
	}
}

// numOptions is the color count plus the "No color" entry.
func (c *ColorPickerOverlay) numOptions() int {
	return len(c.colors) + 1
}

// HandleKeyPress processes a key press. It returns true when the overlay
// should close, after setting Submitted or Canceled.
func (c *ColorPickerOverlay) HandleKeyPress(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyUp:
		if c.cursor > 0 {
			c.cursor--
		}
		return false
	case tea.KeyDown:
		if c.cursor < c.numOptions()-1 {
			c.cursor++
		}
		return false
	case tea.KeyEsc, tea.KeyCtrlC:
		c.Canceled = true
		return true
	case tea.KeyEnter:
		if c.cursor > 0 {
			c.Selected = c.colors[c.cursor-1].Name
		}
		c.Submitted = true
		return true
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "j":
			if c.cursor < c.numOptions()-1 {
				c.cursor++
			}
		case "q":
			c.Canceled = true
			return true
		}
		return false
	}
	return false
}

// Render draws the picker: one row per color, each prefixed with a swatch
// painted in that actual color so the choice is visible before confirming.
func (c *ColorPickerOverlay) Render() string {
	var s strings.Builder

	s.WriteString(tiTitleStyle.Render(fmt.Sprintf("Color for %q", c.instanceTitle)))
	s.WriteString("\n\n")

	for i := 0; i < c.numOptions(); i++ {
		var label, swatch string
		if i == 0 {
			label = noColorOption
			swatch = tiDividerStyle.Render("──")
		} else {
			entry := c.colors[i-1]
			label = entry.Name
			swatch = lipgloss.NewStyle().Foreground(entry.Color.Lip()).Render("██")
		}

		// The cursor marker sits outside the highlight so the swatch keeps its
		// own color rather than being repainted by the selection background.
		if i == c.cursor {
			s.WriteString(tiTitleStyle.Render("> "))
			s.WriteString(swatch)
			s.WriteString(" ")
			s.WriteString(ppSelectedStyle.Render(label))
		} else {
			s.WriteString("  ")
			s.WriteString(swatch)
			s.WriteString(" ")
			s.WriteString(tiButtonStyle.Render(label))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(tiDividerStyle.Render("↑/↓ move • enter confirm • esc cancel"))

	return tiStyle.Render(s.String())
}
