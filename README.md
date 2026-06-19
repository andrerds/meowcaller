# meowcaller

A clean-room, **pure-Go** WhatsApp 1:1 calling library — signaling, keying,
transport, and the MLow audio codec. It is an independent Go **flavor** of an
existing, validated implementation of the same protocol, built module by module
and verified byte-exact against shared known-answer vectors (KATs).

The abstract protocol spec (the "RFC") lives in **wacrg**. This repository is the
Go implementation; its build reference is a set of **datasheets** that carry the
reference source verbatim and state how each behavior must be realized in Go. The
Go code itself reads as original Go and never names any reference library.

No WASM bridge. No cgo for the protocol. No inherited code from earlier attempts.

**Start here:**

- [`PLAN.md`](PLAN.md) — the engineering plan and the model (spec / reference /
  derivative).
- [`AGENTS.md`](AGENTS.md) — how it gets built: human-audited, module by module,
  agents scaffold and explain in conversation, never run ahead.
- [`MODULES.md`](MODULES.md) — the module registry and build order.
- [`datasheets/`](datasheets/) — per-module datasheets (the exemplar is
  [`mlow-toc.md`](datasheets/mlow-toc.md)).

Status: planning. Nothing implemented yet. Scope: 1:1 calls.
