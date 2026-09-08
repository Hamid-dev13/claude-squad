package theme

import (
	"encoding/json"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColorUnmarshalShorthand(t *testing.T) {
	var c Color
	require.NoError(t, json.Unmarshal([]byte(`"#7D56F4"`), &c))
	assert.Equal(t, Color{Light: "#7D56F4", Dark: "#7D56F4"}, c)
}

func TestColorUnmarshalPair(t *testing.T) {
	var c Color
	require.NoError(t, json.Unmarshal([]byte(`{"light":"#874BFD","dark":"#7D56F4"}`), &c))
	assert.Equal(t, Color{Light: "#874BFD", Dark: "#7D56F4"}, c)
}

func TestColorUnmarshalSingleSideMirrors(t *testing.T) {
	var c Color
	require.NoError(t, json.Unmarshal([]byte(`{"dark":"#7D56F4"}`), &c))
	assert.Equal(t, Color{Light: "#7D56F4", Dark: "#7D56F4"}, c,
		"a lone dark value should also apply to light backgrounds")
}

func TestColorRoundTrip(t *testing.T) {
	for _, c := range []Color{mono("62"), pair("#874BFD", "#7D56F4")} {
		data, err := json.Marshal(c)
		require.NoError(t, err)

		var back Color
		require.NoError(t, json.Unmarshal(data, &back))
		assert.Equal(t, c, back)
	}
}

func TestColorValidate(t *testing.T) {
	valid := []string{"#fff", "#FFFFFF", "#7d56f4", "0", "62", "255"}
	for _, v := range valid {
		assert.NoError(t, mono(v).validate(), "%q should be accepted", v)
	}

	invalid := []string{"", "purple", "#ff", "#fffff", "256", "-1", "0x62", "62 "}
	for _, v := range invalid {
		assert.Error(t, mono(v).validate(), "%q should be rejected", v)
	}
}

func TestColorLipIsAdaptive(t *testing.T) {
	assert.Equal(t,
		lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"},
		pair("#874BFD", "#7D56F4").Lip())
}

func TestMergeNilKeepsDefaults(t *testing.T) {
	merged, errs := Merge(Default(), nil)
	assert.Empty(t, errs)
	assert.Equal(t, Default(), merged)
}

func TestMergeOverridesOnlyProvidedKeys(t *testing.T) {
	merged, errs := Merge(Default(), &Palette{TabActive: mono("#ff0000")})
	require.Empty(t, errs)

	assert.Equal(t, mono("#ff0000"), merged.TabActive, "the provided key should win")
	assert.Equal(t, Default().TabInactive, merged.TabInactive, "untouched keys keep their default")
	assert.Equal(t, Default().HelpDesc, merged.HelpDesc)
}

func TestMergeRejectsInvalidColorAndKeepsDefault(t *testing.T) {
	merged, errs := Merge(Default(), &Palette{
		TabActive:   mono("chartreuse"),
		TabInactive: mono("#00ff00"),
	})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "theme.tab_active",
		"the error should name the offending config key")
	assert.Equal(t, Default().TabActive, merged.TabActive, "an invalid color must not be applied")
	assert.Equal(t, mono("#00ff00"), merged.TabInactive, "valid siblings still apply")
}

func TestPaletteUnmarshalFromConfigFragment(t *testing.T) {
	var p Palette
	require.NoError(t, json.Unmarshal([]byte(`{
		"tab_active":   "#ff8800",
		"tab_inactive": {"light": "#cccccc", "dark": "#444444"}
	}`), &p))

	merged, errs := Merge(Default(), &p)
	require.Empty(t, errs)
	assert.Equal(t, mono("#ff8800"), merged.TabActive)
	assert.Equal(t, pair("#cccccc", "#444444"), merged.TabInactive)
	assert.Equal(t, Default().TabBorder, merged.TabBorder)
}

func TestDefaultPaletteIsFullyPopulatedAndValid(t *testing.T) {
	// Guards against adding a Palette field without a default: an empty color
	// would render as the terminal's default and silently break the theme.
	var p Palette
	data, err := json.Marshal(Default())
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &p))

	merged, errs := Merge(Palette{}, &p)
	assert.Empty(t, errs, "every default color must be valid")
	assert.Equal(t, Default(), merged, "every Palette field must have a default")
}

