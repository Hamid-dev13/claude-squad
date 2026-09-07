// Package theme centralizes every color used by the TUI so they can be
// overridden from the user's config file. It has no dependency on the rest of
// the application, which keeps it importable from ui, ui/overlay, app and
// config without creating an import cycle.
package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Color is a themeable color. In JSON it accepts either a single value applied
// to both terminal backgrounds:
//
//	"tab_active": "#7D56F4"
//
// or a light/dark pair:
//
//	"tab_active": {"light": "#874BFD", "dark": "#7D56F4"}
//
// Values are hex ("#RGB" or "#RRGGBB") or ANSI 256 indices ("0" to "255").
type Color struct {
	Light string `json:"light,omitempty"`
	Dark  string `json:"dark,omitempty"`
}

var (
	hexPattern  = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	ansiPattern = regexp.MustCompile(`^(?:[0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
)

func mono(v string) Color      { return Color{Light: v, Dark: v} }
func pair(l, d string) Color   { return Color{Light: l, Dark: d} }
func (c Color) IsEmpty() bool  { return c.Light == "" && c.Dark == "" }
func (c Color) String() string { return c.Light + "/" + c.Dark }

// UnmarshalJSON accepts both the shorthand string form and the light/dark object.
func (c *Color) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" {
		return nil
	}
	if strings.HasPrefix(trimmed, `"`) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*c = mono(v)
		return nil
	}
	// Alias avoids recursing into this method.
	type colorAlias Color
	var alias colorAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*c = Color(alias)
	// A single side is enough: the other one falls back to it.
	if c.Light == "" {
		c.Light = c.Dark
	}
	if c.Dark == "" {
		c.Dark = c.Light
	}
	return nil
}

// MarshalJSON collapses identical light/dark values back to the shorthand form.
func (c Color) MarshalJSON() ([]byte, error) {
	if c.Light == c.Dark {
		return json.Marshal(c.Light)
	}
	type colorAlias Color
	return json.Marshal(colorAlias(c))
}

// validate reports whether both sides are colors lipgloss can render.
func (c Color) validate() error {
	for _, v := range []string{c.Light, c.Dark} {
		if !hexPattern.MatchString(v) && !ansiPattern.MatchString(v) {
			return fmt.Errorf("%q is neither a hex color (#RGB, #RRGGBB) nor an ANSI index (0-255)", v)
		}
	}
	return nil
}

// Lip converts the color to something lipgloss styles accept.
func (c Color) Lip() lipgloss.TerminalColor {
	return lipgloss.AdaptiveColor{Light: c.Light, Dark: c.Dark}
}

// Palette holds every themeable color in the application. Each field maps to
// one key of the "theme" object in ~/.claude-squad/config.json.
type Palette struct {
	// Tab bar at the top of the right-hand pane.
	TabActive   Color `json:"tab_active"`
	TabInactive Color `json:"tab_inactive"`
	TabBorder   Color `json:"tab_border"`

	// Session list on the left.
	StatusReady  Color `json:"status_ready"`
	StatusPaused Color `json:"status_paused"`
	StatAdded    Color `json:"stat_added"`
	StatRemoved  Color `json:"stat_removed"`
	TextPrimary  Color `json:"text_primary"`
	TextMuted    Color `json:"text_muted"`
	TextFooter   Color `json:"text_footer"`
	SelectionBg  Color `json:"selection_bg"`
	SelectionFg  Color `json:"selection_fg"`
	AppTitleBg   Color `json:"app_title_bg"`
	AppTitleFg   Color `json:"app_title_fg"`

	// Bottom menu bar.
	MenuKey         Color `json:"menu_key"`
	MenuDesc        Color `json:"menu_desc"`
	MenuSeparator   Color `json:"menu_separator"`
	MenuActionGroup Color `json:"menu_action_group"`
	MenuText        Color `json:"menu_text"`

	// Diff tab.
	DiffAdded   Color `json:"diff_added"`
	DiffRemoved Color `json:"diff_removed"`
	DiffHunk    Color `json:"diff_hunk"`

	// Misc feedback.
	Error      Color `json:"error"`
	PausedHint Color `json:"paused_hint"`

	// Overlays: text input, branch picker, profile picker, confirmations.
	OverlayAccent   Color `json:"overlay_accent"`
	OverlayAccentFg Color `json:"overlay_accent_fg"`
	OverlayText     Color `json:"overlay_text"`
	OverlayDim      Color `json:"overlay_dim"`
	OverlayShadow   Color `json:"overlay_shadow"`
	ConfirmBorder   Color `json:"confirm_border"`

	// Help screen.
	HelpTitle  Color `json:"help_title"`
	HelpHeader Color `json:"help_header"`
	HelpKey    Color `json:"help_key"`
	HelpDesc   Color `json:"help_desc"`
}

