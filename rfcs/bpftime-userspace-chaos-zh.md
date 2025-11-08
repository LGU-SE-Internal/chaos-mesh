# RFC: 基于 BPFTime 的用户态故障注入

[English Version](./bpftime-userspace-chaos.md)

## 概述

本 RFC 提议在 Chaos Mesh 中添加基于 [bpftime](https://github.com/eunomia-bpf/bpftime) 用户态 eBPF 运行时的新型故障注入能力。通过利用 bpftime 无需内核权限即可钩取和插桩用户态函数的能力，我们可以引入新的混沌类型，实现之前无法实现或不切实际的故障注入场景。

## 核心内容

### 新混沌类型：UserspaceChaos

提议引入新的 CRD `UserspaceChaos`，利用 bpftime 对用户态应用程序进行故障注入。

### bpftime 核心能力

基于 [bpftime 官方文档](https://github.com/eunomia-bpf/bpftime)和[技术论文](https://arxiv.org/abs/2311.07923)，bpftime 提供以下支持故障注入的核心能力：

1. **Uprobe/Uretprobe 支持** - 钩取函数入口和出口点
2. **系统调用跟踪** - 拦截和修改系统调用
3. **eBPF Maps** - 在 eBPF 程序和用户态之间共享状态
4. **JIT 编译** - LLVM 基于的即时编译，性能开销 <5%
5. **共享内存通信** - 高效的进程间通信

### 如何寻找和指定 Hook 点

RFC 新增了详细的技术指南，说明如何发现可钩取的函数：

#### 符号发现方法

**动态链接库：**
```bash
# 列出所有导出符号
nm -D /lib/x86_64-linux-gnu/libc.so.6 | grep " T "

# 使用 readelf 查看详细符号信息
readelf -s /lib/x86_64-linux-gnu/libc.so.6 | grep FUNC
```

**C++ 应用（处理 mangled names）：**
```bash
# 使用 c++filt 解码函数名
nm /usr/bin/myapp | c++filt | grep "processRequest"
```

#### Hook 点解析顺序

1. **库函数**（最高优先级）：通过 dlopen() 和 dlsym() 解析
2. **二进制函数**：从指定二进制的符号表解析
3. **主二进制**：目标进程的主可执行文件

### 配置文件设计

RFC 包含了完整的配置架构设计，包括：

- **目标选择器**：命名空间、标签、注解、Pod 阶段
- **容器定位**：指定容器名称
- **执行模式**：one, all, fixed, fixed-percent, random-max-percent
- **高级过滤**：基于参数值和调用栈的过滤
- **性能调优**：最大事件数、冷却时间

**完整配置示例：**
```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: advanced-fault-injection
spec:
  selector:
    labelSelectors:
      app: payment-service
  mode: one
  duration: "5m"
  functionHook:
    function: "malloc"
    library: "libc.so.6"
    action: fail
    probability: 15
    returnValue: "0"
    errno: "ENOMEM"
    filter:
      arguments:
        - index: 0  # 只影响大于 1MB 的分配
          greaterThan: 1048576
      callStack:
        - "largeAllocation"  # 只在特定调用栈中注入
```

### 交互模式

1. **直接模式** - 立即生效
2. **定时模式** - 周期性执行（使用 cron）
3. **条件模式** - 基于指标触发（未来增强）

### 主要应用场景

本 RFC 提出了 **8 个主要的故障注入场景**：

#### 1. 内存分配失败
- 测试应用程序如何处理 `malloc()` 或 `new` 操作符失败
- 验证内存耗尽时的优雅降级
- **技术实现**：使用 uprobe 钩取 malloc，通过 `bpf_override_return()` 返回 NULL

#### 2. 文件 I/O 函数失败
- 在 libc 层面模拟文件操作失败
- **技术实现**：uprobe + 参数读取 (`bpf_probe_read_user_str`)，支持路径过滤

#### 3. 网络库函数延迟
- 向特定网络库调用注入延迟
- **技术实现**：uprobe + userspace sleep，JIT 编译保证低开销

#### 4. 线程和同步失败
- 通过在 pthread 操作中注入故障来测试并发代码
- **技术实现**：uretprobe 修改返回值寄存器（RAX）

#### 5. 自定义应用程序函数钩取
- 钩取应用程序二进制文件中的特定函数
- 支持 C++ mangled names

#### 6. SSL/TLS 库失败
- 向 OpenSSL/TLS 库函数注入故障
- 测试证书验证错误处理

#### 7. 数据库客户端库失败
- 向数据库客户端库注入故障（MySQL、PostgreSQL、Redis 客户端）

#### 8. 随机数生成操作
- 控制随机性以进行确定性测试

### bpftime 能力与故障场景映射表

| 故障场景 | 使用的 bpftime 能力 | 实现方法 | 参考文档 |
|---------|-------------------|---------|---------|
| 内存分配失败 | Uprobe on malloc | Hook 入口，检查大小，返回 NULL | [uprobe.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md) |
| 文件 I/O 失败 | Uprobe + 参数读取 | Hook 入口，检查路径过滤，返回错误 | [malloc example](https://github.com/eunomia-bpf/bpftime/tree/main/example/malloc) |
| 网络延迟 | Uprobe + sleep | Hook 入口，sleep N 毫秒，继续 | [syscall-tracing.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/syscall-tracing.md) |
| 线程失败 | Uretprobe | Hook 出口，修改返回值为 EAGAIN | [uprobe.md](https://github.com/eunomia-bpf/bpftime/blob/main/docs/uprobe.md) |
| 调用栈过滤 | Stack unwinding | 使用 `bpf_get_stackid()` helper | [eBPF helpers](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html) |
- 测试连接池耗尽处理
- 验证查询超时和重试机制

#### 8. 随机数生成操作
- 控制随机性以进行确定性测试
- 测试依赖随机性的代码
- 验证洗牌/选择算法

### 多语言支持

RFC 详细分析了不同语言类型的支持策略：

#### 语言支持分类

1. **编译型语言（原生二进制）** - ✅ **完全支持**
   - C, C++, Rust, Go
   - 直接钩取原生函数符号
   - 最佳性能，最可靠

2. **虚拟机语言（JIT 编译）** - ⚠️ **部分支持**
   - Java (JVM): 推荐使用现有的 JVMChaos
   - C# (.NET): 需要 CLR profiler 适配器
   - Node.js (V8): 可钩取 V8 内部函数，但受限
   - 建议：使用专门的语言特定混沌类型

3. **解释型语言** - ⚠️ **有限支持**
   - Python: 只能钩取 CPython C API 和原生扩展
   - Ruby: 只能钩取 Ruby C API
   - 无法直接钩取纯 Python/Ruby 函数

#### 语言支持矩阵

| 语言 | 支持级别 | Hook 目标 | 示例函数 |
|------|---------|-----------|---------|
| C/C++ | ✅ 完全 | 原生函数 | `malloc`, `fopen` |
| Rust | ✅ 完全 | 原生函数 | `alloc::vec::Vec::push` |
| Go | ✅ 完全 | 原生函数 | `main.ProcessRequest` |
| Java | ⚠️ 部分 | JVM 内部/JNI | 使用 JVMChaos 更好 |
| Node.js | ⚠️ 部分 | V8 内部 | `v8::internal::Runtime_*` |
| Python | ⚠️ 有限 | CPython C API | `PyFile_OpenCode` |
| C#/.NET | ⚠️ 部分 | CLR 内部 | 需要 profiler API |

#### 推荐方法

- **编译型语言**: 使用 UserspaceChaos（本 RFC）
- **JVM 应用**: 使用 [JVMChaos](https://chaos-mesh.org/docs/simulate-jvm-application-chaos/)
- **Python/Node.js**: 钩取解释器原生库（有限）或使用应用级混沌库
- **.NET**: 等待专用 .NETChaos 类型（未来增强）

详见英文完整版的 "Multi-Language Support Considerations" 章节。

## 技术优势

### 相比现有方案的优势

1. **无需内核修改**
   - 与 KernelChaos 不同，不需要内核模块支持
   - 适用于任何 Linux 内核版本
   - 无内核不稳定风险

2. **应用级精度**
   - 可以针对特定二进制文件中的特定函数
   - 根据参数值进行过滤
   - 比网络或 IO 混沌更精细

3. **性能**
   - 比基于内核的跟踪开销更低
   - eBPF 程序的 JIT 编译
   - 对非钩取代码路径影响最小

4. **灵活性**
   - 可以钩取任何用户态函数
   - 支持无需源代码的编译二进制文件
   - 适用于 C、C++、Go、Rust 应用程序

## 架构设计

### API 规范示例

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: UserspaceChaos
metadata:
  name: malloc-failure
spec:
  selector:
    namespaces:
      - default
    labelSelectors:
      app: my-service
  mode: one
  duration: "30s"
  functionHook:
    function: "malloc"
    library: "libc.so.6"
    action: fail
    probability: 10  # 10% 失败率
    returnValue: "0"  # NULL 指针
```

## 实施计划

实施分为 4 个阶段，预计 12 周：

1. **第 1 阶段：核心基础设施**（第 1-3 周）
   - bpftime 集成
   - CRD 和 API 定义
   - 基本控制器实现

2. **第 2 阶段：故障注入能力**（第 4-6 周）
   - eBPF 程序模板
   - Daemon 集成
   - 测试框架

3. **第 3 阶段：高级功能**（第 7-9 周）
   - 过滤和条件
   - 特定库辅助功能
   - 可观测性

4. **第 4 阶段：文档和稳定化**（第 10-12 周）
   - 文档编写
   - 性能优化
   - 安全加固

## 成功指标

1. **功能性**
   - 支持钩取至少 20 个常见 libc 函数
   - 成功在 3 种以上不同应用类型中注入故障（C、C++、Go）
   - 误报率 <5%

2. **性能**
   - 钩取函数性能开销 <10%
   - 非钩取代码路径开销 <1%
   - bpftime 注入在 5 秒内完成

3. **可用性**
   - 文档涵盖 10 个以上实际场景
   - 90% 以上的用户可以在第一次尝试中成功创建 UserspaceChaos
   - 配置错误时提供清晰的错误信息

4. **可靠性**
   - 99.9% 成功恢复（无挂起进程）
   - 零内核崩溃或系统崩溃
   - 混沌结束后所有资源正确清理

## 安全考虑

1. **权限要求**
   - bpftime 在用户态运行但需要 `CAP_SYS_PTRACE` 来附加到进程
   - 某些系统调用拦截场景可能需要 `CAP_SYS_ADMIN`
   - 应记录所需的最小权限能力

2. **隔离**
   - 注入的故障隔离到目标容器
   - 对同一节点上的其他 pod 无影响
   - 在混沌恢复时正确清理

3. **访问控制**
   - RBAC 策略控制谁可以创建 UserspaceChaos
   - 强制执行命名空间隔离
   - 所有混沌操作的审计跟踪

## 参考资料

1. [bpftime GitHub 仓库](https://github.com/eunomia-bpf/bpftime)
2. [eBPF 文档](https://ebpf.io/)
3. [Linux Uprobe 文档](https://www.kernel.org/doc/html/latest/trace/uprobetracer.html)
4. [Chaos Mesh 文档](https://chaos-mesh.org/)
5. [KernelChaos 设计](https://chaos-mesh.org/docs/simulate-kernel-chaos/)

## 结论

通过 bpftime 集成向 Chaos Mesh 添加 UserspaceChaos 将显著扩展故障注入能力，实现之前困难或不可能的应用级混沌工程。这将帮助用户：

- 更彻底地测试应用程序中的错误处理
- 验证第三方库集成的弹性
- 在函数调用级别模拟真实世界的故障
- 通过更全面的测试提高整体系统可靠性

完整的技术细节、示例配置和实现计划请参阅[英文版 RFC 文档](./bpftime-userspace-chaos.md)。
