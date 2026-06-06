#!/usr/bin/env node
import { execSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import path from 'node:path'

const FRONTEND_FILE_REGEX = /\.(vue|ts|tsx|js|jsx|css|scss|sass|less|styl)$/i
const IMPORT_RE = /\bfrom\s+['"]([^'"]+)['"]|import\s*\(\s*['"]([^'"]+)['"]\s*\)|<style[^>]*\ssrc=['"]([^'"]+)['"][^>]*>|@import\s+['"]([^'"]+)['"]/g

function parseArgs(argv) {
  const args = {
    aliases: [],
    roots: [],
    minParents: 2,
    allow: [],
    cwd: process.cwd(),
    help: false,
  }

  for (let index = 0; index < argv.length; index += 1) {
    const current = argv[index]
    if (current === '--help' || current === '-h') {
      args.help = true
      continue
    }
    if (current === '--alias') {
      args.aliases.push(argv[index + 1])
      index += 1
      continue
    }
    if (current === '--root') {
      args.roots.push(argv[index + 1])
      index += 1
      continue
    }
    if (current === '--allow') {
      args.allow.push(argv[index + 1])
      index += 1
      continue
    }
    if (current === '--min-parents') {
      args.minParents = Number.parseInt(argv[index + 1], 10)
      index += 1
      continue
    }
    if (current === '--cwd') {
      args.cwd = path.resolve(argv[index + 1])
      index += 1
      continue
    }
    throw new Error(`Unknown argument: ${current}`)
  }

  return args
}

function printHelp() {
  console.log(`Usage:
  node check-alias-paths.mjs [--cwd <dir>] [--alias @ --root src] [--allow <pattern>] [--min-parents 2]

Examples:
  node check-alias-paths.mjs --cwd /repo --alias @ --root src
  node check-alias-paths.mjs --allow "src/features/local-only/**"

Behavior:
  - Scans frontend source files for deep relative imports such as ../../foo/bar
  - Flags matches that resolve inside configured alias roots
  - Keeps same-directory or shallow companion imports untouched by default
`)
}

function tryRun(command, cwd) {
  try {
    return execSync(command, {
      cwd,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'ignore'],
      maxBuffer: 1024 * 1024 * 8,
    }).trim()
  } catch {
    return ''
  }
}

function findRepoRoot(cwd) {
  return tryRun('git rev-parse --show-toplevel', cwd) || cwd
}

function readJsonFile(filePath) {
  try {
    return JSON.parse(readFileSync(filePath, 'utf8'))
  } catch {
    return null
  }
}

