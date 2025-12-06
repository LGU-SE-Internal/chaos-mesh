# EnvoyChaos - Envoy gRPC/HTTP Fault Injection for Chaos Mesh

## ⚠️ Important Notice

**Cilium Envoy Limitation**: Cilium's default Envoy proxy is a custom build that does NOT include the HTTP fault filter (see [cilium/proxy#62](https://github.com/cilium/proxy/issues/62)).

✅ **Workaround Available**: You can enable fault injection by:
- Manually compiling Cilium Envoy with fault extension enabled
- Deploying the custom Envoy image as a DaemonSet
- Reference: [Custom Envoy Configurations in Cilium](https://medium.com/@samyak-devops/how-to-apply-custom-envoy-configurations-in-a-cilium-setup-with-rate-limiting-example-5301972460f2)

## Usage with Custom Cilium Envoy

If you've built Cilium Envoy with fault filter support:

### Basic Setup

1. Deploy your custom Cilium Envoy with fault extension enabled
2. Create an EnvoyChaos resource:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-delay-example
  namespace: default
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-grpc-service
  mode: all
  protocol: grpc
  action: delay
  delay:
    fixedDelay: "500ms"
    percentage: 50
  duration: "60s"
```

### Verification

```bash
# Check if CiliumEnvoyConfig was created
kubectl get ciliumenvoyconfigs -A

# View Envoy logs
kubectl logs -n <namespace> <cilium-envoy-pod>

# Monitor controller logs
kubectl logs -n chaos-mesh <controller-pod> | grep envoychaos

# Test the fault injection
# You should observe 500ms delay on 50% of requests
```

## Alternative Solutions

If not using custom Cilium Envoy, consider these alternatives:

### Option 1: Use Istio EnvoyFilter

If you have Istio deployed, use native Istio EnvoyFilter for fault injection:

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: fault-injection
spec:
  workloadSelector:
    labels:
      app: myapp
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: SIDECAR_INBOUND
    patch:
      operation: INSERT_BEFORE
      value:
        name: envoy.filters.http.fault
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault
          delay:
            fixed_delay: 5s
            percentage:
              numerator: 50
              denominator: HUNDRED
```

### Option 2: Use HTTPChaos (Easiest)

Chaos Mesh's HTTPChaos already supports gRPC fault injection without requiring Envoy:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: HTTPChaos
metadata:
  name: grpc-delay
spec:
  selector:
    labelSelectors:
      app: my-grpc-service
  mode: all
  target: Request
  port: 50051
  delay: "500ms"
  duration: "60s"
```

**Benefits**:
- No Envoy dependency
- Uses tproxy technology
- Supports HTTP/1.1, HTTP/2 (gRPC)
- Production-tested

### Option 3: Deploy Standalone Envoy

Deploy standard Envoy as a sidecar or DaemonSet:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: envoy-proxy
spec:
  template:
    spec:
      containers:
      - name: envoy
        image: envoyproxy/envoy:v1.28-latest
```

## Overview

EnvoyChaos enables fault injection for gRPC and HTTP services through Envoy proxy integration. It leverages Envoy's fault filter to inject delays, aborts, and other faults into network traffic managed by Envoy, particularly when deployed with Cilium.

## Architecture

### Integration Approach

EnvoyChaos integrates with Envoy proxy through Kubernetes CRDs:

1. **CiliumEnvoyConfig**: For Envoy deployed via Cilium (recommended)
2. **EnvoyFilter**: For Envoy deployed via Istio or standalone

The controller automatically creates and manages these resources based on the EnvoyChaos specification.

### How It Works

```
EnvoyChaos CRD → Chaos Controller → CiliumEnvoyConfig/EnvoyFilter → Envoy Proxy → Fault Injection
```

When an EnvoyChaos resource is created:

1. The chaos controller selects target pods based on the selector
2. For each selected pod, it creates a CiliumEnvoyConfig resource
3. The CiliumEnvoyConfig configures Envoy's fault filter with the specified fault injection rules
4. Envoy proxy intercepts matching requests and applies the fault injection
5. When the chaos experiment ends, the configuration is removed

## Prerequisites

### Important: Envoy Compatibility

⚠️ **Cilium Envoy does NOT support fault injection** - The Cilium Envoy proxy is a custom build without the HTTP fault filter. See alternatives above.

### Envoy Deployment Options

EnvoyChaos requires a **full-featured Envoy proxy** with fault filter support:

#### Option 1: Istio (Recommended)

If you're using Istio:

- Istio includes full Envoy as a sidecar proxy
- EnvoyChaos can integrate via EnvoyFilter resources
- Set `envoyConfigNamespace` to the appropriate namespace

#### Option 2: Standalone Envoy

If you have Envoy deployed separately:

- Ensure Envoy is configured to watch for configuration updates
- EnvoyChaos can work with custom Envoy deployments
- May require additional configuration for dynamic updates

#### Option 3: Use HTTPChaos Instead

For the simplest solution that doesn't require Envoy:

- Use Chaos Mesh's built-in HTTPChaos
- Supports gRPC fault injection via tproxy
- No additional dependencies required

### Requirements

- Kubernetes cluster with Chaos Mesh installed
- Envoy proxy deployed (via Cilium, Istio, or standalone)
- RBAC permissions to create/update/delete CiliumEnvoyConfig or EnvoyFilter resources

## Usage

### Basic Examples

#### 1. gRPC Delay Injection

Inject a 500ms delay into 50% of gRPC requests:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-delay-example
  namespace: default
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-grpc-service
  mode: all
  protocol: grpc
  action: delay
  delay:
    fixedDelay: "500ms"
    percentage: 50.0
  duration: "60s"
```

#### 2. gRPC Abort Injection

Return UNAVAILABLE (code 14) for 30% of gRPC requests:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-abort-example
  namespace: default
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-grpc-service
  mode: all
  protocol: grpc
  action: abort
  abort:
    grpcStatus: 14  # UNAVAILABLE
    percentage: 30.0
  duration: "120s"
```

#### 3. HTTP Combined Fault Injection

Inject both delays and errors into HTTP traffic:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: http-fault-example
  namespace: default
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-http-service
  mode: fixed
  value: "2"
  protocol: http
  action: fault
  delay:
    fixedDelay: "200ms"
    percentage: 50.0
  abort:
    httpStatus: 503
    percentage: 20.0
  path: "/api/v1/*"
  method: "POST"
  duration: "5m"
```

## Configuration Reference

### Spec Fields

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `selector` | PodSelector | Selects target pods | Yes |
| `mode` | string | Pod selection mode: `one`, `all`, `fixed`, `fixed-percent`, `random-max-percent` | Yes |
| `value` | string | Value for `fixed` or `percent` modes | Conditional |
| `action` | string | Fault action: `fault`, `delay`, `abort` | Yes |
| `protocol` | string | Protocol type: `grpc`, `http` (default: `grpc`) | No |
| `delay` | DelayConfig | Delay configuration | Conditional |
| `abort` | AbortConfig | Abort configuration | Conditional |
| `percentage` | float64 | Overall percentage of requests to affect (0-100) | No |
| `path` | string | URI path filter (supports wildcards) | No |
| `method` | string | gRPC method or HTTP method filter | No |
| `headers` | map[string]string | Header filters | No |
| `targetService` | string | Target Kubernetes service name | No |
| `targetPort` | int32 | Target port number | No |
| `envoyConfigName` | string | Custom Envoy config resource name | No |
| `envoyConfigNamespace` | string | Namespace for Envoy config (defaults to chaos namespace) | No |
| `duration` | string | Duration of the chaos experiment | No |
| `remoteCluster` | string | Target remote cluster | No |

### Delay Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `fixedDelay` | string | Fixed delay duration (e.g., "100ms", "2s") | Yes |
| `percentage` | float64 | Percentage of requests to delay (0-100) | No |

### Abort Configuration

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `httpStatus` | int32 | HTTP status code (100-599) for HTTP protocol | Conditional |
| `grpcStatus` | int32 | gRPC status code (0-16) for gRPC protocol | Conditional |
| `percentage` | float64 | Percentage of requests to abort (0-100) | No |

### gRPC Status Codes

Common gRPC status codes for fault injection:

| Code | Name | Description |
|------|------|-------------|
| 0 | OK | Success |
| 1 | CANCELLED | Operation was cancelled |
| 2 | UNKNOWN | Unknown error |
| 3 | INVALID_ARGUMENT | Invalid argument |
| 4 | DEADLINE_EXCEEDED | Deadline expired |
| 5 | NOT_FOUND | Not found |
| 13 | INTERNAL | Internal server error |
| 14 | UNAVAILABLE | Service unavailable |

## Advanced Usage

### Target Specific gRPC Methods

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-method-specific
spec:
  selector:
    labelSelectors:
      app: my-service
  mode: all
  protocol: grpc
  action: delay
  method: "/myservice.MyService/SpecificMethod"
  delay:
    fixedDelay: "1s"
```

### Filter by Headers

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: header-filter-chaos
spec:
  selector:
    labelSelectors:
      app: my-service
  mode: all
  protocol: http
  action: abort
  headers:
    x-request-id: "test-.*"
    x-user-type: "beta"
  abort:
    httpStatus: 500
```

### Percentage-based Selection

Affect only 25% of matching pods:

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: percentage-chaos
spec:
  selector:
    labelSelectors:
      app: my-service
  mode: random-max-percent
  value: "25"
  protocol: grpc
  action: delay
  delay:
    fixedDelay: "500ms"
```

## ❌ Cilium Envoy Incompatibility

⚠️ **IMPORTANT: Cilium Envoy does NOT support fault injection**

### Why EnvoyChaos Doesn't Work with Cilium

Cilium's Envoy proxy is a **custom build** that excludes the HTTP fault filter:

- **Issue**: [cilium/proxy#62](https://github.com/cilium/proxy/issues/62)
- **Reason**: Cilium Envoy is optimized for L7 visibility and network policy
- **Impact**: CiliumEnvoyConfig will fail or be ignored for fault injection

### What to Use Instead

Do NOT use the Cilium integration section below. Instead:

1. **HTTPChaos** (Recommended): No Envoy needed, works today
2. **Istio EnvoyFilter**: If you have Istio deployed  
3. **Standalone Envoy**: Deploy full Envoy separately

See alternatives at the top of this document.

---

## ~~Integration with Cilium~~ (NOT SUPPORTED)

⚠️ **The following section is DEPRECATED and will not work with Cilium Envoy**

### ~~Cilium Configuration~~

When using Cilium with Envoy, ensure Cilium is configured with L7 proxy support:

```yaml
# Cilium ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: cilium-config
  namespace: kube-system
data:
  enable-l7-proxy: "true"
  envoy-config-path: "/etc/envoy"
```

### Service Requirements

For services to use Envoy L7 features with Cilium:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-grpc-service
  annotations:
    service.cilium.io/global: "true"
spec:
  type: ClusterIP
  ports:
  - port: 50051
    protocol: TCP
    name: grpc
  selector:
    app: my-grpc-service
```

## Troubleshooting

### Issue: Chaos not taking effect

1. Check if Envoy is properly deployed and running
2. Verify CiliumEnvoyConfig resources are created:
   ```bash
   kubectl get ciliumenvoyconfigs -A
   ```
3. Check Envoy logs for configuration errors:
   ```bash
   kubectl logs -n <namespace> <envoy-pod> -c cilium-envoy
   ```

### Issue: gRPC status codes not working

- Ensure you're using valid gRPC status codes (0-16)
- For HTTP traffic, use `httpStatus` instead of `grpcStatus`
- Check that `protocol` is set to "grpc"

### Issue: Headers not matching

- Verify header names and values are correct
- Header matching is case-sensitive
- Use regex patterns for flexible matching

## Comparison with Other Chaos Types

| Feature | EnvoyChaos | HTTPChaos | NetworkChaos |
|---------|------------|-----------|--------------|
| Layer | L7 (Application) | L7 (Application) | L3/L4 (Network) |
| Protocol Support | HTTP, gRPC | HTTP | TCP, UDP |
| Requires Envoy | Yes | No (uses tproxy) | No |
| gRPC-specific features | Yes | Limited | No |
| Performance Impact | Low | Medium | Low |
| Fine-grained control | High | Medium | Low |

## Best Practices

1. **Start with low percentages**: Begin with 10-20% fault injection and gradually increase
2. **Use specific selectors**: Target specific services or endpoints to limit blast radius
3. **Monitor metrics**: Track error rates and latencies during experiments
4. **Test in staging first**: Validate chaos experiments in non-production environments
5. **Set reasonable durations**: Use short durations (1-5 minutes) for initial tests
6. **Combine with SLOs**: Define Service Level Objectives and verify system behavior

## References

- [Envoy Fault Filter Documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/fault_filter)
- [Envoy Gateway Fault Injection](https://gateway.envoyproxy.io/docs/tasks/traffic/fault-injection/)
- [Cilium L7 Protocol Visibility](https://docs.cilium.io/en/stable/gettingstarted/http/)
- [gRPC Status Codes](https://grpc.github.io/grpc/core/md_doc_statuscodes.html)

## Contributing

Contributions are welcome! Please see the [Chaos Mesh contributing guide](../../CONTRIBUTING.md) for more information.

## License

EnvoyChaos is part of Chaos Mesh and is licensed under the Apache License 2.0.
