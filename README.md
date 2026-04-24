# copilot-kubectl-enforced

A CLI tool that provisions a scoped, read-only Kubernetes ServiceAccount (secrets excluded), generates a short-lived kubeconfig, and drops you directly into the **GitHub Copilot CLI** inside an isolated container — then cleans everything up when you exit.

---

## How it works

```
┌─────────────────────────────────────────────────────────────┐
│                      your machine                           │
│                                                             │
│  copilot-kubectl-enforced                                   │
│    │                                                        │
│    ├─ 1. Creates ServiceAccount, ClusterRole (no secrets),  │
│    │       ClusterRoleBinding  ──────────────► cluster      │
│    │                                                        │
│    ├─ 2. Issues a 24 h ServiceAccount token  ◄─── cluster   │
│    │       Writes  ./ro-kubeconfig  (mode 0600)             │
│    │                                                        │
│    ├─ 3. Pulls ghcr.io/qjoly/copilot-kubectl-enforced       │
│    │                                                        │
│    ├─ 4. docker / podman run  ──► container                 │
│    │         -v ro-kubeconfig:/root/.kube/config:ro         │
│    │         -e GH_TOKEN                                    │
│    │         → gh copilot suggest  (interactive)            │
│    │                                                        │
│    └─ 5. On exit: deletes SA, ClusterRole,                  │
│              ClusterRoleBinding, ro-kubeconfig              │
└─────────────────────────────────────────────────────────────┘
```

### RBAC design

The `ClusterRole` is built **dynamically** at runtime using the Kubernetes discovery API: every resource in every API group is enumerated from the live cluster and granted `get`, `list`, `watch` — except `secrets`, which are never included.

This means the role works automatically with CRDs and custom API groups without any manual configuration.

| Resource | Name | Scope |
|---|---|---|
| `ServiceAccount` | `copilot-readonly` | namespace |
| `ClusterRole` | `copilot-readonly` | cluster-wide |
| `ClusterRoleBinding` | `copilot-readonly` | cluster-wide |

---

## Requirements

| Requirement | Notes |
|---|---|
| Go 1.21+ | For building from source |
| `docker` or `podman` | Auto-detected; `docker` preferred |
| A kubeconfig with **cluster-admin** | Used only to provision RBAC |
| `GH_TOKEN` env var | Fine-grained PAT with `copilot_requests: write` — see [docs/github-pat.md](docs/github-pat.md) |
| A GitHub Copilot subscription | Required to use the Copilot CLI |
| `cosign` (optional) | Required for image signature verification — see [docs/cosign.md](docs/cosign.md). Use `--insecure-image` to skip. |

---

## Quickstart

### 1. Install

**Pre-built binary** (Linux, macOS, Windows — amd64 / arm64):

```sh
# macOS (arm64)
curl -fsSL https://github.com/qjoly/copilot-kubectl-enforced/releases/latest/download/copilot-kubectl-enforced_latest_darwin_arm64.tar.gz \
  | tar -xz && sudo mv copilot-kubectl-enforced /usr/local/bin/

# Linux (amd64)
curl -fsSL https://github.com/qjoly/copilot-kubectl-enforced/releases/latest/download/copilot-kubectl-enforced_latest_linux_amd64.tar.gz \
  | tar -xz && sudo mv copilot-kubectl-enforced /usr/local/bin/
```

