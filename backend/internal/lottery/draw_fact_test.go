package lottery

import "testing"

func TestCanonicalDrawHashIgnoresIngestSource(t *testing.T) {
	a := CanonicalDrawHash("hash_ffc_3s", "900-A", []string{"1", "2", "3"})
	b := CanonicalDrawHash(" hash_ffc_3s ", " 900-A ", []string{"1", "2", "3"})
	if a == "" || a != b {
		t.Fatalf("hashes differ: %q %q", a, b)
	}
}

func TestCanonicalDrawHashChangesForCorrection(t *testing.T) {
	a := CanonicalDrawHash("hash_ffc_3s", "900-A", []string{"1", "2", "3"})
	b := CanonicalDrawHash("hash_ffc_3s", "900-A", []string{"1", "2", "4"})
	if a == b {
		t.Fatal("corrected balls must have a distinct draw hash")
	}
}
