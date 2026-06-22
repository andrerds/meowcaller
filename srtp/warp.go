package srtp

// WARP media transport framing. This file currently carries only the surface the
// rtp package depends on (the audio piggyback extension); the rest of WARP (the
// MI tag, the piggyback extension constant, the framing) lands with module #24.

// WarpPiggybackStartPacket is the 0-based packet index at which audio piggyback
// extensions begin: packets #1-2 carry none, #3+ piggyback.
const WarpPiggybackStartPacket = 2

// AudioPiggybackExtensionFor returns the audio piggyback extension word for
// packetIndex, or nil for the first packets / when disabled.
func AudioPiggybackExtensionFor(packetIndex int, enabled bool, startPacket int) *uint32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/41095d4e6ba4610e054e9ede3af1d5e88a83faee/wacore/src/voip/warp.rs#L15-L24
	// TODO
	// agent suggestion: if !enabled || packetIndex < startPacket return nil; else return a *uint32
	// holding BE(WARP_AUDIO_PIGGYBACK_EXT). Implemented + validated under #24 warp.
	// human input:
	return nil
}
