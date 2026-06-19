package mlow

// Pitch / LTP parameters. The decode side (DecodeSmplPitch) reads the LTP gains and
// pitch lags from the bitstream and is the KAT-verified path; the estimator side
// (SmplPitch) is the encoder analysis and is a known soft-divergence (see datasheet).

const (
	// NumSubframes is the estimator's 8 pitch sub-blocks per 20 ms internal frame.
	NumSubframes = 8
	// MaxLTPBufLen is the perceptually-weighted speech buffer length the estimator reads.
	MaxLTPBufLen = 659
)

// ---- Decode side ----

// SmplPitchResult is the decoded LTP/pitch parameters for one internal frame.
type SmplPitchResult struct {
	GainIdx     [4]int32
	FiltIdx     [4]int32
	Lag         int32
	Contour     int32
	SampleLagQ6 [8]int32 // per-segment reconstructed pitch lag in Q6 (1/64-sample)
	NumSeg      int32
	IntLagQ6    [4]int32 // per-subframe pitch lag in Q6
	BlockLags   [8]int32 // per-40-sample-block lags (8 per 20 ms frame)
	NumSubfr    int32
}

// DecodeSmplPitch decodes the LTP gains and pitch lags. p3 = num subframes,
// p6 = config, subfrCounts = per-subframe pulse counts (from the pulse decode).
func DecodeSmplPitch(dec *RangeDecoder, mem *SmplMem, st *SmplLsfState, p2, p3, p6 int32, subfrCounts [4]int32) SmplPitchResult {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/ed12f35/wacore/src/voip/mlow/smpl_pitch.rs#L32-L198
	// TODO
	// agent suggestion: port decode_smpl_pitch read-for-read — the LTP gains loop
	//   (gain/filter CDFs from mem.g_pitch + offsets, keyed on p6 and prev_filt_idx),
	//   then the lag decode (absolute vs delta off st.PrevLag), contour, and the
	//   per-segment/per-subframe Q6 lag reconstruction. Integer/address arithmetic
	//   with wrapping u32/i32 → plain Go uint32/int32 operators.
	// human input:
	panic("mlow: DecodeSmplPitch not yet implemented (scaffold)")
}

// ---- Estimator side ----

// PitchEstState is the per-stream estimator state (cross-frame lag-block predictor).
type PitchEstState struct {
	PrevLag       float32
	PrevPitchCorr float32
	PrevLagblk    int32
	PrevLagidx    int32
}

// ResetCond clears the cross-frame lag-block predictor (smpl_pitch_reset_cond).
func (s *PitchEstState) ResetCond() {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/ed12f35/wacore/src/voip/mlow/smpl_pitch_enc.rs#L337-L341
	// TODO
	// agent suggestion: set PrevLagblk = -1 and PrevLagidx = -1 (the cond_coding=FALSE reset).
	// human input:
	panic("mlow: PitchEstState.ResetCond not yet implemented (scaffold)")
}

// PitchResult is the pitch estimator result for one internal frame.
type PitchResult struct {
	Pitchcorr    float32
	Lags         [NumSubframes]float32
	Laginds      [NumSubframes]int32
	AvgLag       float32
	HarmStrength float32
	BlocksegIdx  int
}

// pitchBlockSeg / pitchBlockTrack mirror the reference PitchTables sub-records.
type pitchBlockSeg struct {
	Nblocks int
	Blocks  []int
	Seglens []int
}

type pitchBlockTrack struct {
	Track       [NumSubframes]int
	Meanblock   float32
	Trackdeltas float32
}

// PitchTables holds the loaded constant tables (the smpl_pitch_tables dump).
type PitchTables struct {
	Blocksegs          []pitchBlockSeg
	Blocktracks        []pitchBlockTrack
	Blocksegs2idx      []int
	BlocksegIdxCmf     []uint32
	DeltaLagCmfs       [][]uint32
	BlocksegsIx        [][2]int
	FirstblockRange    [][2]int
	BlockTransitionCmf [][]uint32
}

// LoadPitchTables decodes the embedded pitch tables once and returns the shared set.
func LoadPitchTables() *PitchTables {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/ed12f35/wacore/src/voip/mlow/smpl_pitch_enc.rs#L87-L92
	// TODO
	// agent suggestion: same protobuf-asset path as LoadLsfCb — embed the reference's
	//   smpl_pitch_tables.bin (zlib+protobuf tables.proto PitchTables) at the package
	//   root, inflate + proto.Unmarshal + narrow (usize<-u32), memoized with sync.Once.
	// human input:
	panic("mlow: LoadPitchTables not yet implemented (scaffold)")
}

// SmplPitch is the full pitch estimator. ltpBuf is the perceptually-weighted speech of
// length MaxLTPBufLen; f2 is the LPC power spectrum; codedAsActiveVoice gates the search.
func SmplPitch(st *PitchEstState, ltpBuf []float32, f2 *[SmplFLen]float32, codedAsActiveVoice bool) PitchResult {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/ed12f35/wacore/src/voip/mlow/smpl_pitch_enc.rs#L848-L1215
	// TODO
	// agent suggestion: faithful f32 port of smpl_pitch — autocorrelation upsample
	//   search, get_maxi/get_maxi_k survivors (strict >, lowest-index-wins), the
	//   harmonicity cache keyed on the rounded harmonic bin, and the block-track lag
	//   selection. NOTE: encoder soft-divergence (~0.03 vs C); not a byte-exact target.
	// human input:
	panic("mlow: SmplPitch not yet implemented (scaffold)")
}
