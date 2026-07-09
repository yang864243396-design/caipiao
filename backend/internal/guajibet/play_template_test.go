package guajibet

import "testing"

func TestIsSSCPlayTemplate(t *testing.T) {
	for _, tpl := range []string{"", "ssc_std", "fast_ssc_std"} {
		if !IsSSCPlayTemplate(tpl) {
			t.Fatalf("want true for %q", tpl)
		}
	}
	if IsSSCPlayTemplate("pk10_std") {
		t.Fatal("pk10_std should be false")
	}
}
