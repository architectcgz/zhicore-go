---
name: hfut-thesis-format
description: Use when creating, migrating, checking, or fixing a Hefei University of Technology (HFUT, 合肥工业大学) undergraduate thesis or graduation design in LaTeX, especially projects based on shinyypig/hfut-thesis, hfut.cls, Thesis.tex, tex/info.tex, biblatex gb7714-2015, XeLaTeX, or HFUT thesis formatting requirements.
---

# HFUT Thesis Format

## Purpose

Help users create, edit, validate, or troubleshoot a HFUT undergraduate thesis project based on the `shinyypig/hfut-thesis` LaTeX template.

Prefer minimal changes to the user's thesis content. Treat `hfut.cls` and page templates as formatting infrastructure; edit them only when the requested behavior cannot be achieved through `Thesis.tex`, `tex/info.tex`, chapter files, figures, or bibliography entries.

## Source Of Truth

Use the upstream template as the primary implementation reference:

- Repository: `https://github.com/shinyypig/hfut-thesis`
- Observed reference commit when this skill was created: `ab86b7acbc3a28d4e55d70417b3a8656c292b805`
- License: MIT

If the user asks for current upstream behavior, check the repository again before making claims.

## Workflow

1. Identify the thesis project root.
   - It usually contains `Thesis.tex`, `hfut.cls`, `ref.bib`, and `tex/info.tex`.
   - If the user wants a new project, use `scripts/create_hfut_thesis_project.sh`.

2. Read the local project before editing.
   - Start with `Thesis.tex`, `tex/info.tex`, `hfut.cls`, `tex/abstract.tex`, and the chapter files under `tex/`.
   - Compare local structure with `references/template-notes.md` when unsure.

3. Keep edits in the normal content layer.
   - Metadata: edit `tex/info.tex`.
   - Abstract and keywords: edit `tex/abstract.tex`.
   - Chapters: add or edit `tex/*.tex`, then include them from `Thesis.tex`.
   - References: edit `ref.bib`; keep citations compatible with `biblatex` and `gb7714-2015`.
   - Figures: put assets under `img/` unless the project already uses another convention.

4. Compile with XeLaTeX through `latexmk`.
   - Preferred command: `latexmk -synctex=1 -pdfxe -shell-escape -interaction=nonstopmode -file-line-error -outdir=tmp Thesis.tex`
   - The helper `scripts/check_hfut_thesis_project.sh` performs structure checks and optionally compiles.

5. Report verification honestly.
   - If `latexmk`, `xelatex`, `biber`, or required fonts/packages are missing, say which dependency blocked compilation.
   - Do not claim visual or format correctness unless a PDF was actually built and inspected or the relevant source was checked.

## Common Tasks

### Create a new thesis project

Run:

```bash
/home/azhi/.codex/skills/hfut-thesis-format/scripts/create_hfut_thesis_project.sh /path/to/new-thesis
```

Then update `tex/info.tex`, replace sample chapter text, and compile.

### Fill thesis metadata

Edit `tex/info.tex` fields such as `\titleCna`, `\titleCnb`, `\titleEn`, `\supervisor`, `\studentID`, `\studentNameCn`, `\studentNameEn`, `\department`, `\major`, and `\enrolmentYear`.

For fixed submission dates, replace the auto date commands with explicit `\newcommand` values as shown in `references/template-notes.md`.

### Add chapters

Create a chapter file under `tex/`, for example `tex/method.tex`, and add this line in `Thesis.tex` after the existing chapter includes:

```tex
\include{tex/method}
```

Use `\section`, `\subsection`, and `\subsubsection`; the template is article-based, so chapter-level commands such as `\chapter` are not expected.

### Troubleshoot compile failures

Read the first real LaTeX error from the build log, not only the final `latexmk` summary. Common causes:

- compiling with pdfLaTeX instead of XeLaTeX;
- missing `biber` or `biblatex-gb7714-2015`;
- selecting `font=adobe` without the Adobe font files under `fonts/`;
- broken image paths or unsupported image formats;
- malformed `.bib` entries;
- using unescaped special characters such as `%`, `_`, `&`, `#` in plain text.

## References

- Read `references/template-notes.md` for the upstream template layout, commands, build command, class options, and formatting behavior.
- Read `references/editing-guide.md` before larger thesis content migrations, Word-to-LaTeX conversion, or format cleanup.
