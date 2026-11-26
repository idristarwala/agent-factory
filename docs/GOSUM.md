# Committing go.sum

To ensure clean clones build:

```bash
go mod tidy
git add go.sum
git commit -m "Commit go.sum for reproducible builds"
```

This is required for CI and for users who clone the repo.
The hourly automation or a local run of the above will complete it.
