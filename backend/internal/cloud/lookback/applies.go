package lookback



// AppliesTo 判断实例 simBet 通道是否启用回头评估。

func AppliesTo(settings Settings, simBet bool) bool {

	if settings.Judgment == JudgmentNone {

		return false

	}

	if simBet {

		return settings.ApplySim

	}

	return settings.ApplyFormal

}



// SyncApplyFlagsFromRunModes 从旧 runModes 字段同步 applyFormal/applySim。

func SyncApplyFlagsFromRunModes(s *Settings) {

	if s == nil {

		return

	}

	if len(s.RunModes) == 0 {

		return

	}

	for _, m := range s.RunModes {

		switch m {

		case RunModeReal:

			s.ApplyFormal = true

		case RunModeSim:

			s.ApplySim = true

		}

	}

}



// SyncRunModesFromApplyFlags 从 applyFormal/applySim 同步旧 runModes 字段（API 兼容）。

func SyncRunModesFromApplyFlags(s *Settings) {

	if s == nil {

		return

	}

	var modes []RunMode

	if s.ApplyFormal {

		modes = append(modes, RunModeReal)

	}

	if s.ApplySim {

		modes = append(modes, RunModeSim)

	}

	s.RunModes = NormalizeRunModes(modes)

}

