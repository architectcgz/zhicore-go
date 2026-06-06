# HFUT Thesis Editing Guide

## Default Editing Strategy

Work in the highest-level file that solves the problem:

```text
metadata problem        -> tex/info.tex
abstract/keywords       -> tex/abstract.tex
body content            -> tex/*.tex
chapter order           -> Thesis.tex
references              -> ref.bib
images                  -> img/
template-level format   -> hfut.cls or tex/pages/*.tex
```

Only edit `hfut.cls` or `tex/pages/*.tex` when the user asks for a format change that cannot be handled in normal content files.

## Content Migration

When converting existing content to this template:

1. Put each major body section in a separate `tex/*.tex` file.
2. Use `\section`, `\subsection`, and `\subsubsection`; avoid `\chapter`.
3. Convert numbered lists to `enumerate` and unordered lists to `itemize`.
4. Put figures under `img/` and use stable relative paths such as `./img/result.png`.
5. Move references into `ref.bib` and cite them by key.
6. Keep Chinese punctuation and English spacing consistent with surrounding text.

## LaTeX Hygiene

Escape special characters in prose when needed:

```tex
\% \_ \& \# \$ \{ \}
```

For paths, code identifiers, and commands, prefer:

```tex
\texttt{some_identifier}
```

For cross-references, prefer `cleveref` labels:

```tex
\label{sec:method}
\cref{sec:method}
\label{fig:architecture}
\cref{fig:architecture}
```

Use clear label prefixes:

```text
sec:
subsec:
fig:
tab:
eq:
alg:
```

## Figures

Typical figure:

```tex
\begin{figure}[htb!]
    \centering
    \includegraphics[width=.8\textwidth]{./img/example.png}
    \caption{图片标题}
    \label{fig:example}
\end{figure}
```

Use vector PDF for diagrams when available. Use PNG/JPEG for screenshots and photographs.

## Tables

Use `booktabs` style if the package is available in the project; otherwise follow existing table style. Keep wide tables inside `table` floats and check generated PDF for overflow.

## Compile Debugging Order

1. Run the same `latexmk -pdfxe` command used by the template.
2. Read the first source-location error in `tmp/Thesis.log` or terminal output.
3. Fix one root cause at a time.
4. Re-run compilation after each fix.
5. If bibliography is empty or stale, clean generated files and rebuild with `biber` available.

## Format Cleanup Checklist

- `Thesis.tex` uses `\documentclass[mode=final,...]{hfut}` for final output.
- Adobe font mode is used only when required font files exist.
- `tex/info.tex` has no duplicate active date command definitions.
- Abstract keyword commands are present before their environments close.
- Every `\include{tex/...}` target exists.
- Every cited key exists in `ref.bib`.
- Every image path exists with matching case.
- No sample paragraphs from the template remain unless intentionally kept.
