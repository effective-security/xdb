## Codebase Memory MCP

Use `codebase-memory-mcp` as the primary tool for understanding code structure, symbols, dependencies, call paths, and architecture.

### Project selection

- Indexed project names are stable and equal to repository directory names.
- Current mappings:
  - `~/code/es/xdb` → `xdb`

- Determine the current repository from its Git root and use its basename as the project name.
- Call `list_projects` once per session to verify that the expected project exists.
- Always pass `project="<repo-name>"` explicitly to every project-scoped MCP tool. Never rely on automatic project detection, filesystem paths, or a default project.
- Never substitute the workspace path, symlink-resolved path, or parent directory for the indexed project name.
- If the expected name is absent from `list_projects`, do not guess or silently reindex. Report the mismatch and ask whether the repository should be indexed.

### Tool usage

- Start architecture exploration with `get_graph_schema(project=...)` and `get_architecture(project=...)`.
- Use `search_graph(project=...)` to locate symbols before calling `get_code_snippet` or `trace_path`.
- Use qualified names returned by `search_graph`; do not construct them manually.
- Use `search_code(project=...)` for indexed text searches and filesystem search only when graph/index data is insufficient.
- Use `detect_changes(project=...)` when evaluating the impact of Git changes.
- Before trusting an empty result, verify the selected project and indexing status.
- Do not run `index_repository`, `delete_project`, or `manage_adr` unless the user explicitly requests an indexing, deletion, or ADR operation.

Example:

```text
list_projects()
search_graph(project="xdb", name_pattern=".*QueryParams.*", label="Function")
get_architecture(project="xdb", aspects=["all"])
```
