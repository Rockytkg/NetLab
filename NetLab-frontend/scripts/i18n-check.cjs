#!/usr/bin/env node

/**
 * i18n Hygiene Audit Script
 *
 * Checks:
 *   1. No hardcoded CJK characters in source files (outside i18n/locales)
 *   2. All locale JSON files have identical keys across languages
 *   3. Every namespace referenced in code has corresponding locale files
 *
 * Usage: node scripts/i18n-check.cjs
 */

const fs = require('fs')
const path = require('path')
const { execSync } = require('child_process')

const ROOT = path.resolve(__dirname, '..')
const SRC = path.join(ROOT, 'src')
const LOCALES = path.join(SRC, 'i18n', 'locales')
const LANGS = ['zh-CN', 'en-US']

let errors = 0
let warnings = 0

function error(msg) {
  console.error(`  ✗ ${msg}`)
  errors++
}

function warn(msg) {
  console.warn(`  ⚠ ${msg}`)
  warnings++
}

function ok(msg) {
  console.log(`  ✓ ${msg}`)
}

// ──────────────────────────────────────────────
// Check 1: Hardcoded CJK in source (not locales)
// ──────────────────────────────────────────────
console.log('\n[1] Scanning for hardcoded Chinese characters...')

const sourceFiles = []
function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist' || entry.name === '.git') continue
      walk(full)
    } else if (/\.(tsx?|jsx?|css)$/.test(entry.name)) {
      sourceFiles.push(full)
    }
  }
}
walk(SRC)

for (const file of sourceFiles) {
  if (file.includes('/locales/')) continue
  if (file.includes('/i18n/')) continue

  const content = fs.readFileSync(file, 'utf-8')
  const lines = content.split('\n')

  let inBlockComment = false

  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim()
    const line = lines[i]

    // Track block comment state
    if (trimmed.startsWith('/*') || trimmed.startsWith('/**') || trimmed.startsWith('{/*')) {
      inBlockComment = true
      if (trimmed.endsWith('*/')) inBlockComment = false // single-line block comment
      continue
    }
    if (inBlockComment) {
      if (trimmed.includes('*/') || trimmed === '*/') inBlockComment = false
      continue
    }
    if (trimmed.startsWith('//')) continue
    if (trimmed.startsWith('*')) continue // JSDoc body lines not caught above

    // Skip lines marked with i18n-allow pragma
    if (trimmed.includes('// i18n-allow')) continue

    // Strip inline // comments
    const codeOnly = trimmed.replace(/\/\/.*$/, '').trim()
    if (!codeOnly) continue

    // Skip import/export lines, console, and i18n infrastructure
    if (codeOnly.includes('import ') || codeOnly.includes('export ')) continue
    if (codeOnly.includes('console.')) continue
    if (codeOnly.includes('t(') || codeOnly.includes('useTranslation') || codeOnly.includes('i18next')) continue

    // Detect CJK (two or more consecutive characters)
    const match = codeOnly.match(/[一-鿿㐀-䶿]{2,}/)
    if (match) {
      const relPath = path.relative(ROOT, file)
      error(`${relPath}:${i + 1} — hardcoded Chinese: "${match[0]}"`)
    }
  }
}

// ──────────────────────────────────────────────
// Check 2: Locale key parity
// ──────────────────────────────────────────────
console.log('\n[2] Checking locale key parity...')

const namespaceFiles = {}
for (const lang of LANGS) {
  const langDir = path.join(LOCALES, lang)
  namespaceFiles[lang] = {}
  for (const file of fs.readdirSync(langDir)) {
    if (!file.endsWith('.json')) continue
    const ns = file.replace('.json', '')
    namespaceFiles[lang][ns] = JSON.parse(fs.readFileSync(path.join(langDir, file), 'utf-8'))
  }
}

const allNamespaces = new Set()
for (const lang of LANGS) {
  for (const ns of Object.keys(namespaceFiles[lang])) {
    allNamespaces.add(ns)
  }
}

for (const ns of allNamespaces) {
  const zhKeys = Object.keys(namespaceFiles['zh-CN']?.[ns] || {})
  const enKeys = Object.keys(namespaceFiles['en-US']?.[ns] || {})

  if (zhKeys.length === 0 && enKeys.length === 0) continue

  const zhOnly = zhKeys.filter((k) => !enKeys.includes(k))
  const enOnly = enKeys.filter((k) => !zhKeys.includes(k))

  if (zhOnly.length > 0) {
    for (const k of zhOnly) error(`"${ns}.json" — key "${k}" exists in zh-CN but missing in en-US`)
  }
  if (enOnly.length > 0) {
    for (const k of enOnly) error(`"${ns}.json" — key "${k}" exists in en-US but missing in zh-CN`)
  }

  if (zhOnly.length === 0 && enOnly.length === 0) {
    ok(`${ns}.json — ${zhKeys.length} keys, parity OK`)
  }
}

// Check for empty values
for (const lang of LANGS) {
  for (const ns of Object.keys(namespaceFiles[lang])) {
    const data = namespaceFiles[lang][ns]
    for (const [key, value] of Object.entries(data)) {
      if (value === '' || value === null) {
        error(`${lang}/${ns}.json — key "${key}" has empty value`)
      }
    }
  }
}

// ──────────────────────────────────────────────
// Check 3: I18nNamespace type covers all namespaces
// ──────────────────────────────────────────────
console.log('\n[3] Checking I18nNamespace type...')

const typesFile = path.join(SRC, 'types', 'i18n.ts')
if (fs.existsSync(typesFile)) {
  const content = fs.readFileSync(typesFile, 'utf-8')
  for (const ns of allNamespaces) {
    if (!content.includes(`'${ns}'`)) {
      error(`I18nNamespace type missing "${ns}"`)
    } else {
      ok(`"${ns}" present in I18nNamespace`)
    }
  }
}

// Check that i18n/index.ts imports all namespace files
const i18nIndex = path.join(SRC, 'i18n', 'index.ts')
if (fs.existsSync(i18nIndex)) {
  const content = fs.readFileSync(i18nIndex, 'utf-8')
  for (const ns of allNamespaces) {
    if (!content.includes(`${ns}.json`)) {
      warn(`i18n/index.ts may be missing import for "${ns}.json"`)
    }
  }
}

// ──────────────────────────────────────────────
// Summary
// ──────────────────────────────────────────────
console.log(`\n${'═'.repeat(50)}`)
console.log(`  Errors:   ${errors}`)
console.log(`  Warnings: ${warnings}`)
console.log(`${'═'.repeat(50)}\n`)

if (errors > 0) {
  console.log('i18n check FAILED. Fix the errors above before committing.')
  process.exit(1)
} else {
  console.log('i18n check PASSED.')
  process.exit(0)
}
