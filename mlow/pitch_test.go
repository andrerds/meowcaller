package mlow

import "testing"

// TestDecodeSmplPitch is the decode-side pitch KAT against pitch_vectors.json: for
// each active captured frame, the chain LSF(0) -> pulses(0) -> pitch(0) must
// reproduce the recorded lag/contour/gain_idx/filt_idx/int_lag_q6.
//
// Skipped until DecodeSmplPitch lands AND module #08 (decode_smpl_pulses) exists:
// the range coder must be advanced past the LSF and pulse symbols before the pitch
// block, so this KAT cannot run on the pitch decode alone.
func TestDecodeSmplPitch(t *testing.T) {
	t.Skip("blocked: needs DecodeSmplPitch impl + module #08 pulse (decode_smpl_pulses) to position the range coder")
}
