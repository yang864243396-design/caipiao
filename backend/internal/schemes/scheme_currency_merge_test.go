package schemes

import (
	"encoding/json"
	"testing"
)

func TestMergeUpdateKeepsSchemeCurrencyTRX(t *testing.T) {
	existing := []byte(`{"runTypeId":"fixed_number","schemeGroups":["1,2"]}`)
	raw := map[string]json.RawMessage{
		"schemeCurrency": json.RawMessage(`"TRX"`),
	}
	patch, err := ParseUpdatePatch(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !patch.HasSchemeCurrency || patch.SchemeCurrency != "TRX" {
		t.Fatalf("patch=%+v", patch)
	}
	out, err := mergeUpdateDefinitionConfig(existing, patch, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := schemeCurrencyFromConfig(out)
	if got != "TRX" {
		t.Fatalf("got %q want TRX; config=%s", got, string(out))
	}
}

func TestAddToCloudPatchSchemeCurrency(t *testing.T) {
	existing := []byte(`{"runTypeId":"fixed_number"}`)
	out, err := mergeDefinitionConfig(existing, AddToCloudConfigPatch{SchemeCurrency: "TRX"})
	if err != nil {
		t.Fatal(err)
	}
	if got := schemeCurrencyFromConfig(out); got != "TRX" {
		t.Fatalf("got %q; config=%s", got, string(out))
	}
}
