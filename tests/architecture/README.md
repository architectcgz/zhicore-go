# 架构测试

这里放源码级边界检查，用于防止服务迁移过程中破坏架构约束。

典型检查包括：

- 服务之间不得导入彼此的 `internal`
- `libs/kit` 不得依赖服务私有包
- contract 不得依赖服务私有模型
- handler、application、domain、infrastructure 的依赖方向符合项目约定

## 当前检查

全局依赖方向检查由 `check_boundaries.py` 执行：

```bash
python3 tests/architecture/check_boundaries.py --root .
```

`make check` 会自动运行该检查。当前规则覆盖：

- 服务之间禁止导入彼此的 `internal` 包。
- `libs/kit` 和 `libs/contracts` 禁止导入 `services/*`。
- 服务内生产代码按允许矩阵检查依赖方向：
  - `api/http` 只允许导入本服务 `application`。
  - `application` 只允许导入本服务 `domain` 和 `ports`。
  - `domain` 默认不允许导入本服务任何层；领域类型优先保持在同一个 Go package 内。需要 `domain/shared` 等子包时，必须补服务级专项规则。
  - `ports` 只允许导入本服务 `domain` 和 `ports`。
  - `infrastructure` 允许导入本服务 `application`、`domain`、`ports` 和 `infrastructure`。
  - `runtime` 允许导入本服务 `api/http`、`application`、`domain`、`ports`、`infrastructure` 和 `runtime`。
  - `cmd/server` 只允许导入本服务 `runtime`。

检查器跳过 `*_test.go`，测试代码可以根据测试边界导入更宽的包；生产依赖方向由非测试 Go 文件守住。

新增服务不需要复制架构测试；检查器会动态扫描 `services/*`。只有服务存在独有边界时，才在本目录补充服务级专项检查。
