# HFUT Thesis Template Notes

These notes summarize the `shinyypig/hfut-thesis` template observed at commit `ab86b7acbc3a28d4e55d70417b3a8656c292b805`.

## Upstream Layout

Expected project files:

```text
Thesis.tex
hfut.cls
ref.bib
build.sh
build.bat
fonts/
img/
tex/
  abstract.tex
  acknowledge.tex
  appendix.tex
  info.tex
  introduction.tex
  pages/
    cover.tex
    chinese_title.tex
    english_title.tex
    statement.tex
    toc.tex
```

## Main Entry

`Thesis.tex` uses:

```tex
\documentclass[mode=final,font=adobe]{hfut}
\input{tex/info}
```

The document order is:

```text
cover
Chinese title page
English title page
statement page
Chinese and English abstract
table of contents
body chapters
references
acknowledgement
appendix
```

The body starts with:

```tex
\pagenumbering{arabic}
\pagestyle{contentstyle}
\include{tex/introduction}
```

Add new chapters by creating `tex/<name>.tex` and adding `\include{tex/<name>}`.

## Class Options

`hfut.cls` supports:

```tex
\documentclass[mode=final,font=fandol]{hfut}
\documentclass[mode=preprint,font=fandol]{hfut}
\documentclass[mode=final,font=adobe]{hfut}
```

`mode=preprint` enables line numbers and visible change markup. `mode=final` is for final printing.

`font=fandol` uses the TeX distribution's Fandol fonts. `font=adobe` expects these files under `fonts/`:

```text
AdobeFangsongStd-Regular.otf
AdobeHeitiStd-Regular.otf
AdobeKaitiStd-Regular.otf
AdobeSongStd-Light.otf
```

When Adobe fonts are unavailable, switch to `font=fandol` before attempting deeper fixes.

## Metadata

`tex/info.tex` defines:

```tex
\def\privacy{密级}
\def\type{设计或者论文}
\def\titleCna{中文题目第一行}
\def\titleCnb{中文题目第二行}
\def\titleEn{English Title}
\def\supervisor{姓名 \quad 职称}
\def\studentID{20xxxxxxxxx}
\def\studentNameCn{中文姓名}
\def\studentNameEn{English Name}
\def\department{学院全称}
\def\major{专业全称}
\def\enrolmentYear{2020级}
```

The template auto-generates finish and sign dates with `\the\year`, `\the\month`, and `\the\day`. For a fixed date, replace them with explicit values:

```tex
\newcommand{\finishedYear}{2026}
\newcommand{\finishedMonth}{5}
\newcommand{\finishedDay}{1}
\newcommand{\signYear}{2026}
\newcommand{\signMonth}{5}
\newcommand{\signDay}{1}
```

Make sure duplicate `\newcommand` definitions are not active at the same time.

## Abstracts And Keywords

`tex/abstract.tex` defines keyword commands and then uses custom environments:

```tex
\newcommand{\keywordsCn}[1][关键词一；关键词二；关键词三]{#1}
\begin{abstract}
中文摘要正文。
\end{abstract}

\newcommand{\keywordsEn}[1][Keyword One, Keyword Two, Keyword Three]{#1}
\begin{abstractEn}
English abstract body.
\end{abstractEn}
```

The template class adds the keyword labels at environment close, so do not manually repeat the label unless the user intentionally wants custom layout.

## Bibliography

`hfut.cls` configures:

```tex
\RequirePackage[
    backend=biber,
    citestyle=numeric-comp,
    bibstyle=gb7714-2015,
    sorting=none,
    gbalign=left,
    url=false,
    doi=false,
    eprint=false,
]{biblatex}
\addbibresource{ref.bib}
```

Use `\cite{key}` or the template's `\supercite{key}`. Keep entries in `ref.bib`.

## Build

The upstream `build.sh` command is:

```bash
latexmk -synctex=1 -pdfxe -shell-escape -interaction=nonstopmode -file-line-error -outdir=tmp Thesis.tex
```

Expected tools include `latexmk`, XeLaTeX, `biber`, and LaTeX packages for `ctex`, `biblatex`, `biblatex-gb7714-2015`, `algorithm`, `changes`, `cleveref`, `caption`, and related dependencies.

## Formatting Behavior In `hfut.cls`

Important built-in choices:

- A4 article class.
- Main body page style header: `合肥工业大学本科毕业设计（论文）`.
- Chinese and English abstracts use fixed 22pt line spacing.
- Section headings are centered, Chinese `\section` uses Heiti/Sanhao style.
- `\subsection` uses Heiti/Xiaosihao.
- Figures, tables, and equations are numbered within sections.
- Captions use Songti/Wuhao formatting.
- References use `gb7714-2015` numeric style through `biblatex`.

Do not duplicate these settings in chapter files.
