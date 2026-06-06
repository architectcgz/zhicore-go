#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf 'Usage: %s PROJECT_ROOT [--compile]\n' "$(basename "$0")" >&2
}

project_root=""
compile="false"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --compile)
      compile="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      usage
      exit 2
      ;;
    *)
      if [ -n "$project_root" ]; then
        usage
        exit 2
      fi
      project_root="$1"
      shift
      ;;
  esac
done

[ -n "$project_root" ] || { usage; exit 2; }
[ -d "$project_root" ] || { printf 'Error: not a directory: %s\n' "$project_root" >&2; exit 1; }

required_files=(
  "Thesis.tex"
  "hfut.cls"
  "ref.bib"
  "tex/info.tex"
  "tex/abstract.tex"
)

missing=0
for path in "${required_files[@]}"; do
  if [ ! -f "$project_root/$path" ]; then
    printf 'Missing: %s\n' "$path" >&2
    missing=1
  fi
done

if [ "$missing" -ne 0 ]; then
  exit 1
fi

printf 'Structure check passed: %s\n' "$project_root"

if grep -q 'font=adobe' "$project_root/Thesis.tex"; then
  for font in AdobeFangsongStd-Regular.otf AdobeHeitiStd-Regular.otf AdobeKaitiStd-Regular.otf AdobeSongStd-Light.otf; do
    if [ ! -f "$project_root/fonts/$font" ]; then
      printf 'Warning: Thesis.tex selects font=adobe but fonts/%s is missing.\n' "$font" >&2
    fi
  done
fi

if [ "$compile" = "true" ]; then
  command -v latexmk >/dev/null 2>&1 || {
    printf 'Error: latexmk is required for --compile.\n' >&2
    exit 1
  }

  (
    cd "$project_root"
    latexmk -synctex=1 -pdfxe -shell-escape -interaction=nonstopmode -file-line-error -outdir=tmp Thesis.tex
  )
fi
