package mlow

import (
	"reflect"
	"testing"
)

// TestCcTablesVsOldBlob cross-checks the seed-built CcTables against the known-good
// cc_blob heap window (the old SmplMem) for every group it replaces: nrgres/gains
// (A/E), LTP gain (C), and the pulse split/runlen CDFs (B). This is the validation
// gate while both the seed and the old blob exist; once decode/encode are rewired to
// CcTables and the bit-exact KATs cover it, the old blob is dropped.
func TestCcTablesVsOldBlob(t *testing.T) {
	cc := LoadCcTables()
	m := LoadSmplMem()
	gNrg, gp, gCC := m.GNrg, m.GPitch, m.GCC

	eq := func(name string, a, b []uint16) {
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s mismatch", name)
		}
	}
	// Group A/E
	eq("nrgres_gain4", cc.NrgresGain4(), m.CDFAt(gNrg+0x1362, 85))
	eq("nrgres_shape4", cc.NrgresShape4(), m.CDFAt(gNrg+0x1098, 99))
	// Group C
	for prev := int32(-1); prev <= 15; prev++ {
		eq("acbgain_row", cc.AcbgainRow(prev), m.CDFAt(gp+0x302+uint32(prev*0x22)+0x22, 17))
	}
	eq("fcbgain_v", cc.FcbgainV(), m.CDFAt(gp+0xdc4, 35))
	for pf := int32(0); pf <= 33; pf++ {
		eq("fcbgain_v_delta", cc.FcbgainVDelta(pf), m.CDFAt(gp+0xe4c-uint32(pf)*2, 35))
	}
	for gi := int32(0); gi < 16; gi++ {
		w0, w2 := cc.AcbgainWeights(gi)
		if w0 != int32(m.I16(0xe85b0+uint32(gi)*4)) || w2 != int32(m.I16(0xe85b0+uint32(gi)*4+2)) {
			t.Errorf("acbgain_weights gi=%d mismatch", gi)
		}
	}
	if cc.NrgStep(2) != int32(m.I16(0xf35e0+4)) {
		t.Errorf("nrg_step mismatch")
	}
	for idx := int32(0); idx < 64; idx++ {
		if cc.GainRecon(true, idx) != int32(m.I16(0xf35f0+uint32(idx)*2)) {
			t.Errorf("gain_recon idx=%d mismatch", idx)
		}
	}
	// Group B: split CDFs
	base := gCC + 0xcd8
	for _, count := range []int32{1, 2, 5, 40, 79, 80, 120, 159} {
		minSplit := count - 80
		if minSplit < 0 {
			minSplit = 0
		}
		lo := count
		if 80 < lo {
			lo = 80
		}
		row := cc.SplitCmf(count)
		if row == nil {
			continue
		}
		n := int(lo-minSplit) + 2
		if int(minSplit)+n > len(row) {
			continue
		}
		eq("split_cmf", row[minSplit:int(minSplit)+n], m.CDFAt(m.U32(base+uint32(count)*8-8)+uint32(minSplit)*2, n))
	}
	// Group B: run-length CDFs
	for _, pos := range []int32{1, 7, 8, 40, 80, 159} {
		oct := (pos + 7) / 8
		magBase := gCC + uint32(oct)*0xa4
		cBaseOff := int32(m.U32(magBase))
		rl := cc.Runlen(oct)
		if rl.MaxSamples() != cBaseOff {
			t.Fatalf("runlen oct=%d maxSamples mismatch", oct)
		}
		for _, c := range []int32{1, 2, 5} {
			off := m.U32(magBase+uint32(c-1)*4-0xa0) + uint32(cBaseOff-pos)*2
			full := rl.Cmf(c)
			start := int(cBaseOff - pos)
			eq("runlen", full[start:start+int(pos+1)], m.CDFAt(off, int(pos+1)))
		}
	}
}
