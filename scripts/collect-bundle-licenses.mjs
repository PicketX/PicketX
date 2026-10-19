// SPDX-License-Identifier: Apache-2.0
// Build-only notice collection from the actual installed/linked dependencies.
// Copies upstream texts verbatim; it is not an automated legal clearance engine.
import assert from 'node:assert/strict'
import { cp, mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import { execFileSync } from 'node:child_process'
import { join, resolve } from 'node:path'

const output = resolve(process.argv[2]), index = []
await mkdir(join(output, 'licenses')) // Exclusive, never overwrite an older collection.
const notice = /^(license|licence|copying|copyright|notice|patents)(?:[._-].*)?$/i
async function collect(root, directory, includeAll = false) {
  const result = []
  for (const entry of await readdir(join(root, directory), { withFileTypes: true })) {
    if (entry.isSymbolicLink() || entry.name === '.git' || entry.name === 'node_modules') continue
    const name = join(directory, entry.name)
    if (entry.isDirectory()) result.push(...await collect(root, name, includeAll || /^licen[sc]es?$/i.test(entry.name)))
    else if (entry.isFile() && (includeAll || notice.test(entry.name))) result.push(name)
  }
  return result.sort()
}
async function retain(dependency, version, root, prefix) {
  assert(root, 'missing installed module directory: ' + dependency)
  const files = await collect(root, '')
  assert(files.length, 'no upstream notice files: ' + dependency)
  const paths = []
  for (const file of files) {
    const target = join('licenses', prefix, file)
    await mkdir(join(output, target, '..'), { recursive: true })
    await cp(join(root, file), join(output, target), { errorOnExist: true, force: false })
    paths.push(target)
  }
  index.push({ dependency, version, files: paths })
}
const run = (program, args) => execFileSync(program, args, { encoding: 'utf8', maxBuffer: 8 << 20, stdio: ['ignore', 'pipe', 'pipe'] }).trim()
const modules = new Set(run('go', ['list', '-deps', '-f', '{{if .Module}}{{if not .Module.Main}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}{{end}}', './cmd/picketx', './cmd/picketx-controller']).split('\n').filter(Boolean))
for (const entry of [...modules].sort()) {
  const [dependency, version, directory] = entry.split('|')
  await retain(dependency, version, directory, join('go', dependency + '@' + version))
}
const npmRoots = run('npm', ['ls', '--omit=dev', '--all', '--parseable', '--prefix', 'apps/web']).split('\n').slice(1)
for (const directory of [...new Set(npmRoots)].sort()) {
  const meta = JSON.parse(await readFile(join(directory, 'package.json'), 'utf8'))
  await retain(meta.name, meta.version, directory, join('npm', meta.name + '@' + meta.version))
}
const goroot = run('go', ['env', 'GOROOT'])
await mkdir(join(output, 'licenses/go-toolchain'))
for (const name of ['LICENSE', 'PATENTS']) await cp(join(goroot, name), join(output, 'licenses/go-toolchain', name))
index.push({ dependency: 'Go toolchain', version: run('go', ['version']), files: ['licenses/go-toolchain/LICENSE', 'licenses/go-toolchain/PATENTS'] })
// The supported builder is Ubuntu/Debian musl-tools, not implicit static glibc.
await mkdir(join(output, 'licenses/c-toolchain'))
for (const name of (await readdir('/usr/share/doc')).filter(n => /^(musl|musl-dev|musl-tools|libgcc-s1|gcc-[0-9]+-base)$/.test(n)).sort()) {
  const target = `licenses/c-toolchain/${name}-copyright`
  await cp(`/usr/share/doc/${name}/copyright`, join(output, target), { dereference: true })
  index.push({ dependency: name, version: run('dpkg-query', ['-W', '-f=${Version}', name]), files: [target] })
}
assert(index.some(x => x.dependency === 'musl'), 'musl copyright required')
await cp('THIRD_PARTY_NOTICES.md', join(output, 'THIRD_PARTY_NOTICES.md'))
await writeFile(join(output, 'licenses/index.json'), JSON.stringify(index, null, 2) + '\n')
console.log(`Retained upstream notices for ${index.length} dependency/toolchain entries`)
