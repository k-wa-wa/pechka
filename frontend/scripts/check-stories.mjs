#!/usr/bin/env node
// Fails if a component under components/ has no matching *.stories.tsx file.
// This is the mechanical half of "don't forget to wire up new UI when adding
// a feature" — see issue #36. It only checks presence, not story quality.
import { readdirSync } from 'node:fs'
import { join } from 'node:path'

const COMPONENTS_DIR = join(import.meta.dirname, '..', 'components')

const files = readdirSync(COMPONENTS_DIR)
const components = files
  .filter((f) => f.endsWith('.tsx') && !f.endsWith('.stories.tsx'))
  .map((f) => f.replace(/\.tsx$/, ''))
const stories = new Set(
  files.filter((f) => f.endsWith('.stories.tsx')).map((f) => f.replace(/\.stories\.tsx$/, ''))
)

const missing = components.filter((c) => !stories.has(c))

if (missing.length > 0) {
  console.error('Missing Storybook stories for:')
  for (const name of missing) {
    console.error(`  - components/${name}.tsx (expected components/${name}.stories.tsx)`)
  }
  console.error('\nAdd a story so the component can be developed/reviewed without the real API.')
  process.exit(1)
}

console.log(`OK: all ${components.length} components have a matching story.`)
