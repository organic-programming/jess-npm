---
# Holon Identity v1
uuid: "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f9a0b"
given_name: Jess
family_name: NPM
motto: "Calculemus!"
composer: "B. ALTER"
clade: "deterministic/toolchain"
status: draft
born: "2026-03-02"

# Lineage
parents: []
reproduction: "assisted"

# Optional
aliases: ["jess"]

# Metadata
generated_by: "codex"
lang: "go"
proto_status: defined
---

# Jess NPM

> *"Calculemus!"*

![Jess NPM](assets/jess-npm.jpg)

## Description

JavaScript/NPM toolchain holon. Wraps npm and Node.js CLIs, exposing
the JS/TS ecosystem as gRPC RPCs - dependency management, builds, tests,
scripts, audits, and package introspection.

Part of the OP standard suite - the JS counterpart to Rob Go.

## Contract

- Proto: `protos/npm/v1/npm.proto`
- Service: `npm.v1.NpmService`
- Transport: `stdio://` (default), `tcp://`, `unix://`

## Technical Notes

- Calls `npm` and `node` via `os/exec.Command` - requires Node.js on `$PATH`.
- Most npm commands support `--json` for structured output.
- `package.json` is parsed in-process for fast introspection.
- gRPC reflection enabled - works with `op grpc://` dispatch.
