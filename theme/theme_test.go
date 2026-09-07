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
