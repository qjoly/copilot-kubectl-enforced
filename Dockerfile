# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# copilot-kubectl-enforced — interactive shell image
#
# Bakes in:
#   - gh CLI (GitHub CLI)
#   - gh copilot extension  (requires GH_TOKEN build-arg)
#   - kubectl
#
# Usage:
#   docker build --build-arg GH_TOKEN=$GH_TOKEN -t copilot-kubectl-enforced:latest .
#   docker run -it \
#     -v /path/to/ro-kubeconfig:/root/.kube/config:ro \
#     -e GH_TOKEN=$GH_TOKEN \
#     copilot-kubectl-enforced:latest
# ---------------------------------------------------------------------------

FROM node:20-slim

# GH_TOKEN is required at build time to install the copilot extension.
# It is NOT stored in the image layers — it is only used during the RUN step.
ARG GH_TOKEN
RUN test -n "$GH_TOKEN" || (echo "ERROR: --build-arg GH_TOKEN=<token> is required" && exit 1)

# ---------------------------------------------------------------------------
# System dependencies
# ---------------------------------------------------------------------------
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        bash \
        && rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# gh CLI — install from the official GitHub APT repository
# ---------------------------------------------------------------------------
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] \
        https://cli.github.com/packages stable main" \
        > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# kubectl — pinned to the latest stable release at build time
# ---------------------------------------------------------------------------
RUN KUBECTL_VERSION=$(curl -fsSL https://dl.k8s.io/release/stable.txt) \
    && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/$(dpkg --print-architecture)/kubectl" \
        -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \
    && kubectl version --client --output=yaml

# ---------------------------------------------------------------------------
# gh copilot extension — baked in at build time using the build-arg token.
# The token is passed as an environment variable for this single RUN step
# only; it is not persisted in the image.
# ---------------------------------------------------------------------------
RUN GH_TOKEN="${GH_TOKEN}" gh extension install github/gh-copilot --force

# ---------------------------------------------------------------------------
# Runtime environment
# ---------------------------------------------------------------------------
ENV KUBECONFIG=/root/.kube/config

# Ensure the .kube directory exists so the volume mount works even if the
# host path resolves to a file (Docker creates a directory if the mount target
# is missing).
RUN mkdir -p /root/.kube

# Friendly shell prompt that shows the Kubernetes context
RUN echo 'export PS1="[copilot-k8s \$(kubectl config current-context 2>/dev/null || echo no-context)] \w \$ "' \
        >> /root/.bashrc \
    && echo 'echo ""' >> /root/.bashrc \
    && echo 'echo "  kubectl  : $(kubectl version --client --short 2>/dev/null)"' >> /root/.bashrc \
    && echo 'echo "  gh       : $(gh --version | head -1)"' >> /root/.bashrc \
    && echo 'echo "  copilot  : $(gh copilot --version 2>/dev/null || echo extension loaded)"' >> /root/.bashrc \
    && echo 'echo ""' >> /root/.bashrc

ENTRYPOINT ["/bin/bash", "--login"]
