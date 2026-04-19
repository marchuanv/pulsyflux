# Contributing

## Local CI Testing with Act

You can run the GitHub Actions CI workflow locally using [Act](https://github.com/nektos/act), which executes workflows in Docker containers.

### Install Act

```bash
# Windows (winget)
winget install nektos.act

# Windows (Chocolatey)
choco install act-cli
```

### Run the workflow

```bash
# Simulate a push event
act push

# Simulate a pull request event
act pull_request
```

### Secrets

To test the publish stage locally, copy `.secrets.example` to `.secrets` and fill in your `NPM_TOKEN`. Without credentials the publish stage is skipped automatically (it only triggers on version tags and requires `NPM_TOKEN`).

```bash
cp .secrets.example .secrets
```

### Windows vs Linux runner limitations

The CI workflow targets `windows-latest`, but Act maps this to a Linux container image (see `.actrc`). Steps that depend on Windows-specific tooling (e.g., building `broker_lib.dll` via `go build -buildmode=c-shared`) will not produce a valid Windows DLL inside the Linux container. Local Act runs are useful for validating workflow structure, dependency installation, and non-platform-specific steps. For full end-to-end validation including native builds, push to GitHub and use the real Windows runners.
