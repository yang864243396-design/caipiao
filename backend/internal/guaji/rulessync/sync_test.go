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
	if plan.SubPlays[0].Label != "直选复式" || plan.SubPlays[0].OutboundPlayCode != "1" {
		t.Fatalf("first sub: %+v", plan.SubPlays[0])
	}
}
