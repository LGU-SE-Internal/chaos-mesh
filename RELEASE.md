# Chaos Mesh Helm Chart Release Process

This document describes the release process for the Chaos Mesh Helm chart in the LGU-SE-Internal fork with RuntimeMutatorChaos support.

## Overview

The Helm chart is published to two locations:
1. **GitHub Pages**: `https://lgu-se-internal.github.io/chaos-mesh`
2. **OCI Registry (GHCR)**: `oci://ghcr.io/lgu-se-internal/charts/chaos-mesh`

## Versioning Scheme

We follow semantic versioning with an optional pre-release suffix:

- **Stable releases**: `v2.7.0`, `v2.8.0`
- **Feature releases**: `v2.7.0-runtime-mutator.1`, `v2.7.0-runtime-mutator.2`
- **Pre-releases**: Any version with a hyphen after the semver is marked as pre-release

## Release Steps

### 1. Prepare the Release

Ensure all changes are committed and pushed to the main branch:

```bash
cd chaos-mesh

# Verify the chart
helm lint helm/chaos-mesh

# Test the chart locally (optional)
helm template chaos-mesh helm/chaos-mesh --namespace chaos-mesh
```

### 2. Create and Push the Tag

```bash
# Choose an appropriate version
VERSION="v2.7.0-runtime-mutator.1"

# Create the tag
git tag ${VERSION}

# Push the tag to trigger the release workflow
git push origin ${VERSION}
```

### 3. Monitor the Release

1. Go to the repository's Actions tab on GitHub
2. Find the "Release Helm Chart (Fork)" workflow
3. Monitor the progress:
   - `validate-tag`: Validates the tag format
   - `release-chart`: Packages and publishes to GitHub Pages
   - `publish-oci`: Publishes to GitHub Container Registry

### 4. Verify the Release

After the workflow completes:

```bash
# Add/update the Helm repository
helm repo add chaos-mesh-fork https://lgu-se-internal.github.io/chaos-mesh
helm repo update

# Search for available versions
helm search repo chaos-mesh-fork -l

# Verify OCI registry
helm show chart oci://ghcr.io/lgu-se-internal/charts/chaos-mesh --version ${VERSION#v}
```

## Installation

### From GitHub Pages

```bash
# Add the repository
helm repo add chaos-mesh-fork https://lgu-se-internal.github.io/chaos-mesh
helm repo update

# Install a specific version
helm install chaos-mesh chaos-mesh-fork/chaos-mesh \
  --namespace chaos-mesh \
  --create-namespace \
  --version 2.7.0-runtime-mutator.1

# Or install the latest version
helm install chaos-mesh chaos-mesh-fork/chaos-mesh \
  --namespace chaos-mesh \
  --create-namespace
```

### From OCI Registry

```bash
# Install directly from GHCR
helm install chaos-mesh oci://ghcr.io/lgu-se-internal/charts/chaos-mesh \
  --namespace chaos-mesh \
  --create-namespace \
  --version 2.7.0-runtime-mutator.1
```

### With Custom Values

```bash
# Install with custom configuration
helm install chaos-mesh chaos-mesh-fork/chaos-mesh \
  --namespace chaos-mesh \
  --create-namespace \
  --set controllerManager.replicas=3 \
  --set dashboard.create=true \
  --set chaosDaemon.runtime=containerd
```

## RuntimeMutatorChaos Configuration

After installation, you can create RuntimeMutatorChaos experiments:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: RuntimeMutatorChaos
metadata:
  name: constant-mutation-example
  namespace: default
spec:
  mode: one
  selector:
    namespaces:
      - your-app-namespace
    labelSelectors:
      app: your-java-app
  runtimeMutator:
    action: constant
    class: com.example.MyClass
    method: myMethod
    from: "100"
    to: "0"
    port: 9090
  duration: "60s"
```

## Troubleshooting

### Workflow Fails at validate-tag

Ensure your tag follows the `v*.*.*` pattern:
- ✅ `v2.7.0`
- ✅ `v2.7.0-runtime-mutator.1`
- ❌ `2.7.0` (missing 'v' prefix)
- ❌ `chart-2.7.0` (wrong prefix)

### Helm Repository Not Found

```bash
# Remove and re-add the repository
helm repo remove chaos-mesh-fork
helm repo add chaos-mesh-fork https://lgu-se-internal.github.io/chaos-mesh
helm repo update
```

### OCI Authentication Errors

The workflow uses `GITHUB_TOKEN` for GHCR authentication. If you need to push manually:

```bash
# Login to GHCR
echo $GITHUB_TOKEN | helm registry login ghcr.io -u $GITHUB_USER --password-stdin

# Push manually
helm package helm/chaos-mesh
helm push chaos-mesh-*.tgz oci://ghcr.io/lgu-se-internal/charts
```

### Chart Not Found After Release

1. Wait a few minutes for GitHub Pages to update
2. Check the workflow logs for errors
3. Verify the gh-pages branch has the new chart:
   ```bash
   git fetch origin gh-pages
   git show origin/gh-pages:index.yaml
   ```

## Release Checklist

- [ ] All code changes committed and tested
- [ ] RuntimeMutatorChaos CRD included in `helm/chaos-mesh/crds/`
- [ ] Chart version in `Chart.yaml` matches intended release
- [ ] Helm lint passes
- [ ] Tag created with correct format
- [ ] Tag pushed to trigger workflow
- [ ] Workflow completed successfully
- [ ] Chart available from Helm repository
- [ ] Chart available from OCI registry
- [ ] GitHub Release created with correct artifacts

## Related Documentation

- [RuntimeMutatorChaos Integration Guide](../.claude/ralph/docs/RuntimeMutatorChaos-Integration-Guide.md)
- [Chaos Mesh Official Documentation](https://chaos-mesh.org/docs/)
- [Helm Chart Best Practices](https://helm.sh/docs/chart_best_practices/)
