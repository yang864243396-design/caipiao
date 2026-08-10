package schemes

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func schemeMultiplierFromConfig(cfgBytes []byte) float64 {
	var cfg map[string]interface{}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		return 1
	}
	return schemeMultiplierFromValue(cfg["multCoeff"])
}

func schemeMultiplierFromValue(raw interface{}) float64 {
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" || value == "<nil>" {
		return 1
	}
	multiplier, err := strconv.ParseFloat(value, 64)
	if err != nil || multiplier < 1 || math.Mod(multiplier, 1) != 0 {
		return 1
	}
	return multiplier
}

func setSchemeConfigMultiplier(cfgBytes []byte, multiplier float64) ([]byte, error) {
	var cfg map[string]interface{}
	if len(strings.TrimSpace(string(cfgBytes))) > 0 {
		if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	cfg["multCoeff"] = strconv.FormatInt(int64(multiplier), 10)
	return json.Marshal(cfg)
}
