package toolloop

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/magomedcoder/coder-server/pkg/domain"
	"github.com/magomedcoder/coder-server/pkg/logger"
)

func parametersJSONByName(genParams *domain.GenerationParams, resolvedToolName string) string {
	if genParams == nil {
		return ""
	}

	want := strings.TrimSpace(resolvedToolName)
	if want == "" {
		return ""
	}

	for _, t := range genParams.Tools {
		if strings.TrimSpace(t.Name) == want {
			return strings.TrimSpace(t.ParametersJSON)
		}
	}

	return ""
}

func TopLevelAllowedPropertyNames(parametersJSON string) (allowed map[string]struct{}, strict bool) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(parametersJSON), &root); err != nil {
		return nil, false
	}

	if t, ok := root["type"]; ok {
		var typeStr string
		if err := json.Unmarshal(t, &typeStr); err != nil {
			return nil, false
		}

		if typeStr != "" && typeStr != "object" {
			return nil, false
		}
	}

	propsRaw, ok := root["properties"]
	if !ok {
		return nil, false
	}

	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsRaw, &props); err != nil || len(props) == 0 {
		return nil, false
	}

	apRaw, ok := root["additionalProperties"]
	if !ok {
		return nil, false
	}

	var apBool bool
	if err := json.Unmarshal(apRaw, &apBool); err != nil || apBool {
		return nil, false
	}

	out := make(map[string]struct{}, len(props))
	for k := range props {
		out[k] = struct{}{}
	}

	return out, true
}

func PruneJSONArgsToSchema(params json.RawMessage, parametersJSON string, resolvedToolName string) (json.RawMessage, []string) {
	allowed, strict := TopLevelAllowedPropertyNames(parametersJSON)
	if !strict || len(allowed) == 0 {
		return params, nil
	}

	if len(params) == 0 {
		return params, nil
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil || m == nil {
		if err != nil {
			logger.W("Tool: phase=args_prune_skip tool=%q err=%v (аргументы не JSON-объект)", resolvedToolName, err)
		} else {
			logger.V("Tool: phase=args_prune_skip tool=%q (аргументы null)", resolvedToolName)
		}

		return params, nil
	}

	dropped := make([]string, 0)
	for k := range m {
		if _, ok := allowed[k]; !ok {
			dropped = append(dropped, k)
			delete(m, k)
		}
	}

	if len(dropped) == 0 {
		return params, nil
	}

	out, err := json.Marshal(m)
	if err != nil {
		return params, nil
	}

	sort.Strings(dropped)

	return out, dropped
}

func MaybePruneArgsJSON(genParams *domain.GenerationParams, resolvedToolName string, params json.RawMessage) json.RawMessage {
	pj := parametersJSONByName(genParams, resolvedToolName)
	if pj == "" {
		return params
	}

	pruned, dropped := PruneJSONArgsToSchema(params, pj, resolvedToolName)
	if len(dropped) > 0 {
		logger.I("Tool: phase=args_pruned tool=%q dropped_keys=%v", resolvedToolName, dropped)
	}

	return pruned
}
