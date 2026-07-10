#!/usr/bin/env python3
"""检查存储层是否把 SQL 硬编码在 Go 代码里。

事实源规则见 docs/architecture/go-service-design.md「数据访问」段：
PostgreSQL repository 的 SQL 默认外置到
services/<service>/internal/<domain>/infrastructure/postgres/sql/*.sql，
由同包 sql.go 用 //go:embed 加载为包级变量；Go 代码只保留事务编排、
参数映射、Scan/Exec 调用和错误翻译。

本脚本只覆盖 postgres 存储层的 .go 源文件（排除 *_test.go），用两个
精确信号判定硬编码，避免把 fmt.Errorf 之类的错误串误判为 SQL：

  A. 调用点字面量：db.Query/Exec/QueryRow(Context)? 的查询参数位直接
     写字符串或反引号字面量，而不是传入 mustSQL 加载的包级变量。
  B. 反引号 SQL 块：文件里出现含 SQL 动词的反引号原始字面量，通常是
     被内联的多行 SQL。

唯一豁免是极简、不承载业务语义、不会成为稳定契约的一次性语句
（例如 SELECT 1 健康检查、TRUNCATE、单表按主键读写）。这类语句必须在
命中行同行或紧邻上一行显式标注：

    // inline-sql-allow: <为什么可以内联的理由>

标记必须带非空理由，逼迫写清豁免依据，不靠脚本猜 SQL 语义。
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

# 只扫描 postgres 存储层实现文件。
POSTGRES_GLOB = "services/*/internal/*/infrastructure/postgres/**/*.go"

# 显式豁免标记，必须带非空理由。
ALLOW_MARKER = re.compile(r"//\s*inline-sql-allow:\s*\S")

# 信号 A：Context 变体，查询位是字面量。
# 形态固定为 method(ctx, <query>, args...)，第一参数是标识符（ctx），
# 第二参数若是字符串/反引号字面量即为硬编码。
CALL_CTX_LITERAL = re.compile(
    r"\.(?:QueryContext|QueryRowContext|ExecContext)\("
    r"\s*[A-Za-z_][\w.]*\s*,\s*"
    r'(?P<lit>["`])'
)

# 信号 A：非 Context 变体，查询位是第一个参数。
CALL_PLAIN_LITERAL = re.compile(
    r"\.(?:Query|QueryRow|Exec)\("
    r'\s*(?P<lit>["`])'
)

# 信号 B：反引号原始字面量里出现 SQL 动词。
SQL_VERB = re.compile(r"\b(?:SELECT|INSERT|UPDATE|DELETE|WITH|MERGE|TRUNCATE)\b")


def iter_target_files(root: Path):
    """列出 postgres 存储层的非 test .go 文件。"""
    for path in sorted(root.glob(POSTGRES_GLOB)):
        if path.name.endswith("_test.go"):
            continue
        if not path.is_file():
            continue
        yield path


def line_is_exempt(lines: list[str], idx: int) -> bool:
    """命中行同行或紧邻上一行带合法豁免标记时放行。"""
    if ALLOW_MARKER.search(lines[idx]):
        return True
    if idx > 0 and ALLOW_MARKER.search(lines[idx - 1]):
        return True
    return False


def check_call_site_literals(rel: str, lines: list[str]) -> list[str]:
    """信号 A：调用点查询位直接写字面量。"""
    findings: list[str] = []
    for idx, line in enumerate(lines):
        if CALL_CTX_LITERAL.search(line) or CALL_PLAIN_LITERAL.search(line):
            if line_is_exempt(lines, idx):
                continue
            findings.append(
                f"{rel}:{idx + 1}: SQL 硬编码在调用点，"
                f"查询参数应传 mustSQL 加载的包级变量，不写字面量"
            )
    return findings


def check_backtick_sql_blocks(rel: str, lines: list[str], text: str) -> list[str]:
    """信号 B：反引号原始字面量里出现 SQL 动词。"""
    findings: list[str] = []
    # Go 反引号字符串不支持转义，成对出现即可切分。
    parts = text.split("`")
    # split 后奇数下标是反引号内部内容。
    offset_line = 1
    consumed = 0
    for i, part in enumerate(parts):
        start_line = offset_line
        consumed += part.count("\n")
        offset_line = 1 + consumed
        if i % 2 == 1 and SQL_VERB.search(part):
            # 反引号块起始行同行或紧邻上一行带豁免标记时放行；
            # 起始行下标为 start_line - 1，与 line_is_exempt 的 0 基下标对齐。
            if line_is_exempt(lines, start_line - 1):
                continue
            findings.append(
                f"{rel}:{start_line}: 反引号里内联了 SQL，"
                f"应外置到 postgres/sql/*.sql 并用 //go:embed 加载；"
                f"确属极简一次性语句时在块起始行或上一行加 // inline-sql-allow: <理由>"
            )
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root",
        default=".",
        help="仓库根目录，默认当前目录",
    )
    args = parser.parse_args()

    root = Path(args.root).resolve()
    if not root.is_dir():
        print(f"root not a directory: {root}", file=sys.stderr)
        return 2

    findings: list[str] = []
    scanned = 0
    for path in iter_target_files(root):
        scanned += 1
        rel = path.relative_to(root).as_posix()
        text = path.read_text(encoding="utf-8")
        lines = text.splitlines()
        findings.extend(check_call_site_literals(rel, lines))
        findings.extend(check_backtick_sql_blocks(rel, lines, text))

    if findings:
        print("发现存储层硬编码 SQL：", file=sys.stderr)
        for item in findings:
            print(f"  {item}", file=sys.stderr)
        print(
            "\n把 SQL 外置到 postgres/sql/*.sql 并用 //go:embed 加载；"
            "确属极简一次性语句时，在命中行或上一行加"
            " `// inline-sql-allow: <理由>`。",
            file=sys.stderr,
        )
        return 1

    print(f"inline-sql ok (scanned {scanned} postgres source files)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
