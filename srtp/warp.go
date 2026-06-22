package srtp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
)

// WARP RTP extension constants and the WARP MESSAGE-INTEGRITY tag (HMAC-SHA1
// appended to protected packets).

const WarpExtProfile uint16 = 0xdebe

const (
	WarpMITagLen             = 4
	WarpPiggybackStartPacket = 2
)

// WarpAudioPiggybackExt is the audio piggyback extension word (big-endian bytes).
var WarpAudioPiggybackExt = [4]byte{0x30, 0x01, 0x00, 0x00}

// AudioPiggybackExtensionFor returns the audio piggyback extension word for
// packetIndex, or nil for the first packets / when disabled.
func AudioPiggybackExtensionFor(packetIndex int, enabled bool, startPacket int) *uint32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/warp.rs#L15-L24
	if !enabled || packetIndex < startPacket {
		return nil
	}
	w := binary.BigEndian.Uint32(WarpAudioPiggybackExt[:])
	return &w
}

// ComputeWarpMITag is the WARP MI tag: the first tagLen bytes of
// HMAC-SHA1(authKey, packetWithoutTag || roc_be32).
func ComputeWarpMITag(authKey, packetWithoutTag []byte, roc uint32, tagLen int) []byte {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/warp.rs#L27-L38
	mac := hmac.New(sha1.New, authKey)
	mac.Write(packetWithoutTag)
	var rocBE [4]byte
	binary.BigEndian.PutUint32(rocBE[:], roc)
	mac.Write(rocBE[:])
	return mac.Sum(nil)[:tagLen]
}

// AppendWarpMITag appends the WARP MI tag to a protected packet.
func AppendWarpMITag(authKey, packetWithoutTag []byte, roc uint32, tagLen int) []byte {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/warp.rs#L41-L52
	tag := ComputeWarpMITag(authKey, packetWithoutTag, roc, tagLen)
	out := make([]byte, 0, len(packetWithoutTag)+len(tag))
	out = append(out, packetWithoutTag...)
	return append(out, tag...)
}
