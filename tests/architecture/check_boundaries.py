#!/usr/bin/env python3
"""Check repository-wide Go import boundaries."""

from __future__ import annotations

import argparse
import dataclasses
import re
import sys
from pathlib import Path


MODULE_PREFIX = "github.com/architectcgz/zhicore-go/"
SERVICE_PREFIX = MODULE_PREFIX + "services/"

# The matrix is intentionally expressed as "allowed same-service layer imports"
# instead of a list of forbidden reverse edges. That makes forward dependencies
# explicit and fails closed when a new layer relation appears without a design
# decision. Non-service imports and cross-service public package imports are
# handled outside this matrix.
ALLOWED_SERVICE_LAYER_IMPORTS = {
    "api/http": {"application"},
    "application": {"domain", "ports"},
    # Domain is intentionally closed by default. If a service later needs
    # domain subpackages such as domain/shared, add a service-specific rule
    # instead of allowing every domain package to depend on every other one.
    "domain": set(),
    "ports": {"domain", "ports"},
    "infrastructure": {"application", "domain", "ports", "infrastructure"},
    "runtime": {"api/http", "application", "domain", "ports", "infrastructure", "runtime"},
    "cmd/server": {"runtime"},
}


@dataclasses.dataclass(frozen=True)
class GoFile:
    path: Path
    imports: list[str]
    source: str


@dataclasses.dataclass(frozen=True)
class GoImport:
    path: str
    alias: str | None


@dataclasses.dataclass(frozen=True)
class Violation:
    rule: str
    path: Path
    imported: str
    message: str

    def __str__(self) -> str:
        return f"{self.path}: {self.rule}: {self.message}; import={self.imported}"


def parse_go_imports(source: str) -> list[str]:
    return [item.path for item in parse_go_import_specs(source)]


def parse_go_import_specs(source: str) -> list[GoImport]:
    imports: list[GoImport] = []
    lines = source.splitlines()
    index = 0
    while index < len(lines):
        line = lines[index].strip()
        if line.startswith("import "):
            rest = line[len("import ") :].strip()
            if rest == "(":
                index += 1
                while index < len(lines):
                    group_line = lines[index].strip()
                    if group_line == ")":
                        break
                    import_spec = extract_import_spec(group_line)
                    if import_spec:
                        imports.append(import_spec)
                    index += 1
            else:
                import_spec = extract_import_spec(rest)
                if import_spec:
                    imports.append(import_spec)
        index += 1
    return imports


def extract_import_path(line: str) -> str | None:
    import_spec = extract_import_spec(line)
    if import_spec:
        return import_spec.path
    return None


def extract_import_spec(line: str) -> GoImport | None:
    line = line.split("//", 1)[0].strip()
    if not line:
        return None
    match = re.match(r'(?:(?P<alias>[A-Za-z_][A-Za-z0-9_]*|\.|_)\s+)?(?P<quote>"[^"]+")', line)
    if match:
        return GoImport(path=match.group("quote").strip('"'), alias=match.group("alias"))
    return None


def check_root(root: Path) -> list[Violation]:
    return check_files(discover_go_files(root))


def discover_go_files(root: Path) -> list[GoFile]:
    files: list[GoFile] = []
    for path in sorted(root.rglob("*.go")):
        if should_skip(path):
            continue
        rel_path = path.relative_to(root)
        source = path.read_text(encoding="utf-8")
        files.append(GoFile(rel_path, parse_go_imports(source), source))
    return files


def should_skip(path: Path) -> bool:
    ignored_parts = {".git", ".worktrees", "vendor"}
    # Test files often import wider internals to build fakes and fixtures; this
    # check protects production dependency direction only.
    return path.name.endswith("_test.go") or any(part in ignored_parts for part in path.parts)


def check_files(files: list[GoFile]) -> list[Violation]:
    violations: list[Violation] = []
    for go_file in files:
        for imported in go_file.imports:
            violations.extend(check_import(go_file.path, imported))
        violations.extend(check_source(go_file.path, go_file.source))
    return violations


def check_source(path: Path, source: str) -> list[Violation]:
    violations: list[Violation] = []
    importer = service_ref_from_path(path)
    if importer and importer.layer == "application":
        violations.extend(check_application_domain_exported_aliases(path, source))
    return violations


def check_application_domain_exported_aliases(path: Path, source: str) -> list[Violation]:
    violations: list[Violation] = []
    domain_selectors = application_domain_import_selectors(path, source)
    for line in iter_type_alias_lines(source):
        match = re.match(r"type\s+([A-Z][A-Za-z0-9_]*)\s*=\s*([A-Za-z_][A-Za-z0-9_]*)\.[A-Za-z0-9_]+(?:\s|$)", line)
        if match and match.group(2) in domain_selectors:
            violations.append(
                violation(
                    "application-domain-exported-alias-not-allowed",
                    path,
                    line,
                    f"application must not re-export domain type alias {match.group(1)}",
                )
            )
    return violations


