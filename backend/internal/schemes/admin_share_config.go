package schemes

import (
	"encoding/json"
	"strings"
)

type AdminShareConfigExtra struct {
	MultCoeff     string
	BetMultiplier map[string]interface{}
}

func mergeAdminShareSnapshotConfig(baseCfg []byte, patch AddToCloudConfigPatch, extra AdminShareConfigExtra) ([]byte, error) {
	return mergeAdminShareSnapshotConfigUpdate(baseCfg, nil, patch, extra)
}

func mergeAdminShareSnapshotConfigUpdate(existing, structural []byte, patch AddToCloudConfigPatch, extra AdminShareConfigExtra) ([]byte, error) {
	cfgBytes, err := mergeDefinitionConfig(existing, patch)
	if err != nil {
		return nil, err
	}
	cfg := map[string]interface{}{}
	if len(cfgBytes) > 0 {
		_ = json.Unmarshal(cfgBytes, &cfg)
	}
	if len(structural) > 0 {
		structMap := map[string]interface{}{}
		_ = json.Unmarshal(structural, &structMap)
		for k, v := range structMap {
			cfg[k] = v
		}
	}
	if strings.TrimSpace(patch.StartTime) == "" && strings.TrimSpace(patch.EndTime) == "" {
		delete(cfg, "startTime")
		delete(cfg, "endTime")
	}
	if mc := strings.TrimSpace(extra.MultCoeff); mc != "" {
		cfg["multCoeff"] = mc
	}
	if extra.BetMultiplier != nil {
		cfg["betMultiplier"] = extra.BetMultiplier
		if rounds := compileBetMultiplierRounds(extra.BetMultiplier, cfg); len(rounds) > 0 {
			cfg["rounds"] = rounds
		}
	}
	normalizeSchemeConfigBetFields(cfg)
	return json.Marshal(cfg)
}
