package mlow

import "math"

const (
	SmplLPCOrder  = 16
	SmplLPCBufLen = 448
	SmplLPCNFFT   = 512
	SmplFLen      = SmplLPCNFFT/2 + 1
)

// smplPI is the truncated literal the reference uses (not math.Pi) — load-bearing
// for bit-faithful window/NLSF math.
//
// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L25
const smplPI = 3.1415926535897

const (
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L411-L414
	lsfCosTabSzFix         = 128
	binDivStepsA2NLSFFix   = 3
	maxIterationsA2NLSFFix = 16
	silkInt16Max           = 32767
)

const (
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L26-L32
	//
	// smplPIF64 mirrors the reference's `SMPL_PI as f64`: the f32 literal widened,
	// not full-precision pi.
	smplPIF64          = float64(float32(3.1415926535897))
	smplLPCReg         = 5e-7
	smplLPCBwe         = 0.9999
	smplLPCWin120msLen = 264
	smplWin3LongLen    = 64
	smplWin3ShortLen   = 32

	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L94
	nfft4 = SmplLPCNFFT / 4 // 128
)

// smplWindowLPC20 applies the 20 ms LPC analysis window to a raw analysis buffer,
// producing the windowed buffer the autocorrelation FFT consumes. useLongWin
// selects the 64-tap vs 32-tap trailing cosine taper.
func genSinWin(n int) []float32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L39-L43
	w := make([]float32, n)
	for i := 0; i < n; i++ {
		t := (float32(i) + 1.0) / (float32(n) + 1.0) * smplPI / 2.0
		w[i] = float32(math.Sin(float64(t)))
	}
	return w
}

func genCosWin(n int) []float32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L46-L50
	w := make([]float32, n)
	for i := 0; i < n; i++ {
		t := (float32(i) + 1.0) / (float32(n) + 1.0) * smplPI / 2.0
		w[i] = float32(math.Cos(float64(t)))
	}
	return w
}

func smplWindowLPC20(input *[SmplLPCBufLen]float32, useLongWin bool) [SmplLPCBufLen]float32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L55-L90
	win1 := genSinWin(smplLPCWin120msLen)
	var win3 []float32
	var win3len int
	if useLongWin {
		win3, win3len = genCosWin(smplWin3LongLen), smplWin3LongLen
	} else {
		win3, win3len = genCosWin(smplWin3ShortLen), smplWin3ShortLen
	}
	var out [SmplLPCBufLen]float32
	for i := 0; i < smplLPCWin120msLen; i++ {
		out[i] = input[i] * win1[i]
	}
	mid := SmplLPCBufLen - smplLPCWin120msLen - smplWin3LongLen
	copy(out[smplLPCWin120msLen:smplLPCWin120msLen+mid], input[smplLPCWin120msLen:smplLPCWin120msLen+mid])
	base := SmplLPCBufLen - smplWin3LongLen
	for i := 0; i < win3len; i++ {
		out[base+i] = input[base+i] * win3[i]
	}
	if !useLongWin {
		for s := base + smplWin3ShortLen; s < base+smplWin3LongLen; s++ {
			out[s] = 0.0
		}
	}
	return out
}

// smplLPCAnalyzeWithF2 runs the full LPC analysis over a windowed buffer: returns
// the post-bandwidth-expansion monic LPC A[0..16] (A[0]=1) and the power spectrum
// F2[0..256] that the pitch and signal-mode paths consume.
// genCosRow accumulates row[k] = cos(omega)*scale, advancing omega by a running
// fmod in f64 (matching the reference, not cos(k*domega)).
func genCosRow(domega, scale float64) [nfft4]float64 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L98-L107
	var row [nfft4]float64
	omega := 0.0
	twoPi := 2.0 * smplPIF64
	for k := 0; k < nfft4; k++ {
		row[k] = math.Cos(omega) * scale
		omega = math.Mod(omega+domega, twoPi)
		if omega < 0 {
			omega += twoPi
		}
	}
	return row
}

type dctTables struct {
	cdif     [SmplLPCOrder / 2][nfft4]float64
	csumdiff [SmplLPCOrder / 4][nfft4]float64
	csumsum  [SmplLPCOrder / 4][nfft4]float64
}

func buildDctTables() dctTables {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L115-L143
	twoPi := 2.0 * smplPIF64
	nfft := float64(SmplLPCNFFT)
	var t dctTables
	for j := 0; j < SmplLPCOrder/2; j++ {
		t.cdif[j] = genCosRow(float64(1+j*2)*twoPi/nfft, 2.0/nfft)
	}
	for j := 0; j < SmplLPCOrder/4; j++ {
		t.csumdiff[j] = genCosRow(float64(2+j*4)*twoPi/nfft, 1.0/nfft)
	}
	for j := 0; j < SmplLPCOrder/4; j++ {
		t.csumsum[j] = genCosRow(float64(4+j*4)*twoPi/nfft, 1.0/nfft)
	}
	return t
}

