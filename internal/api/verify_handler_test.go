package api

import "testing"

func TestTurnstileHMAC_Deterministic(t *testing.T) {
	secret := "test-secret-key"
	got1 := turnstileHMAC(secret, 100, 200, 1700000000)
	got2 := turnstileHMAC(secret, 100, 200, 1700000000)
	if got1 != got2 {
		t.Fatalf("expected identical outputs; got %q and %q", got1, got2)
	}
	if len(got1) != 64 {
		t.Fatalf("expected 64-char hex SHA256; got len %d: %q", len(got1), got1)
	}
}

func TestTurnstileHMAC_DifferentInputs(t *testing.T) {
	secret := "test-secret-key"
	cases := []struct{ chat, user, exp int64 }{
		{100, 200, 1700000000},
		{100, 201, 1700000000},
		{101, 200, 1700000000},
		{100, 200, 1700000001},
	}
	seen := map[string]bool{}
	for _, c := range cases {
		h := turnstileHMAC(secret, c.chat, c.user, c.exp)
		if seen[h] {
			t.Fatalf("collision: chat=%d user=%d exp=%d produced duplicate HMAC %q", c.chat, c.user, c.exp, h)
		}
		seen[h] = true
	}
}

func TestTurnstileHMAC_DifferentSecrets(t *testing.T) {
	h1 := turnstileHMAC("secret-a", 1, 2, 3)
	h2 := turnstileHMAC("secret-b", 1, 2, 3)
	if h1 == h2 {
		t.Fatal("different secrets must produce different HMACs")
	}
}
