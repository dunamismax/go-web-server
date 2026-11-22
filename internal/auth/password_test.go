package auth

import (
	"errors"
	"strings"
	"testing"
)

// testParams keep the suite fast; production uses DefaultParams.
var testParams = Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func newTestHasher(t *testing.T, params Params) *Hasher {
	t.Helper()

	h, err := NewHasher(params)
	if err != nil {
		t.Fatalf("NewHasher: %v", err)
	}
	return h
}

func TestHashAndVerify(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t, testParams)

	hash, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("hash is not PHC-encoded with the configured parameters: %q", hash)
	}

	ok, err := h.Verify("correct horse battery staple", hash)
	if err != nil || !ok {
		t.Fatalf("Verify(correct) = %v, %v; want true, nil", ok, err)
	}

	ok, err = h.Verify("wrong password", hash)
	if err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestHashIsSalted(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t, testParams)

	first, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	second, err := h.Hash("same password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if first == second {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestVerifyUsesParametersFromHash(t *testing.T) {
	t.Parallel()

	older := newTestHasher(t, testParams)
	hash, err := older.Hash("a long enough password")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	stronger := testParams
	stronger.Memory = 128
	newer := newTestHasher(t, stronger)

	ok, err := newer.Verify("a long enough password", hash)
	if err != nil || !ok {
		t.Fatalf("Verify with newer hasher = %v, %v; want true, nil", ok, err)
	}
	if !newer.NeedsRehash(hash) {
		t.Fatal("NeedsRehash = false for a hash made with weaker parameters")
	}
	if older.NeedsRehash(hash) {
		t.Fatal("NeedsRehash = true for a hash made with the current parameters")
	}
}

func TestVerifyRejectsMalformedHashes(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t, testParams)

	malformed := []string{
		"",
		"plaintext",
		"$2a$10$abcdefghijklmnopqrstuv",          // bcrypt
		"$argon2i$v=19$m=64,t=1,p=1$c2FsdA$a2V5", // wrong variant
		"$argon2id$v=18$m=64,t=1,p=1$c2FsdA$a2V5",       // wrong version
		"$argon2id$v=19$m=0,t=1,p=1$c2FsdA$a2V5",        // zero memory
		"$argon2id$v=19$m=64,t=1,p=1$not base64!$a2V5",  // bad salt
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdA$",           // empty key
		"$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5$extra", // too many parts
	}

	for _, encoded := range malformed {
		if _, err := h.Verify("password", encoded); !errors.Is(err, ErrInvalidHash) {
			t.Errorf("Verify(%q) error = %v; want ErrInvalidHash", encoded, err)
		}
		if !h.NeedsRehash(encoded) {
			t.Errorf("NeedsRehash(%q) = false; want true", encoded)
		}
	}
}

func TestVerifyDecoyDoesNotPanic(t *testing.T) {
	t.Parallel()

	newTestHasher(t, testParams).VerifyDecoy("anything")
}