// bruteDct derives the autocorrelation R[0..order] from the power spectrum via the
// precomputed cosine sums. All accumulation in f64.
func bruteDct(t *dctTables, f2 []float64, order int, r []float64) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L147-L186
	half := SmplLPCNFFT / 2
	f2sum := 0.0
	var f2dif, f2sumsum, f2sumdif [nfft4]float64
	for n := 0; n < nfft4; n++ {
		f2sum += f2[n] + f2[nfft4+n]
		f2dif[n] = f2[n] - f2[half-n]
		f2sumsum[n] = f2[n] + f2[half-n] + f2[nfft4+n] + f2[nfft4-n]
		f2sumdif[n] = f2[n] + f2[half-n] - f2[nfft4+n] - f2[nfft4-n]
	}
	f2dif[0] *= 0.5
	r[0] = (2.0*f2sum - f2[0] + f2[half]) / float64(SmplLPCNFFT)
	for j := 0; j < order/2; j++ {
		rtmp := 0.0
		row := &t.cdif[j]
		for k := 0; k < nfft4; k++ {
			rtmp += row[k] * f2dif[k]
		}
		r[1+j*2] = rtmp
	}
	for j := 0; j < order/4; j++ {
		rtmp := 0.0
		row := &t.csumdiff[j]
		for k := 0; k < nfft4; k++ {
			rtmp += row[k] * f2sumdif[k]
		}
		r[2+j*4] = rtmp
	}
	for j := 0; j < order/4; j++ {
		rtmp := 0.0
		row := &t.csumsum[j]
		for k := 0; k < nfft4; k++ {
			rtmp += row[k] * f2sumsum[k]
		}
		r[4+j*4] = rtmp
	}
}

// ac2rcDbl converts autocorrelation R[0..order] to reflection coefficients (Schur),
// with C0[0] *= (1+reg). Each rc[k] is truncated to f32, matching the reference.
func ac2rcDbl(corr []float64, order int, reg float32, rc []float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L190-L220
	c0 := make([]float64, order+1)
	c1 := make([]float64, order+1)
	copy(c0, corr[:order+1])
	c0[0] *= float64(1.0 + reg)
	copy(c1, c0[:order+1])
	for i := 0; i < order; i++ {
		rc[i] = 0
	}
	for k := 0; k < order; k++ {
		if c0[k+1] > c1[0] {
			rc[k] = -1.0
			break
		}
		if c0[k+1] < -c1[0] {
			rc[k] = 1.0
			break
		}
		if c1[0] == 0.0 {
			break
		}
		rcTmp := -c0[k+1] / c1[0]
		rc[k] = float32(rcTmp)
		for n := 0; n < order-k; n++ {
			ctmp1 := c0[n+k+1]
			ctmp2 := c1[n]
			c0[n+k+1] = ctmp1 + ctmp2*rcTmp
			c1[n] = ctmp2 + ctmp1*rcTmp
		}
	}
}

// rc2a converts reflection coefficients to monic LPC A[0..order] (A[0]=1).
func rc2a(rc []float32, order int, a []float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L223-L238
	for i := 1; i < order+1; i++ {
		a[i] = 0
	}
	a[0] = 1.0
	for k := 0; k < order; k++ {
		rcTmp := rc[k]
		for n := 0; n < (k+1)/2; n++ {
			tmp1 := a[n+1]
			tmp2 := a[k-n]
			a[n+1] = tmp1 + tmp2*rcTmp
			a[k-n] = tmp2 + tmp1*rcTmp
		}
		a[k+1] = rcTmp
	}
}

// bweExpand bandwidth-expands the monic LPC coefficients: A[i] *= bwe^i.
func bweExpand(a []float32, order int, bwe float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L241-L247
	c := bwe
	for i := 1; i < order+1; i++ {
		a[i] *= c
		c *= bwe
	}
}