// MarshalJSON writes only the colors that are actually set, in declaration
// order. Without it, saving a config with a partial theme would fill the file
// with dozens of empty keys.
func (p Palette) MarshalJSON() ([]byte, error) {
	val := reflect.ValueOf(p)
	typ := val.Type()

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i := 0; i < typ.NumField(); i++ {
		color, ok := val.Field(i).Interface().(Color)
		if !ok || color.IsEmpty() {
			continue
		}
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(strings.Split(typ.Field(i).Tag.Get("json"), ",")[0])
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(color)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')

	return buf.Bytes(), nil
}

// Default returns the palette shipped with claude-squad. Leaving the "theme"
// key out of the config file reproduces exactly this.
func Default() Palette {
	return Palette{
		TabActive:   pair("#874BFD", "#7D56F4"),
		TabInactive: pair("#874BFD", "#7D56F4"),
		TabBorder:   pair("#874BFD", "#7D56F4"),

		StatusReady:  mono("#51bd73"),
		StatusPaused: mono("#888888"),
		StatAdded:    mono("#51bd73"),
		StatRemoved:  mono("#de613e"),
		TextPrimary:  pair("#1a1a1a", "#dddddd"),
		TextMuted:    pair("#A49FA5", "#777777"),
		TextFooter:   mono("#808080"),
		SelectionBg:  mono("#dde4f0"),
		SelectionFg:  mono("#1a1a1a"),
		AppTitleBg:   mono("62"),
		AppTitleFg:   mono("230"),

		MenuKey:         pair("#655F5F", "#7F7A7A"),
		MenuDesc:        pair("#7A7474", "#9C9494"),
		MenuSeparator:   pair("#DDDADA", "#3C3C3C"),
		MenuActionGroup: mono("99"),
		MenuText:        mono("205"),

		DiffAdded:   mono("#22c55e"),
		DiffRemoved: mono("#ef4444"),
		DiffHunk:    mono("#0ea5e9"),

		Error:      mono("#FF0000"),
		PausedHint: mono("#FFD700"),

		OverlayAccent:   mono("62"),
		OverlayAccentFg: mono("0"),
		OverlayText:     mono("7"),
		OverlayDim:      mono("240"),
		OverlayShadow:   mono("#333333"),
		ConfirmBorder:   mono("#de613e"),

		HelpTitle:  mono("#7D56F4"),
		HelpHeader: mono("#36CFC9"),
		HelpKey:    mono("#FFCC00"),
		HelpDesc:   mono("#FFFFFF"),
	}
}

var (
	mu      sync.RWMutex
	current = Default()
	hooks   []func()
)

// Current returns the palette in effect.
func Current() Palette {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// OnRefresh registers a callback that rebuilds a package's styles. The callback
// runs immediately with the default palette, then again on every Apply. UI
// packages call this from init() so their style variables are never nil.
func OnRefresh(f func()) {
	mu.Lock()
	hooks = append(hooks, f)
	mu.Unlock()
	f()
}

// Apply merges the user's overrides on top of the defaults and rebuilds every
// registered style. Invalid or unset entries keep their default, and each
// rejected key is returned so the caller can warn about it. Passing nil resets
// to the defaults.
func Apply(overrides *Palette) []error {
	merged, errs := Merge(Default(), overrides)

	mu.Lock()
	current = merged
	registered := make([]func(), len(hooks))
	copy(registered, hooks)
	mu.Unlock()

	for _, f := range registered {
		f()
	}
	return errs
}

// Merge overlays the non-empty, valid fields of overrides onto base. It is
// exported mainly so it can be tested without touching global state.
func Merge(base Palette, overrides *Palette) (Palette, []error) {
	if overrides == nil {
		return base, nil
	}

	var errs []error
	baseVal := reflect.ValueOf(&base).Elem()
	overVal := reflect.ValueOf(overrides).Elem()
	typ := baseVal.Type()

	for i := 0; i < typ.NumField(); i++ {
		over, ok := overVal.Field(i).Interface().(Color)
		if !ok || over.IsEmpty() {
			continue
		}
		key := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if err := over.validate(); err != nil {
			errs = append(errs, fmt.Errorf("theme.%s: %w", key, err))
			continue
		}
		baseVal.Field(i).Set(reflect.ValueOf(over))
	}
	return base, errs
}
