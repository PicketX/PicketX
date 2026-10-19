// SPDX-License-Identifier: Apache-2.0
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { readFile } from 'node:fs/promises'
import { join } from 'node:path'

test('reviewed distribution retains upstream notices and actual linked dependency licenses', async () => {
  const bundle = process.env.PICKETX_TEST_BUNDLE
  assert(bundle, 'actual test bundle required for distribution notice check')
  const manifest = await readFile(join(bundle, 'SHA256SUMS'), 'utf8')
  const notices = await readFile(join(bundle, 'THIRD_PARTY_NOTICES.md'), 'utf8')
  assert(notices.includes('shadcn') && notices.includes('modernc.org/libc'))
  const licenseIndex = JSON.parse(await readFile(join(bundle, 'licenses/index.json'), 'utf8'))
  for (const dependency of ['github.com/casbin/casbin/v3', 'modernc.org/libc', 'github.com/go-sql-driver/mysql', 'react', 'react-remove-scroll-bar', '@rolldown/binding-linux-x64-gnu', 'i18next', 'Go toolchain', 'musl']) {
    const entries = licenseIndex.filter(x => x.dependency === dependency)
    assert(entries.length > 0, 'missing dependency notices: ' + dependency)
    for (const entry of entries) for (const path of entry.files) {
      assert(manifest.includes(path), 'license not covered by checksum: ' + path)
      assert((await readFile(join(bundle, path))).length > 0)
    }
  }
})