func smplLPCAnalyzeWithF2(windowed *[SmplLPCBufLen]float32) ([SmplLPCOrder + 1]float32, [SmplFLen]float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L255-L283
	var xbuf [SmplLPCNFFT]float32
	copy(xbuf[:SmplLPCBufLen], windowed[:])
	var f [SmplLPCNFFT]float32
	rfftForwardOrdered(xbuf[:], f[:])

	var f2 [SmplFLen]float32
	f2[0] = f[0] * f[0]
	f2[SmplLPCNFFT/2] = f[1] * f[1]
	for i := 1; i < SmplLPCNFFT/2; i++ {
		f2[i] = f[2*i]*f[2*i] + f[2*i+1]*f[2*i+1]
	}
	f2d := make([]float64, SmplFLen)
	for i := 0; i < SmplFLen; i++ {
		f2d[i] = float64(f2[i])
	}

	tables := buildDctTables()
	var r [SmplLPCOrder + 1]float64
	bruteDct(&tables, f2d, SmplLPCOrder, r[:])

	var rc [SmplLPCOrder]float32
	ac2rcDbl(r[:], SmplLPCOrder, smplLPCReg, rc[:])
	var a [SmplLPCOrder + 1]float32
	rc2a(rc[:], SmplLPCOrder, a[:])
	bweExpand(a[:], SmplLPCOrder, smplLPCBwe)
	return a, f2
}

// smplLPCInterpol returns the per-subframe interpolated LPC predictor coefficients
// (interpolation index 0) and the carried last-subframe NLSF. nlsf2a is the
// decoder's NLSF→A conversion, supplied by the caller.
func smplLPCInterpol(
	lsf, prevLSF []float32,
	nlsf2a func(nlsf []float32) []float32,
) (predcoefs [4][SmplLPCOrder + 1]float32, ilsf [SmplLPCOrder]float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L358-L367
	// TODO
	// agent suggestion: delegate to smplLPCInterpolIdx with interpolIdx=0.
	// human input:
	return predcoefs, ilsf
}

// smplLPCInterpolIdx is smplLPCInterpol for an explicit interpolation-weight row.
func smplLPCInterpolIdx(
	lsf, prevLSF []float32,
	interpolIdx int,
	nlsf2a func(nlsf []float32) []float32,
) (predcoefs [4][SmplLPCOrder + 1]float32, ilsf [SmplLPCOrder]float32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L370-L407
	// TODO
	// agent suggestion: pick the interp row (clamp idx to 1); seed prev from prevLSF
	// when its last entry is non-zero else from lsf; per subframe interpolate
	// (1-w)*prev + w*lsf (or copy lsf when w==1), nlsf2a → A, force A[0]=1, then
	// lpc_stabilize via repeated bandwidth expansion until stable.
	// human input:
	return predcoefs, ilsf
}

func silkRshiftRound(a, shift int32) int32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L419-L425
	if shift == 1 {
		return (a >> 1) + (a & 1)
	}
	return ((a >> (shift - 1)) + 1) >> 1
}

func silkSmlaww(a32, b32, c32 int32) int32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L428-L434
	return int32(int64(a32) + ((int64(b32) * int64(c32)) >> 16))
}

func silkDiv32(a, b int32) int32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L437-L439
	return a / b
}

// silkBwexpander32 chirp-expands the Q16 LPC coefficients in place.
func silkBwexpander32(ar []int32, d int, chirpQ16 int32) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L448-L457
	chirp := chirpQ16
	chirpMinusOne := chirpQ16 - 65536
	for i := 0; i < d-1; i++ {
		ar[i] = int32((int64(chirp) * int64(ar[i])) >> 16)
		mul := chirp * chirpMinusOne
		chirp += silkRshiftRound(mul, 16)
	}
	ar[d-1] = int32((int64(chirp) * int64(ar[d-1])) >> 16)
}

func silkA2NLSFTransPoly(p []int32, dd int) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L459-L466
	for k := 2; k <= dd; k++ {
		for n := dd; n >= k+1; n-- {
			p[n-2] -= p[n]
		}
		p[k-2] -= p[k] << 1
	}
}

func silkA2NLSFEvalPoly(p []int32, x int32, dd int) int32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L468-L475
	xQ16 := x << 4
	y32 := p[dd]
	for n := dd - 1; n >= 0; n-- {
		y32 = silkSmlaww(p[n], y32, xQ16)
	}
	return y32
}

func silkA2NLSFInit(aQ16, p, q []int32, dd int) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L477-L490
	p[dd] = 1 << 16
	q[dd] = 1 << 16
	for k := 0; k < dd; k++ {
		p[k] = -aQ16[dd-k-1] - aQ16[dd+k]
		q[k] = -aQ16[dd-k-1] + aQ16[dd+k]
	}
	for k := dd; k >= 1; k-- {
		p[k-1] -= p[k]
		q[k-1] += q[k]
	}
	silkA2NLSFTransPoly(p, dd)
	silkA2NLSFTransPoly(q, dd)
}

