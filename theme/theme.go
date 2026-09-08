// Package theme centralizes every color used by the TUI so they can be
// overridden from the user's config file. It has no dependency on the rest of
// the application, which keeps it importable from ui, ui/overlay, app and
// config without creating an import cycle.
package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
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

// Text colors picked when a color is used as a background. Not pure black and
// white: those read as harsh against the mid-tone fills of the palette.
const (
	darkText  = "#141414"
	lightText = "#FAFAFA"
)

// Contrast returns a text color readable on top of c, chosen per side so a
// light/dark pair still works on both terminal backgrounds.
//
// ANSI indices cannot be resolved to a luminance — the terminal's palette
// decides what "62" actually looks like — so they conservatively get light
// text, which suits the mid-to-dark tones an accent color usually is.
func (c Color) Contrast() Color {
	return Color{
		Light: contrastFor(c.Light),
		Dark:  contrastFor(c.Dark),
	}
}

func contrastFor(value string) string {
	r, g, b, ok := parseHex(value)
	if !ok {
		return lightText
	}
	// WCAG relative luminance. The 0.179 threshold is the point at which black
	// and white text have equal contrast ratio against the background.
	if relativeLuminance(r, g, b) > 0.179 {
		return darkText
	}
	return lightText
}

// parseHex accepts "#RGB" and "#RRGGBB", the two forms validate allows.
func parseHex(value string) (r, g, b uint8, ok bool) {
	if !hexPattern.MatchString(value) {
		return 0, 0, 0, false
	}
	digits := value[1:]
	if len(digits) == 3 {
		// Expand "#abc" to "#aabbcc".
		digits = string([]byte{
			digits[0], digits[0],
			digits[1], digits[1],
			digits[2], digits[2],
		})
	}
	var parsed [3]uint8
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(digits[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0, 0, 0, false
		}
		parsed[i] = uint8(v)
	}
	return parsed[0], parsed[1], parsed[2], true
}

// mutedBlend is how much of a color survives when it is used as a large
// surface rather than an accent. A full-width row painted in the raw color
// would overpower its own text and wreck the diff stats' green and red.
const mutedBlend = 0.30

// Muted returns a dimmed version of c, blended toward the terminal's own
// background, suitable as a fill behind text. It mirrors what the default
// theme already does by hand: an accent of #A78BFA with a #3B2F5C band.
//
// ANSI indices cannot be blended — there is no RGB to interpolate — so they
// are returned unchanged and will paint the surface at full strength.
func (c Color) Muted() Color {
	return Color{
		Light: blend(c.Light, lightText, mutedBlend),
		Dark:  blend(c.Dark, darkText, mutedBlend),
	}
}

// blend mixes fg over bg at the given alpha, both given as hex.
func blend(fg, bg string, alpha float64) string {
	fr, fgn, fb, ok := parseHex(fg)
	if !ok {
		return fg
	}
	br, bgn, bb, ok := parseHex(bg)
	if !ok {
		return fg
	}
	mix := func(a, b uint8) uint8 {
		return uint8(math.Round(alpha*float64(a) + (1-alpha)*float64(b)))
	}
	return fmt.Sprintf("#%02X%02X%02X", mix(fr, br), mix(fgn, bgn), mix(fb, bb))
}

func relativeLuminance(r, g, b uint8) float64 {
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
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

	// InstanceColors are the colors a session can be tagged with. Unlike the
	// fields above this is a list, and setting it replaces the defaults
	// wholesale rather than merging entry by entry.
	InstanceColors []NamedColor `json:"instance_colors,omitempty"`

	// BorderStyle is the line weight of the tab bar and the pane frame:
	// BorderRounded or BorderThick.
	BorderStyle string `json:"border_style,omitempty"`
}

// Line weights available for the tab bar and pane frame. Unicode box drawing
// has no heavy rounded corner, so thick borders are necessarily square.
const (
	BorderRounded = "rounded"
	BorderThick   = "thick"
)

func validBorderStyle(v string) bool {
	return v == BorderRounded || v == BorderThick
}

// NamedColor is one entry of the session color palette. The name is what gets
// stored in state.json, so re-theming a color keeps existing sessions tagged.
type NamedColor struct {
	Name  string `json:"name"`
	Color Color  `json:"color"`
}

// DefaultInstanceColors returns the colors offered when tagging a session.
func DefaultInstanceColors() []NamedColor {
	return []NamedColor{
		{"violet", pair("#6D28D9", "#A78BFA")},
		{"blue", pair("#1D4ED8", "#60A5FA")},
		{"cyan", pair("#0E7490", "#22D3EE")},
		{"green", pair("#15803D", "#4ADE80")},
		{"amber", pair("#B45309", "#FBBF24")},
		{"orange", pair("#C2410C", "#FB923C")},
		{"rose", pair("#BE123C", "#FB7185")},
		{"slate", pair("#475569", "#94A3B8")},
	}
}

// InstanceColor looks up a session color by name. The second return value is
// false for an unknown name or an untagged session, in which case callers fall
// back to the regular theme colors.
func (p Palette) InstanceColor(name string) (Color, bool) {
	if name == "" {
		return Color{}, false
	}
	for _, c := range p.InstanceColors {
		if c.Name == name {
			return c.Color, true
		}
	}
	return Color{}, false
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
		// InstanceColors is a list and is appended after the loop.
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

	if len(p.InstanceColors) > 0 {
		value, err := json.Marshal(p.InstanceColors)
		if err != nil {
			return nil, err
		}
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString(`"instance_colors":`)
		buf.Write(value)
	}

	if p.BorderStyle != "" {
		value, err := json.Marshal(p.BorderStyle)
		if err != nil {
			return nil, err
		}
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString(`"border_style":`)
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

		InstanceColors: DefaultInstanceColors(),
		BorderStyle:    BorderRounded,
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
		// InstanceColors is a list, not a Color, and is handled below.
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

	// A custom session palette replaces the defaults instead of merging: the
	// list is ordered and its length is meaningful, so a per-entry merge would
	// make it impossible to offer fewer colors than the default eight.
	if len(overrides.InstanceColors) > 0 {
		valid := make([]NamedColor, 0, len(overrides.InstanceColors))
		seen := make(map[string]bool, len(overrides.InstanceColors))
		for i, entry := range overrides.InstanceColors {
			switch {
			case entry.Name == "":
				errs = append(errs, fmt.Errorf("theme.instance_colors[%d]: name is required", i))
			case seen[entry.Name]:
				errs = append(errs, fmt.Errorf("theme.instance_colors[%d]: duplicate name %q", i, entry.Name))
			default:
				if err := entry.Color.validate(); err != nil {
					errs = append(errs, fmt.Errorf("theme.instance_colors[%d] (%s): %w", i, entry.Name, err))
					continue
				}
				seen[entry.Name] = true
				valid = append(valid, entry)
			}
		}
		if len(valid) > 0 {
			base.InstanceColors = valid
		}
	}

	if overrides.BorderStyle != "" {
		if validBorderStyle(overrides.BorderStyle) {
			base.BorderStyle = overrides.BorderStyle
		} else {
			errs = append(errs, fmt.Errorf("theme.border_style: %q is not %q or %q",
				overrides.BorderStyle, BorderRounded, BorderThick))
		}
	}

	return base, errs
}
