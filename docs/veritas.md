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
- A Git post-commit hook in `.git/hooks/post-commit` automatically uploads code
  quality snapshots on every commit in the background. You can also run
  `veritas upload-analysis` manually.
