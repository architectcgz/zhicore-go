#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
SKILL_DIR = SCRIPT_DIR.parent
ASSETS_DIR = SKILL_DIR / "assets"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="List or apply project-template starter assets."
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List available templates and exit.",
    )
    parser.add_argument(
        "--template",
        help="Template path relative to assets/, for example backend/go-backend-onion-template",
    )
    parser.add_argument(
        "--dest",
        help="Destination directory to write rendered starter files into.",
    )
    parser.add_argument(
        "--var",
        action="append",
        default=[],
        metavar="KEY=VALUE",
        help="Placeholder assignment, repeatable.",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Allow overwriting existing files.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Print planned outputs without writing files.",
    )
    return parser.parse_args()


def list_templates() -> int:
    manifests = sorted(ASSETS_DIR.glob("**/manifest.json"))
    if not manifests:
        print("No templates found.", file=sys.stderr)
        return 1

    for manifest_path in manifests:
        rel = manifest_path.parent.relative_to(ASSETS_DIR).as_posix()
        data = json.loads(manifest_path.read_text(encoding="utf-8"))
        print(f"{rel}")
        print(f"  name: {data['name']}")
        print(f"  category: {data['category']}")
        print(f"  summary: {data['summary']}")
        placeholders = ", ".join(sorted(data.get("placeholders", {}).keys()))
        print(f"  placeholders: {placeholders or '(none)'}")
    return 0


def parse_vars(raw_items: list[str]) -> dict[str, str]:
    values: dict[str, str] = {}
    for item in raw_items:
        if "=" not in item:
            raise SystemExit(f"FAIL: invalid --var value, expected KEY=VALUE: {item}")
        key, value = item.split("=", 1)
        key = key.strip()
        if not key:
            raise SystemExit(f"FAIL: empty placeholder name in --var: {item}")
        values[key] = value
    return values


def resolve_template(template_arg: str) -> Path:
    template_dir = (ASSETS_DIR / template_arg).resolve()
    if not template_dir.is_dir():
        raise SystemExit(f"FAIL: template not found: {template_arg}")
    if ASSETS_DIR not in template_dir.parents:
        raise SystemExit(f"FAIL: template must stay under {ASSETS_DIR}")
    return template_dir


def output_name_for(template_root: Path, source_file: Path, replacements: dict[str, str]) -> Path:
    rel = source_file.relative_to(template_root / "starter-files")
    if rel.name.endswith(".tmpl"):
        rel = rel.with_name(rel.name[:-5])
    rendered = render_content(rel.as_posix(), replacements)
    return Path(rendered)


def render_content(content: str, replacements: dict[str, str]) -> str:
    rendered = content
    for key, value in replacements.items():
        rendered = rendered.replace(key, value)
    return rendered


def ensure_required_placeholders(manifest: dict[str, object], replacements: dict[str, str]) -> None:
    required = set(manifest.get("placeholders", {}).keys())
    missing = sorted(key for key in required if key not in replacements)
    if missing:
        joined = ", ".join(missing)
        raise SystemExit(f"FAIL: missing placeholder values: {joined}")


def apply_template(
    template_dir: Path,
    dest_dir: Path,
    replacements: dict[str, str],
    *,
    force: bool,
    dry_run: bool,
) -> int:
    manifest_path = template_dir / "manifest.json"
    if not manifest_path.is_file():
        raise SystemExit(f"FAIL: manifest missing: {manifest_path}")
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    ensure_required_placeholders(manifest, replacements)

    starter_dir = template_dir / manifest["starter_files_dir"]
    if not starter_dir.is_dir():
        raise SystemExit(f"FAIL: starter files directory missing: {starter_dir}")

    source_files = sorted(path for path in starter_dir.rglob("*") if path.is_file())
    planned: list[tuple[Path, Path]] = []
    for source_file in source_files:
        out_rel = output_name_for(template_dir, source_file, replacements)
        out_path = dest_dir / out_rel
        planned.append((source_file, out_path))
        if out_path.exists() and not force:
            raise SystemExit(
                f"FAIL: destination file already exists, rerun with --force if intended: {out_path}"
            )

    if dry_run:
        print(f"[template] dry run: {template_dir.relative_to(ASSETS_DIR).as_posix()}")
        for source_file, out_path in planned:
            print(f"  {source_file.relative_to(starter_dir)} -> {out_path}")
        return 0

    dest_dir.mkdir(parents=True, exist_ok=True)
    for source_file, out_path in planned:
        out_path.parent.mkdir(parents=True, exist_ok=True)
        content = source_file.read_text(encoding="utf-8")
        out_path.write_text(render_content(content, replacements), encoding="utf-8")

    tree_file = template_dir / manifest["tree_file"]
    if tree_file.is_file():
        rendered_tree = render_content(tree_file.read_text(encoding="utf-8"), replacements)
        (dest_dir / "_template-tree.txt").write_text(rendered_tree, encoding="utf-8")

    print(f"PASS: applied template {manifest['name']} to {dest_dir}")
    return 0


def main() -> int:
    args = parse_args()

    if args.list:
        return list_templates()

    if not args.template:
        raise SystemExit("FAIL: --template is required unless --list is used")
    if not args.dest:
        raise SystemExit("FAIL: --dest is required unless --list is used")

    template_dir = resolve_template(args.template)
    replacements = parse_vars(args.var)
    dest_dir = Path(args.dest).expanduser().resolve()

    return apply_template(
        template_dir,
        dest_dir,
        replacements,
        force=args.force,
        dry_run=args.dry_run,
    )


if __name__ == "__main__":
    raise SystemExit(main())
