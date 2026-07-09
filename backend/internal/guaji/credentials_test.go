package guaji

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, err := CredentialsKey("test-dev-key-for-guaji-credentials", "")
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := EncryptSecret(key, "secret-password-123")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := DecryptSecret(key, cipher)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "secret-password-123" {
		t.Fatalf("got %q", plain)
	}
}
