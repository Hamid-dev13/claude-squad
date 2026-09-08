package session

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceColorSurvivesSerialization(t *testing.T) {
	inst, err := NewInstance(InstanceOptions{Title: "tagged", Path: ".", Program: "echo"})
	require.NoError(t, err)
	inst.Color = "amber"

	data := inst.ToInstanceData()
	assert.Equal(t, "amber", data.Color)

	encoded, err := json.Marshal(data)
	require.NoError(t, err)

	var back InstanceData
	require.NoError(t, json.Unmarshal(encoded, &back))
	assert.Equal(t, "amber", back.Color)
}

func TestUntaggedInstanceOmitsColorKey(t *testing.T) {
	inst, err := NewInstance(InstanceOptions{Title: "plain", Path: ".", Program: "echo"})
	require.NoError(t, err)

	encoded, err := json.Marshal(inst.ToInstanceData())
	require.NoError(t, err)

	assert.NotContains(t, string(encoded), `"color"`,
		"an untagged session must not add a key to state.json")
}

// state.json files written before session colors existed have no "color" key.
// They must still load, leaving the instance untagged.
func TestInstanceDataWithoutColorLoads(t *testing.T) {
	legacy := `{
		"title": "old-session",
		"path": "/tmp/repo",
		"branch": "hamid/old-session",
		"status": 1,
		"height": 0,
		"width": 0,
		"created_at": "2026-08-01T10:00:00Z",
		"updated_at": "2026-08-01T10:00:00Z",
		"auto_yes": false,
		"program": "claude"
	}`

	var data InstanceData
	require.NoError(t, json.Unmarshal([]byte(legacy), &data))

	assert.Equal(t, "old-session", data.Title)
	assert.Empty(t, data.Color, "a session predating colors is simply untagged")
}
