// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"os"
	"sort"

	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/crypto/argon2"
)

// Format 1 deliberately fixes every KDF/AEAD parameter. Untrusted envelopes
// cannot select an excessive work factor or downgrade the algorithms.
type vaultHeader struct {
	Version     int      `json:"version"`
	WorkspaceID string   `json:"workspace_id"`
	KDF         string   `json:"kdf"`
	MemoryKiB   uint32   `json:"memory_kib"`
	Iterations  uint32   `json:"iterations"`
	Lanes       uint8    `json:"lanes"`
	Cipher      string   `json:"cipher"`
	Salt        []byte   `json:"salt"`
	IDs         []string `json:"ids"`
}
type vaultEnvelope struct {
	vaultHeader
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func (m *Manager) parseEnvelope(raw []byte) (vaultEnvelope, error) {
	var env vaultEnvelope
	if strict(raw, &env) != nil || env.Version != 1 || env.WorkspaceID != m.workspaceID || env.KDF != "argon2id-v19" || env.MemoryKiB != 64*1024 || env.Iterations != 3 || env.Lanes != 4 || env.Cipher != "aes-256-gcm" || len(env.Salt) != 16 || len(env.Nonce) != 12 || len(env.Ciphertext) < 16 || len(env.IDs) > maxCredentials {
		return env, contracts.Fail("unavailable")
	}
	prev := ""
	for _, id := range env.IDs {
		if !contracts.ValidID(id) || id <= prev {
			return env, contracts.Fail("unavailable")
		}
		prev = id
	}
	return env, nil
}
func (m *Manager) vaultCipher(salt []byte) (cipher.AEAD, error) {
	if len(m.passphrase) == 0 {
		return nil, contracts.Fail("unavailable")
	}
	if len(m.key) == 0 || !bytes.Equal(m.salt, salt) {
		wipe(m.key)
		m.key = argon2.IDKey(m.passphrase, salt, 3, 64*1024, 4, 32)
		m.salt = append(m.salt[:0], salt...)
	}
	block, e := aes.NewCipher(m.key)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	aead, e := cipher.NewGCM(block)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	return aead, nil
}
func (m *Manager) readVault() (map[string][]byte, error) {
	if len(m.passphrase) == 0 {
		return nil, contracts.Fail("unavailable")
	}
	raw, e := readFile(m.root, "vault.json")
	if os.IsNotExist(e) {
		return map[string][]byte{}, nil
	}
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	env, e := m.parseEnvelope(raw)
	if e != nil {
		return nil, e
	}
	aead, e := m.vaultCipher(env.Salt)
	if e != nil {
		return nil, e
	}
	aad, _ := json.Marshal(env.vaultHeader)
	plain, e := aead.Open(nil, env.Nonce, env.Ciphertext, aad)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer wipe(plain)
	var values map[string][]byte
	if strict(plain, &values) != nil || len(values) != len(env.IDs) {
		wipeMap(values)
		return nil, contracts.Fail("unavailable")
	}
	for _, id := range env.IDs {
		if !validValue(id, values[id]) {
			wipeMap(values)
			return nil, contracts.Fail("unavailable")
		}
	}
	return values, nil
}
func (m *Manager) saveVault(values map[string][]byte) error {
	salt := m.salt
	if len(salt) == 0 {
		salt = make([]byte, 16)
		if _, e := rand.Read(salt); e != nil {
			return contracts.Fail("unavailable")
		}
	}
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	header := vaultHeader{1, m.workspaceID, "argon2id-v19", 64 * 1024, 3, 4, "aes-256-gcm", salt, ids}
	aead, e := m.vaultCipher(salt)
	if e != nil {
		return e
	}
	nonce := make([]byte, aead.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return contracts.Fail("unavailable")
	}
	plain, e := json.Marshal(values)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer wipe(plain)
	aad, _ := json.Marshal(header)
	raw, e := json.Marshal(vaultEnvelope{header, nonce, aead.Seal(nil, nonce, plain, aad)})
	if e != nil {
		return contracts.Fail("unavailable")
	}
	return atomicWrite(m.root, "vault.json", raw)
}
