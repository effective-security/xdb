# Roadmap

Active planned work. Order is suggested, not a delivery commitment. Current runtime navigation
is in [codemap.md](codemap.md); defects are in
the local, gitignored `FINDINGS.md`, whose IDs are not cited here.

## Implemented baseline

- v1.0: `xdb.ID` is `type ID uint64`, an immutable value: comparable with
  `==`, ordered, a valid map key and safe to share between goroutines.
  Clients replace `xdb.ID{}` with `xdb.ID(0)`; `Set` and `Reset` on a copy
  no longer change the original.

## Active work

| ID  | Work | Ownership / contract |
| --- | ---- | -------------------- |

## Completion policy

Move completed features into their owning docs and update the codemap in
the same change. Contract changes include proto/SQL generation and
compatibility notes; shared-state changes require race and multi-process
checks. Never infer completion from a historical milestone's recorded test
or benchmark result.

## Lint

- Tighten the golangci-lint configuration.
- Report uses of deprecated APIs (staticcheck SA1019) in lint and CI.
