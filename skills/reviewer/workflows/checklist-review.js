export const meta = {
  name: 'code-review-checklist',
  description: 'Enforce dimension-by-dimension code review with mandatory verdicts to prevent quality blind spots',
  phases: [
    { title: 'Load', detail: 'read diff and checklist' },
    { title: 'Dimension Check', detail: 'check each quality dimension' },
    { title: 'Consolidate', detail: 'merge findings and produce report' },
  ],
}

// Read checklist and target diff/files from args
const checklistPath = args?.checklistPath || `${process.env.HOME}/.agents/skills/reviewer/review-checklist.yaml`
const diffSource = args?.diffSource || 'git diff --cached'  // default: staged changes
const targetFiles = args?.files || []  // optional: specific files to review

phase('Load')

// Load checklist
const checklistYaml = await agent(
  `Read ${checklistPath} and parse it as YAML. Return the structured checklist object with dimensions array.`,
  {
    label: 'load-checklist',
    phase: 'Load',
    schema: {
      type: 'object',
      properties: {
        dimensions: {
          type: 'array',
          items: {
            type: 'object',
            properties: {
              dimension: { type: 'string' },
              source: { type: 'string' },
              verdict_required: { type: 'boolean' },
              items: { type: 'array', items: { type: 'string' } }
            },
            required: ['dimension', 'items', 'verdict_required']
          }
        }
      },
      required: ['dimensions']
    }
  }
)

if (!checklistYaml?.dimensions?.length) {
  log('ERROR: checklist load failed or empty')
  return { error: 'checklist load failed' }
}

const dimensions = checklistYaml.dimensions.filter(d => d.verdict_required)
log(`Loaded ${dimensions.length} mandatory dimensions`)

// Load diff
const diffContent = await agent(
  targetFiles.length
    ? `Read these files and show their current content: ${targetFiles.join(', ')}`
    : `Run: ${diffSource}. Return the full diff output.`,
  {
    label: 'load-diff',
    phase: 'Load',
    schema: {
      type: 'object',
      properties: {
        diff: { type: 'string' },
        files_changed: { type: 'array', items: { type: 'string' } },
        lines_added: { type: 'number' },
        lines_removed: { type: 'number' }
      },
      required: ['diff']
    }
  }
)

if (!diffContent?.diff) {
  log('No diff to review')
  return { verdict: 'no_diff', dimensions_checked: 0 }
}

log(`Diff loaded: ${diffContent.files_changed?.length || 0} files, +${diffContent.lines_added || 0} -${diffContent.lines_removed || 0}`)

phase('Dimension Check')

// Check each dimension
const dimensionResults = await pipeline(
  dimensions,
  (dim, _, idx) => agent(
    `You are reviewing code changes for quality issues.

**Dimension**: ${dim.dimension}
**Source reference**: ${dim.source}

**Check items**:
${dim.items.map((item, i) => `${i + 1}. ${item}`).join('\n')}

**Diff to review**:
\`\`\`
${diffContent.diff}
\`\`\`

**Task**:
For EACH item above, check whether this diff has risks in that area.
Return a structured verdict:
- verdict: "pass" (no issues found), "findings" (issues found), or "N/A" (dimension not applicable to this diff)
- findings: array of issues found (empty if pass or N/A)
- checked_items: count of items checked
- skipped_reason: if N/A, why this dimension doesn't apply

Each finding must include:
- item: which check item triggered this
- severity: Blocker / Major / Minor / Nit
- location: file:line or function/component name
- issue: what is wrong
- impact: why it matters
- suggestion: how to fix

Be thorough: check ALL items in the list. Do not skip items because "it looks fine at a glance".`,
    {
      label: `check-${dim.dimension}`,
      phase: 'Dimension Check',
      schema: {
        type: 'object',
        properties: {
          verdict: { type: 'string', enum: ['pass', 'findings', 'N/A'] },
          findings: {
            type: 'array',
            items: {
              type: 'object',
              properties: {
                item: { type: 'string' },
                severity: { type: 'string', enum: ['Blocker', 'Major', 'Minor', 'Nit'] },
                location: { type: 'string' },
                issue: { type: 'string' },
                impact: { type: 'string' },
                suggestion: { type: 'string' }
              },
              required: ['item', 'severity', 'location', 'issue', 'impact']
            }
          },
          checked_items: { type: 'number' },
          skipped_reason: { type: 'string' }
        },
        required: ['verdict', 'findings', 'checked_items']
      }
    }
  )
)

phase('Consolidate')

// Consolidate findings
const allFindings = dimensionResults.filter(Boolean).flatMap((r, idx) =>
  (r.findings || []).map(f => ({ ...f, dimension: dimensions[idx].dimension }))
)

const verdictSummary = dimensionResults.filter(Boolean).reduce((acc, r, idx) => {
  acc[dimensions[idx].dimension] = r.verdict
  return acc
}, {})

const blockers = allFindings.filter(f => f.severity === 'Blocker')
const majors = allFindings.filter(f => f.severity === 'Major')
const minors = allFindings.filter(f => f.severity === 'Minor')

log(`Review complete: ${blockers.length} blockers, ${majors.length} major, ${minors.length} minor`)

// Final gate verdict
let gateVerdict = 'pass'
if (blockers.length > 0) {
  gateVerdict = 'blocked'
} else if (majors.length > 0) {
  gateVerdict = 'pass_with_major_issues'
} else if (minors.length > 0) {
  gateVerdict = 'pass_with_minor_issues'
}

return {
  gate_verdict: gateVerdict,
  dimensions_checked: dimensions.length,
  verdict_summary: verdictSummary,
  findings: {
    total: allFindings.length,
    blocker: blockers.length,
    major: majors.length,
    minor: minors.length,
    by_dimension: allFindings.reduce((acc, f) => {
      acc[f.dimension] = (acc[f.dimension] || 0) + 1
      return acc
    }, {})
  },
  all_findings: allFindings.sort((a, b) => {
    const severityOrder = { Blocker: 0, Major: 1, Minor: 2, Nit: 3 }
    return severityOrder[a.severity] - severityOrder[b.severity]
  })
}
