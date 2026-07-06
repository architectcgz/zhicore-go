# Application Service 拆分复核

## Review 对象

- 分支：`task/2026-07-06-application-service-split`
- 基线：`master`
- 提交：
  - `5721754 refactor(comment): 拆分评论应用服务文件`
  - `ca68383 refactor(user): 拆分用户应用服务文件`
  - `4346968 refactor(content): 拆分内容应用服务文件`
- Diff 范围：仅 `services/zhicore-comment/internal/comment/application`、`services/zhicore-user/internal/user/application`、`services/zhicore-content/internal/content/application`。

## 分类判断

结构性 R1 机械重构。目标是把三个超长 application `service.go` 按 use case 拆成更小文件，保持同一 `application` package，不新增 service 层，不改变导出方法签名、HTTP contract、事件 contract、ports 或 domain 类型。

## Gate Verdict

`pass`

## Findings

### Blocker

未发现 blocker。

### Major

未发现 major finding。独立 reviewer 确认未发现行为变化、导出 API / DTO / 事件 payload / HTTP contract 变化，也未发现新增 package、依赖方向倒置、重复定义或遗漏 import。

### Minor

未发现 minor finding。

### Note

- `master...HEAD` 只改了三个服务的 `internal/*/application` 包；`api/http`、`libs/contracts`、`libs/kit`、`domain`、`ports`、`runtime`、`infrastructure`、`migrations` 没有 diff。
- 导出 `Service` 方法签名已对比，Comment / User / Content 三个 application 包的方法集合保持一致，只改变文件位置。
- 文件归属符合计划：Comment 按 create / delete / like / list / mapper / pagination / events 拆分；User 按 profile / relationship / cache / events 拆分；Content 按 create / draft body / publish / published body / body validation / events 拆分。
- `services/zhicore-content/internal/content/application/author_workbench.go` 仍为 423 行，但该文件不在本次 diff 内，且计划明确不拆。

## Material Findings

无。

## 验证证据

实现过程执行并通过：

```bash
cd services/zhicore-comment && go test ./internal/comment/application
cd services/zhicore-user && go test ./internal/user/application
cd services/zhicore-content && go test ./internal/content/application
python3 tests/architecture/check_boundaries.py --root .
make test-size
make check
```

独立 reviewer 额外执行并通过：

```bash
PYTHONDONTWRITEBYTECODE=1 python3 tests/architecture/check_boundaries.py --root .
PYTHONDONTWRITEBYTECODE=1 make test-size
PYTHONDONTWRITEBYTECODE=1 make check
go test -count=1 ./internal/comment/application
go test -count=1 ./internal/user/application
go test -count=1 ./internal/content/application
```

## Required Re-validation

无 material finding。若后续继续移动 application 公开 DTO、事件构造或 HTTP handler 调用点，应重新运行对应服务 application 测试、`python3 tests/architecture/check_boundaries.py --root .` 和 `make check`。

## Residual Risk

- 本次未新增测试，原因是改动为 R1 机械拆文件，行为由现有 application 包测试和 `make check` 覆盖。
- 全量 `make check` 中部分 Go 包命中 cache；独立 reviewer 已对三个被拆 application 包使用 `go test -count=1` 强制重跑。

## 技术债状态

未触达已登记技术债。`author_workbench.go` 仍超过 400 行，但不属于本次拆分范围，未新增或扩大该 surface。
