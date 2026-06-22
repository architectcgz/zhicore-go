# 测试目录

测试策略、写法规范、规模控制、风险分级和验证命令选择见 `docs/architecture/testing.md`。

测试放置规则：

- `tests/architecture`：源码级架构边界检查。
- `tests/system/http`：面向已迁移服务的黑盒 HTTP 场景。
- `tests/runtime`：需要真实基础设施、容器、端口或外部进程的集成测试。
- `tests/testkit`：黑盒测试可复用 fixture、builder 和断言。

服务内部行为优先放在对应服务模块内的 package-local `*_test.go`。只有跨服务、黑盒或架构级测试才放到这里。