def application_domain_import_selectors(path: Path, source: str) -> set[str]:
    importer = service_ref_from_path(path)
    if importer is None:
        return set()

    selectors: set[str] = set()
    for item in parse_go_import_specs(source):
        imported_ref = service_ref_from_import(item.path)
        if imported_ref is None or imported_ref.service != importer.service or imported_ref.layer != "domain":
            continue
        if item.alias in {".", "_"}:
            continue
        selectors.add(item.alias or item.path.rsplit("/", 1)[-1])
    return selectors


def iter_type_alias_lines(source: str) -> list[str]:
    lines: list[str] = []
    in_type_group = False
    for raw_line in source.splitlines():
        line = raw_line.split("//", 1)[0].strip()
        if not line:
            continue
        if in_type_group:
            if line == ")":
                in_type_group = False
                continue
            lines.append("type " + line)
            continue
        if line == "type (":
            in_type_group = True
            continue
        if line.startswith("type "):
            lines.append(line)
    return lines


def check_import(path: Path, imported: str) -> list[Violation]:
    violations: list[Violation] = []
    importer = service_ref_from_path(path)
    imported_ref = service_ref_from_import(imported)
    importer_service = service_name_from_path(path)
    imported_service = service_name_from_import(imported)

    if importer_service and imported_service and importer_service != imported_service and "/internal/" in imported:
        violations.append(
            violation(
                "cross-service-internal-import",
                path,
                imported,
                "services must not import another service's internal package",
            )
        )

    if is_shared_library_path(path) and imported.startswith(SERVICE_PREFIX):
        violations.append(
            violation(
                "shared-library-imports-service",
                path,
                imported,
                "libs must not depend on service packages",
            )
        )

    if importer and imported_ref and importer.service == imported_ref.service:
        allowed_imports = ALLOWED_SERVICE_LAYER_IMPORTS.get(importer.layer)
        if allowed_imports is not None and imported_ref.layer not in allowed_imports:
            violations.append(
                violation(
                    "service-layer-import-not-allowed",
                    path,
                    imported,
                    f"{importer.layer} may import {sorted(allowed_imports)}, not {imported_ref.layer}",
                )
            )

    return violations


@dataclasses.dataclass(frozen=True)
class ServiceRef:
    service: str
    layer: str


def service_ref_from_path(path: Path) -> ServiceRef | None:
    service = service_name_from_path(path)
    layer = layer_from_path(path)
    if not service or not layer:
        return None
    return ServiceRef(service, layer)


def service_ref_from_import(imported: str) -> ServiceRef | None:
    service = service_name_from_import(imported)
    layer = layer_from_import(imported)
    if not service or not layer:
        return None
    return ServiceRef(service, layer)


def service_name_from_path(path: Path) -> str | None:
    parts = path.parts
    if len(parts) >= 2 and parts[0] == "services":
        return parts[1]
    return None


def service_name_from_import(imported: str) -> str | None:
    if not imported.startswith(SERVICE_PREFIX):
        return None
    rest = imported[len(SERVICE_PREFIX) :]
    return rest.split("/", 1)[0]


def is_shared_library_path(path: Path) -> bool:
    parts = path.parts
    return len(parts) >= 2 and parts[0] == "libs" and parts[1] in {"kit", "contracts"}


def layer_from_path(path: Path) -> str | None:
    parts = path.parts
    if len(parts) >= 4 and parts[0] == "services" and parts[2:4] == ("api", "http"):
        return "api/http"
    if len(parts) >= 5 and parts[0] == "services" and parts[2:4] == ("cmd", "server"):
        return "cmd/server"
    if len(parts) >= 6 and parts[0] == "services" and parts[2] == "internal":
        return parts[4]
    return None


def layer_from_import(imported: str) -> str | None:
    parts = imported.split("/")
    try:
        services_index = parts.index("services")
    except ValueError:
        return None
    if len(parts) <= services_index + 2:
        return None
    after_service = parts[services_index + 2 :]
    if len(after_service) >= 2 and after_service[:2] == ["api", "http"]:
        return "api/http"
    if len(after_service) >= 2 and after_service[:2] == ["cmd", "server"]:
        return "cmd/server"
    if len(after_service) >= 3 and after_service[0] == "internal":
        return after_service[2]
    return None


def layer_from_internal_import(imported: str) -> str | None:
    parts = imported.split("/")
    try:
        internal_index = parts.index("internal")
    except ValueError:
        return None
    if len(parts) <= internal_index + 2:
        return None
    return parts[internal_index + 2]


def violation(rule: str, path: Path, imported: str, message: str) -> Violation:
    return Violation(rule=rule, path=path, imported=imported, message=message)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", type=Path, help="repository root")
    args = parser.parse_args(argv)

    violations = check_root(args.root.resolve())
    if violations:
        for item in violations:
            print(item, file=sys.stderr)
        return 1
    print("architecture boundaries ok")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
