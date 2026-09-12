# Code Quality & Telemetry (Veritas)

This repository integrates with [Veritas](https://github.com/walterjgsp/veritas)
for static code analysis (Cognitive Complexity, Cyclomatic Complexity, LCOM4,
and Code Duplication) and automated test reporting.

---

## Configuration

Veritas is configured via [`.veritas/config.json`](../.veritas/config.json):

```json
{
  "project": "please-rules",
  "server": "http://localhost:8080",
  "test_results_path": "plz-out/log/test_results.xml",
  "thresholds": {
    "cognitive_complexity": 15,
    "cyclomatic_complexity": 10,
    "lcom4": 1,
    "duplication_pct": 5.0
  }
}
```

To initialize or reconfigure Veritas with post-commit hooks:

```bash
veritas init --project please-rules --server http://localhost:8080 --install-hooks
```

---

## Static Analysis

Run code quality analysis locally across all files:

```bash
# Analyze repository and show summary
veritas analyze --summary-only .

# Check for quality threshold violations
veritas analyze -V .
```

---

## Automatic Test Reporting & Snapshots

- `./pleasew test` automatically uploads test results to the Veritas dashboard
  (`http://localhost:8080`). To disable telemetry, run with `VERITAS_NO_HOOK=1`.
- **Git Hooks**: `.git/hooks/post-commit` and `.git/hooks/post-merge` automatically
  trigger `veritas upload-analysis` in the background after every local commit or merge.
  - Background processes are detached using `nohup` so that static analysis finishes
    even after Git terminates the hook process group.
  - Execution traces and logs are stored at `~/.veritas/hook.log` (`tail -f ~/.veritas/hook.log`).
- **Manual Upload**: You can upload a fresh code quality snapshot at any time by running:
  ```bash
  veritas upload-analysis
  ```
- **Remote vs Local Commits**: Git hooks are client-side only. Merges or commits performed
  directly on GitHub (e.g. web UI PR squash & merge) do not invoke local hooks. Pulling the changes
  locally (`git pull`) triggers `post-merge`, or you can trigger `veritas upload-analysis` in CI.

