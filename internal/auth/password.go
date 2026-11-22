// Package auth hashes and verifies passwords with Argon2id.
//
// Hashes use the PHC string format, so the parameters travel with each hash:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>
//
// Old hashes keep verifying after the parameters are raised, and NeedsRehash
// tells the caller when to upgrade one after a successful login.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Password length limits, counted in bytes. The minimum follows NIST SP
// 800-63B's advice to require length rather than character classes. The
// maximum bounds the work an unauthenticated request can demand.
const (
	MinPasswordLength = 12
	MaxPasswordLength = 128
)

// ErrInvalidHash is returned when a stored hash is not a well-formed
// Argon2id PHC string.
var ErrInvalidHash = errors.New("auth: invalid password hash")

// Params are the Argon2id cost parameters.
type Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 // bytes
	KeyLength   uint32 // bytes
}

// DefaultParams follows the OWASP Password Storage Cheat Sheet's Argon2id
// baseline (19 MiB, 2 iterations, 1 thread). Raise Memory when the host can
// spare it; existing hashes are upgraded as their owners log in.
var DefaultParams = Params{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// Hasher hashes and verifies passwords with a fixed set of parameters.
type Hasher struct {
	params Params
	// decoy is verified against when no account matches, so a failed login
	// costs the same whether or not the email exists.
	decoy string
}

// NewHasher returns a Hasher that creates hashes with params.
func NewHasher(params Params) (*Hasher, error) {
	h := &Hasher{params: params}

	decoy, err := h.Hash(rand.Text())
	if err != nil {
		return nil, err
	}
	h.decoy = decoy

	return h, nil
}

// Hash returns the PHC-encoded Argon2id hash of password.
func (h *Hasher) Hash(password string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generate salt: %w", err)
	}

	key := argon2.IDKey(
		[]byte(password), salt,
		h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength,
	)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.Memory, h.params.Iterations, h.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether password matches encoded. It returns ErrInvalidHash
// when encoded cannot be parsed.
func (h *Hasher) Verify(password, encoded string) (bool, error) {
	params, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey(
		[]byte(password), salt,
		params.Iterations, params.Memory, params.Parallelism, params.KeyLength,
	)

	return subtle.ConstantTimeCompare(key, candidate) == 1, nil
}

// VerifyDecoy spends the same effort as Verify without a real account. Call
// it when a login names an unknown email so response time does not reveal
// which emails are registered.
func (h *Hasher) VerifyDecoy(password string) {
	_, _ = h.Verify(password, h.decoy)
}

// NeedsRehash reports whether encoded was created with parameters other than
// the Hasher's current ones.
func (h *Hasher) NeedsRehash(encoded string) bool {
	params, _, _, err := decode(encoded)
	if err != nil {
		return true
	}
	return params != h.params
}

func decode(encoded string) (Params, []byte, []byte, error) {
	// "$argon2id$v=19$m=19456,t=2,p=1$salt$key" splits into six parts, the
	// first of which is empty.
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var params Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if params.Memory == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return Params{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return Params{}, nil, nil, ErrInvalidHash
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return Params{}, nil, nil, ErrInvalidHash
	}

	params.SaltLength = uint32(len(salt)) //nolint:gosec // bounded by the stored hash's length
	params.KeyLength = uint32(len(key))   //nolint:gosec // bounded by the stored hash's length

	return params, salt, key, nil
}
