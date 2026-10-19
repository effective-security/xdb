# Roadmap

Active planned work. Order is suggested, not a delivery commitment. Current runtime navigation
is in [codemap.md](codemap.md); defects are in
the local, gitignored `FINDINGS.md`, whose IDs are not cited here.

## Implemented baseline

- v1.0: `xdb.ID` as an immutable `uint64` value, typed keyset cursors
  (`xdb.Cursor`), `QueryRows` iterators and exact imports in generated
  code; the client changes are in [RELEASE_NOTES.md](RELEASE_NOTES.md).

## Active work

| ID  | Work | Ownership / contract |
| --- | ---- | -------------------- |
| RM-001 | Remove the deprecated `EncodeCursor` and `DecodeCursor` once clients use `NewCursor` and `ParseCursor` | `query.go`; breaking: RELEASE_NOTES.md entry of the removing version |

## Completion policy

Move completed features into their owning docs and update the codemap in
the same change. Contract changes include proto/SQL generation and
compatibility notes; shared-state changes require race and multi-process
checks. Never infer completion from a historical milestone's recorded test
or benchmark result.

## Lint

- Tighten the golangci-lint configuration.
- Report uses of deprecated APIs (staticcheck SA1019) in lint and CI.
