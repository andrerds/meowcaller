<!-- Datasheet template. Copy to datasheets/<module>.md. A datasheet carries the
     reference source VERBATIM (full context, no summary) and explains how the
     behavior is realized in Go. The Go implementation never names the reference. -->

# Datasheet: `<package>/<module>`

**Status:** planned | scaffolded | implemented | verified
**Registry #:** NN · **Depends on:** `<modules>` · **Depended on by:** `<modules>`
**Abstract spec (wacrg):** `<wacrg docs/... path or URL>`

## What this module does

One or two sentences, in plain terms.

## Reference implementation (verbatim)

The reference source for this behavior, pasted **as-is and in full** — this is the
ground truth, not a paraphrase. Do not abbreviate it.

```rust
// <file>.rs — pasted verbatim
```

(Paste the test/vectors block too when it clarifies the contract.)

## Go implementation

How this must be built in Go — concretely, in Go terms. Not assumptions about the
reference; the Go target.

- **Package / file:** `<pkg>/<file>.go`
- **Public API (signatures only):**

```go
// clean Go — NEVER names or alludes to the reference library
```

- **Behavior in Go:** the algorithm/format restated for a Go implementer (types,
  integer widths, slices vs arrays, error handling, allocation, endianness). Call
  out where Go idiom differs from the reference (e.g. `[]byte` vs `&[u8]`,
  explicit error returns vs panics).

## Validation (KAT)

- **Vector:** the `testdata/*.json` file (copied verbatim into `<pkg>/testdata/`)
  and exactly what it covers.
- **Test:** the Go test name + `go test` command.
- **Done when:** the precise pass condition.

## Open decisions

`TODO(human)` items: choices left to the reviewer, source ambiguities, and any
behavior not pinned by a vector. Each becomes a conversation; when it resolves, a
decision artifact is written to wacrg (on direction) and may be linked from the Go
file by URL.