// silkA2NLSF converts monic whitening coefficients (Q16) to NLSF (Q15). It mutates
// aQ16 (bandwidth expansion on non-convergence). d is the even filter order.
func silkA2NLSF(nlsf, aQ16 []int32, d int) {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L494-L589
	dd := d >> 1
	p := make([]int32, dd+1)
	q := make([]int32, dd+1)
	silkA2NLSFInit(aQ16, p, q, dd)

	useQ := false
	poly := func() []int32 {
		if useQ {
			return q
		}
		return p
	}
	xlo := silkLSFCosTabFIXQ12[0]
	ylo := silkA2NLSFEvalPoly(poly(), xlo, dd)

	var rootIx int
	if ylo < 0 {
		nlsf[0] = 0
		useQ = true
		ylo = silkA2NLSFEvalPoly(q, xlo, dd)
		rootIx = 1
	}
	k := 1
	var iter, thr int32
	for {
		xhi := silkLSFCosTabFIXQ12[k]
		yhi := silkA2NLSFEvalPoly(poly(), xhi, dd)

		if (ylo <= 0 && yhi >= thr) || (ylo >= 0 && yhi <= -thr) {
			if yhi == 0 {
				thr = 1
			} else {
				thr = 0
			}
			xloL, yloL, xhiL := xlo, ylo, xhi
			ffrac := int32(-256)
			for m := int32(0); m < binDivStepsA2NLSFFix; m++ {
				xmid := silkRshiftRound(xloL+xhiL, 1)
				ymid := silkA2NLSFEvalPoly(poly(), xmid, dd)
				if (yloL <= 0 && ymid >= 0) || (yloL >= 0 && ymid <= 0) {
					xhiL = xmid
					yhi = ymid
				} else {
					xloL = xmid
					yloL = ymid
					ffrac += 128 >> m
				}
			}
			absYloL := yloL
			if absYloL < 0 {
				absYloL = -absYloL
			}
			if absYloL < 65536 {
				den := yloL - yhi
				nom := (yloL << (8 - binDivStepsA2NLSFFix)) + (den >> 1)
				if den != 0 {
					ffrac += silkDiv32(nom, den)
				}
			} else {
				ffrac += silkDiv32(yloL, (yloL-yhi)>>(8-binDivStepsA2NLSFFix))
			}
			nlsf[rootIx] = min((int32(k)<<8)+ffrac, silkInt16Max)

			rootIx++
			if rootIx >= d {
				break
			}
			useQ = rootIx&1 != 0
			xlo = silkLSFCosTabFIXQ12[k-1]
			ylo = (1 - (int32(rootIx) & 2)) << 12
		} else {
			k++
			xlo = xhi
			ylo = yhi
			thr = 0
			if k > lsfCosTabSzFix {
				iter++
				if iter > maxIterationsA2NLSFFix {
					nlsf[0] = silkDiv32(1<<15, int32(d)+1)
					for kk := 1; kk < d; kk++ {
						nlsf[kk] = nlsf[kk-1] + nlsf[0]
					}
					return
				}
				silkBwexpander32(aQ16, d, int32(65536-(1<<iter)))
				silkA2NLSFInit(aQ16, p, q, dd)
				useQ = false
				xlo = silkLSFCosTabFIXQ12[0]
				ylo = silkA2NLSFEvalPoly(p, xlo, dd)
				if ylo < 0 {
					nlsf[0] = 0
					useQ = true
					ylo = silkA2NLSFEvalPoly(q, xlo, dd)
					rootIx = 1
				} else {
					rootIx = 0
				}
				k = 1
			}
		}
	}
}

// smplA2NLSF16 converts post-BWE float LPC A[0..16] (A[0]=1) into the analysis
// NLSF in radians (0..pi) via the fixed-point silk forward A→NLSF.
func smplA2NLSF16(a []float32) [SmplLPCOrder]float32 {
	// Source of truth: https://github.com/oxidezap/whatsapp-rust/blob/674e85164b35ca19115dfebcf605708d15951ee7/wacore/src/voip/mlow/smpl_lpc.rs#L592-L604
	var aQ16 [SmplLPCOrder]int32
	for i := range SmplLPCOrder {
		aQ16[i] = int32(math.Round(float64(-a[i+1] * 65536.0)))
	}
	var lsfQ15 [SmplLPCOrder]int32
	silkA2NLSF(lsfQ15[:], aQ16[:], SmplLPCOrder)
	var nlsf [SmplLPCOrder]float32
	for i := range SmplLPCOrder {
		nlsf[i] = float32(lsfQ15[i]) / 32768.0 * smplPI
	}
	return nlsf
}