func TestPaletteMarshalOmitsUnsetColors(t *testing.T) {
	data, err := json.Marshal(Palette{TabActive: mono("#ff8800")})
	require.NoError(t, err)
	assert.JSONEq(t, `{"tab_active":"#ff8800"}`, string(data),
		"unset colors must not be written back to config.json")
}

func TestContrastPicksReadableText(t *testing.T) {
	for _, tc := range []struct {
		background string
		want       string
		why        string
	}{
		{"#FFFFFF", darkText, "white needs dark text"},
		{"#000000", lightText, "black needs light text"},
		{"#FBBF24", darkText, "amber is bright"},
		{"#1D4ED8", lightText, "a deep blue is dark"},
		{"#fff", darkText, "the short hex form is expanded"},
		{"#4ADE80", darkText, "green reads as bright: luminance is green-weighted"},
		{"62", lightText, "an ANSI index cannot be measured, so assume a dark accent"},
	} {
		got := contrastFor(tc.background)
		assert.Equal(t, tc.want, got, "%s (%s)", tc.background, tc.why)
	}
}

func TestContrastHandlesEachSideSeparately(t *testing.T) {
	// A pair whose two sides need opposite text colors.
	c := pair("#FBBF24", "#1D4ED8").Contrast()
	assert.Equal(t, darkText, c.Light)
	assert.Equal(t, lightText, c.Dark)
}

func TestContrastIsAlwaysValid(t *testing.T) {
	// Whatever it returns has to be a color the rest of the theme accepts.
	for _, entry := range DefaultInstanceColors() {
		assert.NoError(t, entry.Color.Contrast().validate(), "entry %q", entry.Name)
	}
	assert.NoError(t, mono("nonsense").Contrast().validate(),
		"even an unparseable background must yield a usable text color")
}

func TestMutedDimsTowardTheTerminalBackground(t *testing.T) {
	m := pair("#B45309", "#FBBF24").Muted()

	assert.Equal(t, "#594719", m.Dark, "on a dark terminal the band darkens")
	assert.Equal(t, "#E5C8B2", m.Light, "on a light terminal it pales instead")
}

// The muted band must stay far enough from the text drawn on it. The theme's
// own hand-picked pair (#A78BFA accent, #3B2F5C band) is the reference for how
// dim it should be.
func TestMutedIsCloseToTheHandPickedBand(t *testing.T) {
	m := pair("#874BFD", "#A78BFA").Muted()

	r, g, b, ok := parseHex(m.Dark)
	require.True(t, ok)
	refR, refG, refB, ok := parseHex("#3B2F5C")
	require.True(t, ok)

	for _, d := range []int{
		int(r) - int(refR), int(g) - int(refG), int(b) - int(refB),
	} {
		assert.LessOrEqual(t, abs(d), 16,
			"each channel should land within 16 of the reference band, got %s", m.Dark)
	}
}

func TestMutedIsDarkEnoughForLightText(t *testing.T) {
	for _, entry := range DefaultInstanceColors() {
		band := entry.Color.Muted()
		assert.NoError(t, band.validate(), "entry %q", entry.Name)
		assert.Equal(t, lightText, contrastFor(band.Dark),
			"the dark-terminal band for %q must still take light text", entry.Name)
		assert.Equal(t, darkText, contrastFor(band.Light),
			"the light-terminal band for %q must still take dark text", entry.Name)
	}
}

func TestMutedLeavesAnsiIndicesAlone(t *testing.T) {
	// There is no RGB to interpolate, so the value passes through untouched
	// rather than being silently mangled into an invalid color.
	assert.Equal(t, mono("62"), mono("62").Muted())
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestParseHex(t *testing.T) {
	r, g, b, ok := parseHex("#FBBF24")
	require.True(t, ok)
	assert.Equal(t, [3]uint8{0xFB, 0xBF, 0x24}, [3]uint8{r, g, b})

	r, g, b, ok = parseHex("#abc")
	require.True(t, ok)
	assert.Equal(t, [3]uint8{0xAA, 0xBB, 0xCC}, [3]uint8{r, g, b},
		"the short form expands each digit")

	_, _, _, ok = parseHex("62")
	assert.False(t, ok, "an ANSI index is not a hex color")
}

func TestDefaultInstanceColorsAreValidAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for i, entry := range DefaultInstanceColors() {
		assert.NotEmpty(t, entry.Name, "entry %d has no name", i)
		assert.False(t, seen[entry.Name], "duplicate name %q", entry.Name)
		assert.NoError(t, entry.Color.validate(), "entry %q", entry.Name)
		seen[entry.Name] = true
	}
	assert.Len(t, seen, len(DefaultInstanceColors()))
}

