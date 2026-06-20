package mlow

import "testing"

// TestPostfilter is the postfilter KAT placeholder. The HP comb and per-packet
// harmonic postfilter have raw reference vectors (hp_postfilter_vectors.raw /
// harm_postfilter_vectors.raw); the comb is validated end-to-end. All bodies are
// stubs, so this is skipped — enable it (and copy the .raw vectors into testdata)
// when the postfilter bodies are implemented.
func TestPostfilter(t *testing.T) {
	t.Skip("blocked: postfilter bodies are stubs; enable with hp_postfilter_vectors.raw / harm_postfilter_vectors.raw when implemented")
}
