package rulessync

import "testing"

func TestBuildPlan_sscFirstGroup(t *testing.T) {
	tpl := RulesTemplate{
		Name: "时时彩",
		Groups: []RulesGroup{{
			Name: "前三码",
			Team: []RulesTeam{{
				Name: "前三直选",
				Rule: []RulesRule{{
					ID: "1", Name: "直选复式", FullName: "前三直选复式", Active: true,
				}, {
					ID: "2", Name: "直选单式", FullName: "前三直选单式", Active: true,
				}},
			}},
		}},
	}
	plan, err := BuildPlan("ssc_std", "1", tpl)
	if err != nil {
		t.Fatal(err)
	}
	if plan.RulesTypeName != "时时彩" {
		t.Fatalf("rules type name = %q", plan.RulesTypeName)
	}
	if len(plan.PlayTypes) != 1 || plan.PlayTypes[0].Label != "前三码" {
		t.Fatalf("play type: %+v", plan.PlayTypes)
	}
	if len(plan.SubPlays) != 2 {
		t.Fatalf("sub plays = %d", len(plan.SubPlays))
	}
	if plan.SubPlays[0].Label != "前三直选复式" || plan.SubPlays[0].OutboundPlayCode != "1" {
		t.Fatalf("first sub: %+v", plan.SubPlays[0])
	}
	if plan.SubPlays[1].Label != "前三直选单式" {
		t.Fatalf("second sub: %+v", plan.SubPlays[1])
	}
}

func TestBuildPlan_usesFullNameForAllSubs(t *testing.T) {
	tpl := RulesTemplate{
		Name: "时时彩",
		Groups: []RulesGroup{{
			Name: "任选",
			Team: []RulesTeam{
				{
					Name: "任选二",
					Rule: []RulesRule{{
						ID: "74", Name: "直选复式", FullName: "任二直选复式", Active: true,
					}, {
						ID: "75", Name: "直选单式", FullName: "任二直选单式", Active: true,
					}},
				},
				{
					Name: "任选三",
					Rule: []RulesRule{{
						ID: "80", Name: "直选复式", FullName: "任三直选复式", Active: true,
					}, {
						ID: "81", Name: "直选单式", FullName: "任三直选单式", Active: true,
					}},
				},
			},
		}},
	}
	plan, err := BuildPlan("ssc_std", "1", tpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.SubPlays) != 4 {
		t.Fatalf("sub plays = %d", len(plan.SubPlays))
	}
	want := map[string]string{
		"74": "任二直选复式",
		"75": "任二直选单式",
		"80": "任三直选复式",
		"81": "任三直选单式",
	}
	for _, sp := range plan.SubPlays {
		if got := sp.Label; got != want[sp.SubID] {
			t.Fatalf("sub %s label = %q, want %q", sp.SubID, got, want[sp.SubID])
		}
	}
}

func TestBuildPlan_fallsBackToShortNameWhenFullNameEmpty(t *testing.T) {
	tpl := RulesTemplate{
		Name: "时时彩",
		Groups: []RulesGroup{{
			Name: "前三码",
			Team: []RulesTeam{{
				Name: "前三直选",
				Rule: []RulesRule{{
					ID: "1", Name: "直选复式", FullName: "", Active: true,
				}},
			}},
		}},
	}
	plan, err := BuildPlan("ssc_std", "1", tpl)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SubPlays[0].Label != "直选复式" {
		t.Fatalf("fallback label = %q", plan.SubPlays[0].Label)
	}
}
