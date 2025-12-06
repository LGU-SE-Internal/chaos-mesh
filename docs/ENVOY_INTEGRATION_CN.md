# Envoy gRPC Fault Integration - 集成说明

## 重要说明 (Important Notice)

⚠️ **Cilium Envoy 限制**: Cilium 的默认 Envoy proxy 是定制版本，不包含 HTTP fault filter 功能（参见 [cilium/proxy#62](https://github.com/cilium/proxy/issues/62)）。

✅ **解决方案**: 可以通过以下方式启用 fault filter：
- 手动编译 Cilium Envoy 并启用 fault extension
- 将定制的 Envoy image 部署为 DaemonSet
- 参考: [How to Apply Custom Envoy Configurations in Cilium](https://medium.com/@samyak-devops/how-to-apply-custom-envoy-configurations-in-a-cilium-setup-with-rate-limiting-example-5301972460f2)

## 问题解答 (Answers to Questions)

### 1. 是否需要额外安装Envoy组件？

**答：取决于您的环境。**

- **如果使用默认 Cilium Envoy**：不支持，需要使用定制 image 或其他方案
- **如果使用定制 Cilium Envoy**（启用 fault filter）：可以使用，配合 CiliumEnvoyConfig
- **如果使用 Istio**：不需要额外安装，Istio 的 Envoy sidecar 支持 fault filter
- **如果没有 Envoy**：需要部署标准 Envoy 或使用 HTTPChaos 替代方案

### 2. 当前的Chaos Mesh流程是否能集成Envoy？

**答：可以集成。**

EnvoyChaos 已经完全集成到 Chaos Mesh 的现有流程中：
- 遵循 Chaos Mesh 的 CRD 设计模式
- 使用相同的 Pod 选择器机制
- 支持相同的调度和工作流功能
- 集成到现有的 Controller Manager 中

### 3. 以怎样的方式集成？k8s apply 还是调用Envoy的SDK？

**答：通过 Kubernetes API (kubectl apply) 方式集成。**

集成方式采用 **声明式 Kubernetes API**：

1. **用户层面**：通过 `kubectl apply -f envoychaos.yaml` 创建 EnvoyChaos 资源
2. **控制器层面**：EnvoyChaos 控制器自动创建 `CiliumEnvoyConfig` 或 `EnvoyFilter` 资源
3. **Envoy层面**：Envoy 代理自动应用配置

**不需要**直接调用 Envoy 的 SDK 或 xDS API。

## 使用方案 (Usage Options)

### 方案 0: 使用定制 Cilium Envoy（新方案）

如果您已经编译了包含 fault filter 的定制 Cilium Envoy：

**步骤**：
1. 编译 Cilium Envoy 并启用 fault extension
2. 将定制 image 部署为 DaemonSet
3. 使用 EnvoyChaos CRD 配合 CiliumEnvoyConfig

**示例配置**：
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-delay-custom-cilium
spec:
  selector:
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

**验证方法**：
```bash
# 检查 CiliumEnvoyConfig 是否创建
kubectl get ciliumenvoyconfigs -A

# 查看 Envoy 日志
kubectl logs -n <namespace> <cilium-envoy-pod>

# 测试故障注入效果
# 应该观察到 50% 的请求有 500ms 延迟
```

**参考**：
- [Custom Envoy Configurations in Cilium](https://medium.com/@samyak-devops/how-to-apply-custom-envoy-configurations-in-a-cilium-setup-with-rate-limiting-example-5301972460f2)
- [Cilium Envoy Configuration](https://docs.cilium.io/en/stable/network/servicemesh/envoy-traffic-management/)

## 替代方案 (Alternative Solutions)

如果不使用定制 Cilium Envoy，推荐以下替代方案：

### 方案 1: 使用 Istio EnvoyFilter

如果您的集群中已部署 Istio：

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

### 方案 2: 使用 HTTPChaos（最简单）

Chaos Mesh 的 HTTPChaos 已支持 gRPC 故障注入，无需 Envoy：

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

**优势**：
- 不依赖 Envoy
- 使用 tproxy 技术
- 支持 HTTP/1.1, HTTP/2 (gRPC)
- 已在生产环境验证

### 方案 3: 部署独立 Envoy

部署标准 Envoy 作为 sidecar 或 DaemonSet：

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
        # 配置 Envoy 监听和管理端口
```

## 推荐使用方式 (Recommendations)

1. **已有 Istio**：使用 Istio EnvoyFilter
2. **没有 Istio**：使用 HTTPChaos（支持 gRPC）
3. **需要 Envoy 特性**：部署独立 Envoy

## 架构设计 (Architecture Design)

### 整体架构

```
┌─────────────────────┐
│   EnvoyChaos CRD    │
│   (用户定义)         │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  EnvoyChaos         │
│  Controller         │
│  (Chaos Mesh)       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ CiliumEnvoyConfig   │
│     (自动创建)       │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   Envoy Proxy       │
│   (Cilium中)        │
│   + Fault Filter    │
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│   gRPC/HTTP         │
│   服务流量           │
└─────────────────────┘
```

### 工作流程

1. **创建混沌实验**：
   ```bash
   kubectl apply -f envoychaos-example.yaml
   ```

2. **控制器处理**：
   - EnvoyChaos Controller 监听 EnvoyChaos 资源
   - 根据 PodSelector 选择目标 Pods
   - 为每个选中的 Pod 创建 CiliumEnvoyConfig

3. **Envoy配置**：
   - Cilium 监听 CiliumEnvoyConfig 资源
   - 自动更新 Envoy 的配置
   - 应用 HTTP Fault Filter

4. **故障注入**：
   - Envoy 拦截匹配的请求
   - 根据配置注入延迟或错误
   - 返回修改后的响应

5. **清理恢复**：
   - 删除 EnvoyChaos 资源
   - 控制器自动删除 CiliumEnvoyConfig
   - Envoy 恢复正常配置

## 技术实现 (Technical Implementation)

### 关键组件

#### 1. EnvoyChaos CRD

定义在 `api/v1alpha1/envoychaos_types.go`，包含：

- **PodSelector**: 选择目标 Pods（继承自通用选择器）
- **Action**: 故障类型（delay, abort, fault）
- **Protocol**: 协议类型（grpc, http）
- **Delay Config**: 延迟配置（固定延迟，百分比）
- **Abort Config**: 中断配置（状态码，百分比）
- **Filters**: 请求过滤器（路径，方法，头部）

#### 2. EnvoyChaos Controller

实现在 `controllers/chaosimpl/envoychaos/impl.go`：

```go
// Apply - 应用混沌
func (impl *Impl) Apply(ctx context.Context, ...) {
    // 1. 通过 selector 选择目标 Pod
    // 2. 查找 Pod 所属的 Service（或使用 targetService 字段）
    // 3. 生成 Envoy Fault Filter 配置
    // 4. 创建 CiliumEnvoyConfig 资源，引用正确的 Service
}

// Recover - 恢复正常
func (impl *Impl) Recover(ctx context.Context, ...) {
    // 1. 删除 CiliumEnvoyConfig 资源
    // 2. 恢复 Envoy 原始配置
}
```

**Service 发现机制**：
- 如果指定了 `targetService` 字段，直接使用该 Service
- 如果未指定，控制器会自动查找匹配 Pod 标签的 Service
- Service 查找基于 Service selector 与 Pod labels 的匹配

#### 3. Envoy Fault Filter 配置生成

```go
func generateFaultConfig(envoychaos *v1alpha1.EnvoyChaos) {
    config := {
        "name": "envoy.filters.http.fault",
        "typedConfig": {
            "@type": "envoy.extensions.filters.http.fault.v3.HTTPFault",
            // 延迟配置
            "delay": {...},
            // 中断配置
            "abort": {...},
            // 匹配规则
            "headers": {...},
        }
    }
}
```

### 与 Cilium Envoy 的集成

#### CiliumEnvoyConfig 资源

控制器自动生成的 CiliumEnvoyConfig 结构：

```yaml
apiVersion: cilium.io/v2
kind: CiliumEnvoyConfig
metadata:
  name: chaos-{chaos-name}-{pod-name}
  namespace: {namespace}
  labels:
    chaos-mesh.org/injected: "true"
    chaos-mesh.org/chaos: {chaos-name}
spec:
  # 引用实际的 Kubernetes Service，而非 Pod
  services:
  - name: {service-name}  # 通过 targetService 指定或自动发现
    namespace: {service-namespace}
    ports: [50051]  # 可选：通过 targetPort 指定
  resources:
  - "@type": type.googleapis.com/envoy.config.listener.v3.Listener
    name: chaos-listener-{service-name}
    filterChains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typedConfig:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          statPrefix: chaos_http
          httpFilters:
          - name: envoy.filters.http.fault
            typedConfig:
              "@type": type.googleapis.com/envoy.extensions.filters.http.fault.v3.HTTPFault
              # 故障注入配置
              delay:
                fixedDelay: "500ms"
                percentage:
                  numerator: 50
                  denominator: HUNDRED
              abort:
                httpStatus: 503
                percentage:
                  numerator: 30
                  denominator: HUNDRED
          - name: envoy.filters.http.router
```

**重要说明**：
- `services` 字段必须引用实际的 Kubernetes Service，不能是 Pod 名称
- 如果未指定 `targetService`，控制器会自动查找匹配 Pod 的 Service
- `ports` 字段可选，用于限制只对特定端口应用故障注入

## 使用示例 (Usage Examples)

### 示例 1: gRPC 延迟注入

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-delay-test
  namespace: default
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-grpc-service
  mode: one
  protocol: grpc
  action: delay
  delay:
    fixedDelay: "500ms"
    percentage: 50
  method: "/myservice.MyService/MyMethod"
  duration: "60s"
```

### 示例 2: gRPC 错误注入

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: grpc-abort-test
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
    percentage: 30
  duration: "120s"
```

### 示例 3: HTTP 混合故障

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: EnvoyChaos
metadata:
  name: http-mixed-fault
  namespace: default
spec:
  selector:
    labelSelectors:
      app: my-http-service
  mode: fixed
  value: "2"
  protocol: http
  action: fault
  delay:
    fixedDelay: "200ms"
    percentage: 50
  abort:
    httpStatus: 503
    percentage: 20
  path: "/api/*"
  duration: "5m"
```

## 与其他 Chaos 类型的对比

| 特性 | EnvoyChaos | HTTPChaos | NetworkChaos |
|------|-----------|-----------|--------------|
| 层级 | L7 (应用层) | L7 (应用层) | L3/L4 (网络层) |
| 协议 | HTTP, gRPC | HTTP | TCP, UDP |
| 依赖 | Envoy Proxy | tproxy | iptables/tc |
| gRPC支持 | 优秀 | 有限 | 无 |
| 性能影响 | 低 | 中 | 低 |
| 细粒度控制 | 高 | 中 | 低 |
| 安装要求 | Cilium + Envoy | 无 | 无 |

## 优势 (Advantages)

1. **无需额外组件**：利用现有的 Cilium Envoy
2. **声明式配置**：通过 Kubernetes CRD 管理
3. **自动化**：控制器自动处理配置更新
4. **gRPC 原生支持**：直接支持 gRPC 状态码
5. **细粒度控制**：可以基于方法、路径、头部等过滤
6. **可观测性**：与 Envoy 的度量和追踪集成

## 局限性 (Limitations)

1. **依赖 Envoy**：需要 Cilium 或其他 Envoy 部署
2. **配置复杂度**：Envoy 配置较为复杂
3. **延迟**：配置更新可能有轻微延迟
4. **范围限制**：仅适用于通过 Envoy 代理的流量

## 故障排查 (Troubleshooting)

### 混沌不生效

1. 检查 Envoy 是否正在运行并支持 fault filter
2. 验证 CiliumEnvoyConfig 资源是否创建：
   ```bash
   kubectl get ciliumenvoyconfigs -A
   ```
3. 查看 CiliumEnvoyConfig 详情：
   ```bash
   kubectl describe ciliumenvoyconfig <config-name> -n <namespace>
   ```
4. 查看 Envoy 日志：
   ```bash
   kubectl logs -n <namespace> <envoy-pod> -c cilium-envoy
   ```
5. 检查 Envoy 是否加载了 fault filter：
   ```bash
   # 通过 admin interface 检查配置
   kubectl port-forward -n <namespace> <envoy-pod> 15000:15000
   curl localhost:15000/config_dump | grep -A 20 "fault"
   ```

### CiliumEnvoyConfig 不工作

**可能原因**：
1. **Envoy 不支持 fault filter**：确认使用的是包含 fault extension 的定制 image
2. **Service 引用错误**：检查 `spec.services` 是否引用了正确的 Service
3. **资源格式错误**：验证 CiliumEnvoyConfig 的 `spec.resources` 格式

**解决方法**：
```bash
# 1. 确认 Envoy image 包含 fault filter
kubectl get daemonset <envoy-daemonset> -o yaml | grep image

# 2. 手动测试 CiliumEnvoyConfig
kubectl apply -f - <<EOF
apiVersion: cilium.io/v2
kind: CiliumEnvoyConfig
metadata:
  name: test-fault-filter
spec:
  services:
  - name: my-service
    namespace: default
  resources:
  - "@type": type.googleapis.com/envoy.config.listener.v3.Listener
    name: test-listener
    # ... fault filter configuration
EOF

# 3. 查看 Envoy 是否接受配置
kubectl logs -n <namespace> <envoy-pod> | grep -i "fault\|error"
```

### gRPC 状态码无效

- 确保使用有效的 gRPC 状态码（0-16）
- 对于 HTTP 流量，使用 `httpStatus` 而非 `grpcStatus`
- 检查 `protocol` 字段是否设置为 "grpc"

### 配置未应用

1. 检查 EnvoyChaos 资源状态：
   ```bash
   kubectl describe envoychaos <name>
   ```
2. 查看控制器日志：
   ```bash
   kubectl logs -n chaos-mesh <controller-manager-pod>
   ```

## 已知限制 (Known Limitations)

### Cilium Envoy 不支持

⚠️ **重要**: Cilium 的 Envoy proxy 是定制版本，**不包含 HTTP fault filter 功能**。

**原因**：
- Cilium Envoy 是为网络层面的 L7 可见性和策略而定制的
- 移除了许多标准 Envoy 的 HTTP filters 以减小体积
- 参见 GitHub issue: [cilium/proxy#62](https://github.com/cilium/proxy/issues/62)

**影响**：
- EnvoyChaos 不能直接与 Cilium Envoy 配合使用
- 尝试创建 CiliumEnvoyConfig 会失败或被忽略

**解决方案**：
1. 使用 HTTPChaos（推荐，无需 Envoy）
2. 部署 Istio 并使用 Istio EnvoyFilter
3. 部署独立的标准 Envoy

### 其他限制

1. **依赖 Envoy**: 需要完整版 Envoy (非 Cilium 版本)
2. **配置复杂度**: Envoy 配置较为复杂
3. **延迟**: 配置更新可能有轻微延迟
4. **范围限制**: 仅适用于通过 Envoy 代理的流量

## 参考资料 (References)

- [Envoy Fault Filter 文档](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/fault_filter)
- [Envoy Gateway Fault Injection](https://gateway.envoyproxy.io/docs/tasks/traffic/fault-injection/)
- [Cilium L7 协议可见性](https://docs.cilium.io/en/stable/gettingstarted/http/)
- [gRPC 状态码](https://grpc.github.io/grpc/core/md_doc_statuscodes.html)
- [CiliumEnvoyConfig CRD](https://docs.cilium.io/en/stable/network/servicemesh/envoy-traffic-management/)

## 贡献 (Contributing)

欢迎贡献！请参阅 [Chaos Mesh 贡献指南](../CONTRIBUTING.md)。

## 许可证 (License)

EnvoyChaos 是 Chaos Mesh 的一部分，采用 Apache License 2.0 许可。