func TestInstanceColorLookup(t *testing.T) {
	p := Default()

	c, ok := p.InstanceColor("amber")
	require.True(t, ok)
	assert.Equal(t, pair("#B45309", "#FBBF24"), c)

	_, ok = p.InstanceColor("chartreuse")
	assert.False(t, ok, "an unknown name must not resolve")

	_, ok = p.InstanceColor("")
	assert.False(t, ok, "an untagged instance must not resolve")
}

func TestMergeReplacesInstanceColorsWholesale(t *testing.T) {
	merged, errs := Merge(Default(), &Palette{
		InstanceColors: []NamedColor{
			{"hot", mono("#ff0000")},
			{"cold", mono("#0000ff")},
		},
	})
	require.Empty(t, errs)

	// Replacement, not a merge: offering fewer colors than the default eight
	// has to be possible.
	require.Len(t, merged.InstanceColors, 2)
	_, ok := merged.InstanceColor("amber")
	assert.False(t, ok, "a default color must be gone once the list is overridden")
	c, ok := merged.InstanceColor("hot")
	require.True(t, ok)
	assert.Equal(t, mono("#ff0000"), c)
}

func TestMergeInstanceColorsRejectsBadEntries(t *testing.T) {
	merged, errs := Merge(Default(), &Palette{
		InstanceColors: []NamedColor{
			{"good", mono("#00ff00")},
			{"", mono("#00ff00")},
			{"bad", mono("not-a-color")},
			{"good", mono("#123456")},
		},
	})

	require.Len(t, errs, 3)
	assert.Contains(t, errs[0].Error(), "name is required")
	assert.Contains(t, errs[1].Error(), "bad")
	assert.Contains(t, errs[2].Error(), "duplicate")

	require.Len(t, merged.InstanceColors, 1, "only the valid entry survives")
	assert.Equal(t, "good", merged.InstanceColors[0].Name)
}

func TestMergeKeepsDefaultsWhenEveryInstanceColorIsInvalid(t *testing.T) {
	merged, errs := Merge(Default(), &Palette{
		InstanceColors: []NamedColor{{"bad", mono("nope")}},
	})

	require.Len(t, errs, 1)
	assert.Equal(t, Default().InstanceColors, merged.InstanceColors,
		"an entirely unusable list must leave the defaults in place rather than "+
			"leaving no color to pick from")
}

func TestPaletteMarshalIncludesInstanceColors(t *testing.T) {
	data, err := json.Marshal(Palette{
		TabActive:      mono("#ff8800"),
		InstanceColors: []NamedColor{{"hot", mono("#ff0000")}},
	})
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"tab_active":"#ff8800","instance_colors":[{"name":"hot","color":"#ff0000"}]}`,
		string(data))
}

func TestPaletteUnmarshalInstanceColors(t *testing.T) {
	var p Palette
	require.NoError(t, json.Unmarshal([]byte(`{
		"instance_colors": [
			{"name": "hot",  "color": "#ff0000"},
			{"name": "cool", "color": {"light": "#1D4ED8", "dark": "#60A5FA"}}
		]
	}`), &p))

	require.Len(t, p.InstanceColors, 2)
	assert.Equal(t, mono("#ff0000"), p.InstanceColors[0].Color)
	assert.Equal(t, pair("#1D4ED8", "#60A5FA"), p.InstanceColors[1].Color)
}

func TestApplyRunsRegisteredHooks(t *testing.T) {
	t.Cleanup(func() {
		mu.Lock()
		hooks = nil
		current = Default()
		mu.Unlock()
	})

	mu.Lock()
	hooks = nil
	mu.Unlock()

	var seen []Color
	OnRefresh(func() { seen = append(seen, Current().TabActive) })
	require.Len(t, seen, 1, "OnRefresh should fire once on registration")
	assert.Equal(t, Default().TabActive, seen[0])

	errs := Apply(&Palette{TabActive: mono("#123456")})
	assert.Empty(t, errs)
	require.Len(t, seen, 2, "Apply should fire the hook again")
	assert.Equal(t, mono("#123456"), seen[1])
	assert.Equal(t, mono("#123456"), Current().TabActive)
}