function detectAliasRoots(repoRoot) {
  const configCandidates = [
    path.join(repoRoot, 'tsconfig.json'),
    path.join(repoRoot, 'tsconfig.app.json'),
    path.join(repoRoot, 'jsconfig.json'),
  ]

  const detected = []
  for (const configPath of configCandidates) {
    if (!existsSync(configPath)) continue
    const config = readJsonFile(configPath)
    const paths = config?.compilerOptions?.paths
    if (!paths || typeof paths !== 'object') continue

    for (const [aliasPattern, targets] of Object.entries(paths)) {
      if (!Array.isArray(targets) || targets.length === 0) continue
      const alias = aliasPattern.replace(/\/\*$/, '')
      for (const target of targets) {
        if (typeof target !== 'string') continue
        const root = target.replace(/\/\*$/, '').replace(/^\.\//, '')
        if (!root) continue
        detected.push({ alias, root })
      }
    }
  }

  return detected
}

function buildAliasConfig(options, repoRoot) {
  const explicitPairs = []
  if (options.aliases.length > 0 || options.roots.length > 0) {
    if (options.aliases.length === 0 || options.roots.length === 0) {
      throw new Error('Both --alias and --root are required when providing explicit alias config.')
    }
    for (const alias of options.aliases) {
      for (const root of options.roots) {
        explicitPairs.push({ alias, root })
      }
    }
  }

  const detectedPairs = detectAliasRoots(repoRoot)
  const pairs = explicitPairs.length > 0 ? explicitPairs : detectedPairs
  const unique = new Map()
  for (const pair of pairs) {
    const normalizedRoot = path.resolve(repoRoot, pair.root)
    const key = `${pair.alias}::${normalizedRoot}`
    unique.set(key, { alias: pair.alias, root: normalizedRoot })
  }
  return [...unique.values()]
}

function listFrontendFiles(repoRoot) {
  const rgFiles = tryRun(
    "rg --files --glob '!node_modules' --glob '!dist' --glob '!build' --glob '!coverage' --glob '!storybook-static' --glob '!vendor' .",
    repoRoot
  )

  if (!rgFiles) return []
  return rgFiles
    .split('\n')
    .filter(Boolean)
    .filter((file) => FRONTEND_FILE_REGEX.test(file))
}

function countParentSegments(specifier) {
  const segments = specifier.split('/').filter(Boolean)
  let count = 0
  for (const segment of segments) {
    if (segment === '..') {
      count += 1
      continue
    }
    break
  }
  return count
}

function shouldSkipByAllowlist(relativeFilePath, allowPatterns) {
  return allowPatterns.some((pattern) => relativeFilePath.includes(pattern))
}

function resolveWithExtensions(basePath) {
  const extensions = ['', '.ts', '.tsx', '.js', '.jsx', '.vue', '.css', '.scss', '.sass', '.less', '.styl']
  for (const extension of extensions) {
    const candidate = `${basePath}${extension}`
    if (existsSync(candidate)) return candidate
  }

  const indexExtensions = ['index.ts', 'index.tsx', 'index.js', 'index.jsx', 'index.vue', 'index.css', 'index.scss']
  for (const extension of indexExtensions) {
    const candidate = path.join(basePath, extension)
    if (existsSync(candidate)) return candidate
  }

  return basePath
}

function inspectFile(repoRoot, filePath, aliasRoots, options) {
  const absoluteFilePath = path.join(repoRoot, filePath)
  const content = readFileSync(absoluteFilePath, 'utf8')
  const findings = []

  for (const match of content.matchAll(IMPORT_RE)) {
    const specifier = match[1] || match[2] || match[3] || match[4]
    if (!specifier || !specifier.startsWith('../')) continue

    const parentCount = countParentSegments(specifier)
    if (parentCount < options.minParents) continue
    if (shouldSkipByAllowlist(filePath, options.allow)) continue

    const targetPath = resolveWithExtensions(path.resolve(path.dirname(absoluteFilePath), specifier))
    const targetRelativeToRepo = path.relative(repoRoot, targetPath)
    if (targetRelativeToRepo.startsWith('..')) continue

    const matchedAlias = aliasRoots.find(({ root }) => {
      const relativeToRoot = path.relative(root, targetPath)
      return relativeToRoot && !relativeToRoot.startsWith('..') && !path.isAbsolute(relativeToRoot)
    })

    if (!matchedAlias) continue

    const line = content.slice(0, match.index).split('\n').length
    findings.push({
      filePath,
      line,
      specifier,
      alias: matchedAlias.alias,
      targetRelativeToRepo,
    })
  }

  return findings
}

function main() {
  const options = parseArgs(process.argv.slice(2))
  if (options.help) {
    printHelp()
    return
  }

  const repoRoot = findRepoRoot(options.cwd)
  const aliasRoots = buildAliasConfig(options, repoRoot)

  if (aliasRoots.length === 0) {
    console.log('[check-alias-paths] skipped: no alias roots detected and no --alias/--root provided')
    return
  }

  const files = listFrontendFiles(repoRoot)
  const findings = files.flatMap((filePath) => inspectFile(repoRoot, filePath, aliasRoots, options))

  if (findings.length === 0) {
    console.log(`[check-alias-paths] OK (${files.length} files scanned)`)
    return
  }

  console.error('[check-alias-paths] deep relative cross-directory imports detected:\n')
  for (const finding of findings) {
    console.error(
      `${finding.filePath}:${finding.line}  ${finding.specifier}  ->  ${finding.alias}/${finding.targetRelativeToRepo}`
    )
  }
  console.error('\nUse alias paths for shared cross-directory references, or allowlist intentional local exceptions with --allow.')
  process.exitCode = 1
}

main()
