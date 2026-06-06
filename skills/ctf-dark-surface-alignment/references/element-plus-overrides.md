# Element Plus Overrides

Read this file when light backgrounds leak through Element Plus wrappers.

## Check these first

- `ElTable`
- `ElDialog`
- `ElInput`
- `ElTextarea`
- `ElButton`
- table scrollbar, inner wrapper, header wrapper, and body wrapper nodes

## Preferred approach

- If a page already uses the shared `teacher-surface` system, fix the missing wrapper or shared override instead of writing another isolated dark block.
- Add a stable class to the rendered component when you need a precise override target.

## Typical table fix

```css
.teacher-surface-table {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: var(--journal-surface);
  --el-table-border-color: var(--journal-border);
  --el-fill-color-blank: var(--journal-surface);
}

.teacher-surface-table.el-table,
.teacher-surface-table .el-table__inner-wrapper,
.teacher-surface-table .el-table__body-wrapper,
.teacher-surface-table .el-table__header-wrapper {
  background: var(--journal-surface);
}
```

## Typical dialog and input fix

```css
:deep(.el-dialog) {
  border: 1px solid var(--journal-border);
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--journal-surface) 96%, var(--color-bg-base)),
    color-mix(in srgb, var(--journal-surface-subtle) 94%, var(--color-bg-base))
  );
}

:deep(.el-input__wrapper),
:deep(.el-textarea__inner) {
  border: 1px solid var(--journal-border);
  background: var(--journal-surface);
  color: var(--journal-ink);
}
```
