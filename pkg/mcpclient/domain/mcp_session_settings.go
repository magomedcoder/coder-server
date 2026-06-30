package domain

import (
	"encoding/json"
)

type MCPSessionSettingsWire struct {
	Enabled   bool    `json:"enabled"`
	ServerIDs []int64 `json:"server_ids"`
}

var DefaultMCPSessionSettingsJSON = []byte(`{"enabled":false,"server_ids":[]}`)

func DefaultMCPSessionSettingsJSONCopy() []byte {
	return append([]byte(nil), DefaultMCPSessionSettingsJSON...)
}

func MarshalMCPSessionSettings(enabled bool, serverIDs []int64) ([]byte, error) {
	if serverIDs == nil {
		serverIDs = []int64{}
	}

	return json.Marshal(MCPSessionSettingsWire{
		Enabled:   enabled,
		ServerIDs: serverIDs,
	})
}

func UnmarshalMCPSessionSettings(raw []byte) (enabled bool, serverIDs []int64, err error) {
	if len(raw) == 0 {
		return false, []int64{}, nil
	}

	var w MCPSessionSettingsWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return false, nil, err
	}

	if w.ServerIDs == nil {
		w.ServerIDs = []int64{}
	}

	return w.Enabled, w.ServerIDs, nil
}
