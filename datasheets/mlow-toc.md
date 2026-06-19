# Datasheet: `mlow/toc`

**Status:** planned (recovered + KAT-proven; ready to scaffold)
**Registry #:** 02 · **Depends on:** none · **Depended on by:** `mlow/decoder`, `rtp` (routing)
**Abstract spec (wacrg):** `docs/codec/mlow/decode-pipeline.md` (§TOC routing)

## What this module does

Parses the first byte (the "smpl TOC") of a bare MLow frame to learn how to decode
the rest, and routes standard-Opus frames away from the MLow path. Every inbound
media frame starts here.

## Reference implementation (verbatim)

```rust
//! MLow "smpl_toc" — the first byte of a bare MLow frame (WASM func 3544). The smpl TOC is only
//! valid when `(b & 0xC0) != 0xC0`; `(b & 0xC0) == 0xC0` marks a STANDARD Opus/CELT TOC instead,
//! which we route to stock libopus. Ported from the Go reference (`mlow_toc.go`).
//!
//! Bit layout (LSB = bit0): bit7=SID(DTX/CNG), bit6=VAD, bit5=internal rate(0→16k,1→32k),
//! bits4:3→frame size index into {10,20,60,120}ms, bit2=flag2, bit1=voiced-enable, bit0=flag0.

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct MlowToc {
    pub std_opus: bool,
    pub sid: bool,
    pub vad: bool,
    pub sample_rate: i32,
    pub frame_ms: i32,
    pub voiced: bool,
    pub active: bool,
    pub flag2: bool,
    pub flag0: bool,
}

/// Frame duration (ms) of a standard Opus packet from its TOC config field `b>>3` (RFC 6716
/// Table 2). 2.5 ms is rounded up — the smpl path only needs an output length for CNG frames.
fn standard_opus_frame_ms(b: u8) -> i32 {
    let config = b >> 3;
    if config < 12 {
        [10, 20, 40, 60][(config & 3) as usize] // SILK NB/MB/WB
    } else if config < 16 {
        [10, 20][((config - 12) & 1) as usize] // Hybrid
    } else {
        match config & 3 {
            0 => 3, // 2.5 ms rounded up
            1 => 5,
            2 => 10,
            _ => 20,
        }
    }
}

/// Parse the smpl TOC byte (WASM func 3544).
pub(crate) fn parse_mlow_toc(b: u8) -> MlowToc {
    if b & 0xC0 == 0xC0 {
        return MlowToc {
            std_opus: true,
            sid: false,
            vad: false,
            sample_rate: 16000,
            frame_ms: standard_opus_frame_ms(b),
            voiced: false,
            active: false,
            flag2: false,
            flag0: false,
        };
    }
    let bit1 = (b >> 1) & 1 != 0;
    let vad = (b >> 6) & 1 != 0;
    MlowToc {
        std_opus: false,
        sid: b >> 7 != 0,
        vad,
        sample_rate: if b & 0x20 != 0 { 32000 } else { 16000 },
        frame_ms: [10, 20, 60, 120][((b >> 3) & 3) as usize],
        voiced: vad && bit1,
        active: vad || bit1,
        flag2: (b >> 2) & 1 != 0,
        flag0: b & 1 != 0,
    }
}
```

Test contract (verbatim, abridged to the assertions): exhaustive over all 256 byte
values against `testdata/toc_vectors.json`, asserting every field
(`std, sid, vad, sr, ms, voiced, active, f2, f0`).

## Go implementation

- **Package / file:** `mlow/toc.go`
- **Public API (signatures only):**

```go
package mlow

// SmplTOC is the decoded smpl TOC. When StdOpus is true, the remaining fields are
// unused and the frame is a standard Opus/CELT packet (route to a stock Opus
// decoder, not the MLow path).
type SmplTOC struct {
	StdOpus    bool
	SID        bool
	VAD        bool
	SampleRate int // Hz: 16000 or 32000
	FrameMs    int
	Voiced     bool
	Active     bool
	Flag2      bool
	Flag0      bool
}

func ParseSmplTOC(b byte) SmplTOC
```

- **Behavior in Go:**
  - `(b & 0xC0) == 0xC0` → standard-Opus branch: `StdOpus=true`, `SampleRate=16000`,
    `FrameMs=standardOpusFrameMs(b)`, all other fields zero-value.
  - `standardOpusFrameMs`: `config := b >> 3`; `config<12` → `[]int{10,20,40,60}[config&3]`;
    `config<16` → `[]int{10,20}[(config-12)&1]`; else `switch config&3 {0→3,1→5,2→10,_→20}`.
  - smpl branch: `SID = b>>7 != 0`, `VAD = (b>>6)&1 != 0`,
    `SampleRate = 32000 if b&0x20 != 0 else 16000`,
    `FrameMs = []int{10,20,60,120}[(b>>3)&3]`, `bit1 = (b>>1)&1 != 0`,
    `Voiced = VAD && bit1`, `Active = VAD || bit1`, `Flag2 = (b>>2)&1 != 0`,
    `Flag0 = b&1 != 0`.
  - Go-vs-reference notes: the reference's `i32` fields become Go `int`; the
    function is pure (no receiver, no error). Index slices are fine — every index
    is masked to its valid range, so no bounds risk. No reference library is named
    anywhere in the Go file.

## Validation (KAT)

- **Vector:** `mlow/testdata/toc_vectors.json` — 256 entries, one per byte value,
  each with `b, std, sid, vad, sr, ms, voiced, active, f2, f0`. Copy it verbatim
  from the reference's `testdata/toc_vectors.json`.
- **Test:** `go test ./mlow -run TestParseSmplTOCAgainstKAT`.
- **Done when:** all 256 entries match exactly. (This port has been proven to pass
  256/256.)

## Open decisions

- None outstanding — the full byte space is covered by the KAT, so there is no
  unobserved input. Safe first module.
