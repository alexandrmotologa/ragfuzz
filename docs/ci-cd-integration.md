# CI/CD Quality Gate Integration

ragfuzz is engineered to function as an automated quality gate in continuous integration and continuous deployment pipelines. It enforces reliability thresholds before RAG updates, embedding model migrations, or prompt changes reach production.

---

## 1. Exit Codes

ragfuzz adheres to standard Unix exit codes:
- `0`: Quality gate passed. All reliability thresholds were satisfied.
- `1`: Quality gate failed. One or more threshold criteria were breached.
- `2`: Runtime error. Configuration parsing error, connection failure, or invalid arguments.

---

## 2. GitHub Actions Integration

Add ragfuzz to your workflow to block pull requests that degrade retrieval recall or groundedness:

```yaml
name: RAG Reliability Gate

on:
  pull_request:
    branches: [ main ]

jobs:
  ragfuzz-gate:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download ragfuzz binary
        run: |
          curl -sSL -o /usr/local/bin/ragfuzz https://github.com/alexandrmotologa/ragfuzz/releases/latest/download/ragfuzz-linux-amd64
          chmod +x /usr/local/bin/ragfuzz

      - name: Start target RAG pipeline service
        run: |
          docker run -d -p 8080:8080 --name my-rag-service my-company/rag-api:staging
          sleep 5

      - name: Run ragfuzz Quality Gate
        run: |
          ragfuzz run \
            --config .ragfuzz.yaml \
            --target http://localhost:8080/query \
            --queries 100

      - name: Publish Test Results
        if: always()
        uses: EnricoMi/publish-unit-test-result-action@v2
        with:
          files: junit.xml

      - name: Upload HTML Evaluation Report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: ragfuzz-evaluation-report
          path: report.html
```

---

## 3. Baseline Regression Comparison (`ragfuzz diff`)

To prevent performance drift between git commits, export a baseline report on your main branch and compare new test runs in pull requests:

```bash
# In PR branch: run evaluation and save report
ragfuzz run --config ragfuzz.yaml --target http://localhost:8080/query

# Compare against the main branch baseline
ragfuzz diff baseline-report.json ragfuzz-report.json
```

If any attack vector regresses by more than 5.0%, `ragfuzz diff` returns exit code 1 and lists the exact regressed test cases.
