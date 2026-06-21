package srtp

// E2eSrtpKeys holds the per-participant session keys for the end-to-end 1:1 SRTP
// cipher: the AES-128 cipher key, the 14-byte master salt, and the auth key.
type E2eSrtpKeys struct {
	CipherKey [16]byte
	Salt      [14]byte
	AuthKey   [20]byte
}

// aesCmKdf is the AES-CM PRF (libsrtp KDF): IV = master salt with label XORed into
// byte 7, zero-padded to 16, then AES-128-CTR keystream over len zero bytes.
func aesCmKdf(masterKey, masterSalt []byte, label byte, length int) []byte {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L24-L32
	// TODO
	// agent suggestion: iv := make([]byte,16); copy(iv, masterSalt[:14]); iv[7] ^= label;
	// out := make([]byte, length); aes.NewCipher(masterKey) -> cipher.NewCTR(block, iv) ->
	// XORKeyStream(out, out). Panic on aes.NewCipher err (16-byte key invariant).
	// human input:
	return nil
}

// deriveSessionKeysFromMaster splits the 46-byte master into key (16) + salt (14)
// and runs the AES-CM PRF three times (labels 0x00/0x01/0x02) for cipher/auth/salt.
func deriveSessionKeysFromMaster(master []byte) E2eSrtpKeys {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L34-L49
	// TODO
	// agent suggestion: masterKey=master[0:16], masterSalt=master[16:30]; copy
	// aesCmKdf(...,0x00,16) into CipherKey, 0x01,20 into AuthKey, 0x02,14 into Salt.
	// human input:
	return E2eSrtpKeys{}
}

// DeriveE2eKeys derives the E2E 1:1 keys from callKey (>=32B) using participantLid
// as the HKDF info. The bool is false when callKey is shorter than 32 bytes.
func DeriveE2eKeys(callKey []byte, participantLid string) (E2eSrtpKeys, bool) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L55-L61
	// TODO
	// agent suggestion: if len(callKey)<32 return zero,false; master :=
	// util.HKDFSHA256(make([]byte,32), callKey[:32], []byte(participantLid), 46);
	// return deriveSessionKeysFromMaster(master), true. Mirrors Rust Option as (val, ok).
	// human input:
	return E2eSrtpKeys{}, false
}

// DeriveE2eKeysFromRaw derives the E2E 1:1 keys from a keygen-v2 <raw_e2e> blob
// (>=32B) in place of callKey. The bool is false when rawE2e is shorter than 32 bytes.
func DeriveE2eKeysFromRaw(rawE2e []byte, participantLid string) (E2eSrtpKeys, bool) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L64-L70
	// TODO
	// agent suggestion: identical to DeriveE2eKeys but with rawE2e as the HKDF IKM.
	// human input:
	return E2eSrtpKeys{}, false
}

// BuildE2eRtpIV builds the E2E RTP IV: salt right-aligned into 16 bytes, SSRC XORed
// at bytes 4-7, and the 48-bit packet index (ROC<<16 | seq) XORed at bytes 8-13.
func BuildE2eRtpIV(salt []byte, ssrc uint32, roc uint32, seq uint16) [16]byte {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L74-L92
	// TODO
	// agent suggestion: off := 14 - len(salt); copy(iv[off:], salt); XOR the four SSRC
	// bytes into iv[4:8]; packetIndex := uint64(roc)*0x10000 + uint64(seq); XOR hi16 into
	// iv[8:10] and lo32 into iv[10:14] per the verbatim's big-endian byte extraction.
	// human input:
	return [16]byte{}
}

// CryptPayload AES-128-CTR encrypts/decrypts an RTP payload (the cipher is symmetric).
func CryptPayload(keys *E2eSrtpKeys, ssrc uint32, seq uint16, roc uint32, payload []byte) []byte {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L95-L101
	// TODO
	// agent suggestion: iv := BuildE2eRtpIV(keys.Salt[:], ssrc, roc, seq); out :=
	// append([]byte(nil), payload...); aes.NewCipher(keys.CipherKey[:]) ->
	// cipher.NewCTR(block, iv[:]) -> XORKeyStream(out, out); return out.
	// human input:
	return nil
}

// RocTracker is the send-side ROC tracker for monotonic 16-bit sequence numbers.
type RocTracker struct {
	roc         uint32
	lastSeq     uint16
	initialized bool
}

// Advance folds seq into the tracker and returns the current ROC, bumping it on the
// 0xFFFF->0x0000 wrap (a signed 16-bit gap below -32768).
func (t *RocTracker) Advance(seq uint16) uint32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L112-L124
	// TODO
	// agent suggestion: if !initialized seed lastSeq=seq, initialized=true, return roc;
	// if int32(seq)-int32(t.lastSeq) < -32768 roc++ (the int32 cast is load-bearing for
	// the wrap test); lastSeq=seq; return roc.
	// human input:
	return 0
}

// RecvRocTracker is the recv-side ROC estimator (RFC 3711 guess-index): it tolerates
// reorder/loss by guessing each packet's ROC from the highest seq seen.
type RecvRocTracker struct {
	roc         uint32
	sL          uint16
	initialized bool
}

// GuessRoc guesses the ROC for seq and folds it into the state, seeding from the
// first packet (roc=0). A reordered late packet returns the lower ROC untouched.
func (t *RecvRocTracker) GuessRoc(seq uint16) uint32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/e2e_srtp.rs#L139-L168
	// TODO
	// agent suggestion: port guess_roc read-for-read — pick v in {roc-1,roc,roc+1} via the
	// signed 16-bit gap against 0x8000 (int32 casts, wrapping add/sub for the ROC), then
	// fold: v==roc updates sL if seq>sL; v==roc+1 advances; v==roc-1 returns lower, no state change.
	// human input:
	return 0
}