All releases and checksums are on the [Releases page](https://github.com/qjoly/copilot-kubectl-enforced/releases).

**From source:**

```sh
git clone https://github.com/qjoly/copilot-kubectl-enforced.git
cd copilot-kubectl-enforced
go build -o copilot-kubectl-enforced .
```

### 2. Export your GitHub token

Create a fine-grained PAT scoped **only** to Copilot (see [docs/github-pat.md](docs/github-pat.md)):

```sh
export GH_TOKEN=github_pat_xxxxxxxxxxxx
```

### 3. Run

```sh
copilot-kubectl-enforced
```

The tool connects to the cluster in your current `KUBECONFIG`, provisions the RBAC resources, generates a restricted kubeconfig, and opens the Copilot CLI. When you exit, everything is deleted automatically.

---

## Usage

```
copilot-kubectl-enforced [flags]

Flags:
      --build            Build the image from the local Dockerfile instead of pulling
      --image string     Container image to run
                         (default "ghcr.io/qjoly/copilot-kubectl-enforced:latest")
      --insecure-image   Skip cosign signature verification (unsigned or local images)
      --kubeconfig       Admin kubeconfig path
                         (default: $KUBECONFIG or ~/.kube/config)
      --namespace        Namespace for the ServiceAccount  (default "default")
      --no-cleanup       Skip deleting RBAC resources and kubeconfig on exit
      --out string       Path for the generated RO kubeconfig  (default "./ro-kubeconfig")
      --runtime string   Container runtime: docker or podman  (default: auto-detect)
      --sa-name string   Name of the SA / ClusterRole / CRB  (default "copilot-readonly")
      --token-ttl        ServiceAccount token lifetime  (default 24h)
  -h, --help
```

### Examples

```sh
# Use a specific kubeconfig and namespace
copilot-kubectl-enforced --kubeconfig ~/.kube/staging --namespace platform

# Use podman explicitly
copilot-kubectl-enforced --runtime podman

# Use a specific image tag (e.g. a commit build)
copilot-kubectl-enforced --image ghcr.io/qjoly/copilot-kubectl-enforced:sha-538cd59

# Keep RBAC resources after exit (for debugging)
copilot-kubectl-enforced --no-cleanup

# Build the image locally instead of pulling (skips signature verification)
GH_TOKEN=$GH_TOKEN copilot-kubectl-enforced --build

# Use an unsigned or locally-built image (skips cosign verification)
copilot-kubectl-enforced --insecure-image
```

---

## Container image

Images are published to [ghcr.io/qjoly/copilot-kubectl-enforced](https://github.com/qjoly/copilot-kubectl-enforced/pkgs/container/copilot-kubectl-enforced).

| Tag | Updated on |
|---|---|
| `latest` | Release tag (`v*`) |
| `v1.2.3` | Release tag — immutable |
| `edge` | Every commit to `main` |
| `sha-<7chars>` | Every commit to `main` — immutable |

The image contains:

- `kubectl` (latest stable at build time)
- `gh` CLI (latest stable at build time)
- The `gh copilot` extension is installed at container startup via `GH_TOKEN`

The image requires **no token at build time**. `GH_TOKEN` is only needed at runtime and is forwarded automatically from your shell environment.

### Image signatures

Every image published to `ghcr.io` is signed with
[cosign keyless signing](docs/cosign.md) via GitHub Actions OIDC.  The CLI
verifies the signature automatically before starting the container — no
configuration needed as long as `cosign` is in your `PATH`.

```sh
# Verification happens automatically:
copilot-kubectl-enforced

# Skip verification for unsigned or locally-built images:
copilot-kubectl-enforced --insecure-image
```

See **[docs/cosign.md](docs/cosign.md)** for full details on how signing works,
manual verification, and troubleshooting.

### Build locally

```sh
docker build -t copilot-kubectl-enforced:local .
```

---

## Cleanup behaviour

On exit — whether the user types `exit`, closes the terminal, or hits Ctrl+C — the tool:

1. Deletes the `ClusterRoleBinding`
2. Deletes the `ClusterRole`
3. Deletes the `ServiceAccount`
4. Deletes the `ro-kubeconfig` file from disk

If provisioning fails partway through, only the resources that were actually created are deleted. Use `--no-cleanup` to skip this (e.g. to inspect what was created).

---

## GitHub token

A fine-grained PAT with a single permission is sufficient:

| Permission | Level |
|---|---|
| `copilot_requests` (account) | `write` |

No repository or organisation permissions are needed. See **[docs/github-pat.md](docs/github-pat.md)** for step-by-step instructions and a pre-filled token creation URL.

---

## Development

```sh
# Run directly
go run main.go

# Build
go build -o copilot-kubectl-enforced .

# Vet
go vet ./...
```

### CI / CD

| Workflow | Trigger | Action |
|---|---|---|
| `ci.yml` | Push / PR to `main` | Build, vet, GoReleaser check, push `sha-*` + `edge` Docker image, sign image with cosign |
| `release.yml` | Push `v*` tag | Push versioned Docker image, sign image with cosign, create GitHub Release with binaries |

To cut a release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

---

## Project structure

```
.
├── main.go                        Entry point
├── cmd/
│   └── root.go                    Cobra CLI, flags, lifecycle orchestration
├── internal/
│   ├── k8s/
│   │   ├── client.go              Kubernetes client setup
│   │   ├── rbac.go                ServiceAccount + ClusterRole + CRB (create / delete)
│   │   └── kubeconfig.go          TokenRequest + kubeconfig file generation
│   └── container/
│       └── runner.go              Image detection, pull, build, run, signal forwarding
├── Dockerfile                     Node.js 20 + gh CLI + kubectl
├── entrypoint.sh                  Installs gh copilot at startup, execs gh copilot suggest
├── .goreleaser.yaml               Cross-platform binary release config
├── .github/workflows/
│   ├── ci.yml                     CI + edge Docker image
│   └── release.yml                Versioned release
└── docs/
    ├── cosign.md              Image signature verification guide
    └── github-pat.md          Fine-grained PAT guide
```

---

## License

MIT
