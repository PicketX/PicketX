// SPDX-License-Identifier: Apache-2.0
// Integration utilities only: normal CA/SAN verification, never a TLS bypass.
import https from 'node:https'
import { spawnSync } from 'node:child_process'
import { chmod, writeFile } from 'node:fs/promises'
import { join } from 'node:path'

export async function createTestCertificate(directory) {
  const ca = join(directory, 'ca.pem'), cert = join(directory, 'leaf.pem'), key = join(directory, 'leaf.key')
  const run = args => { if (spawnSync('openssl', args, { stdio: 'ignore', timeout: 15000 }).status !== 0) throw new Error('test certificate generation failed') }
  run(['req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1', '-keyout', join(directory, 'ca.key'), '-out', ca, '-subj', '/CN=PicketX disposable integration CA', '-addext', 'basicConstraints=critical,CA:TRUE'])
  run(['req', '-newkey', 'rsa:2048', '-nodes', '-keyout', key, '-out', join(directory, 'leaf.csr'), '-subj', '/CN=PicketX integration HTTPS'])
  const extensions = join(directory, 'leaf.ext')
  await writeFile(extensions, 'basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature,keyEncipherment\nextendedKeyUsage=serverAuth\nsubjectAltName=IP:127.0.0.1,IP:127.0.0.2\n', { mode: 0o600 })
  run(['x509', '-req', '-in', join(directory, 'leaf.csr'), '-CA', ca, '-CAkey', join(directory, 'ca.key'), '-CAcreateserial', '-days', '1', '-out', cert, '-extfile', extensions])
  await chmod(key, 0o600); await chmod(join(directory, 'ca.key'), 0o600)
  return { ca, cert, key }
}

export async function requestHTTPS(url, { ca, method = 'GET', headers = {}, body, timeout = 5000, servername } = {}) {
  const failure = () => new Error('HTTPS observation failed')
  let parsed; try { parsed = new URL(url) } catch { throw failure() }
  if (parsed.protocol !== 'https:') throw failure()
  return new Promise((resolve, reject) => {
    const request = https.request(parsed, { ca, method, headers, servername }, response => {
      const chunks = []; let bytes = 0
      response.on('data', chunk => { bytes += chunk.length; if (bytes > 1024 * 1024) request.destroy(); else chunks.push(chunk) })
      response.on('error', () => reject(failure()))
      response.on('end', () => {
        if (bytes > 1024 * 1024) return reject(failure())
        const text = Buffer.concat(chunks).toString('utf8')
        resolve({ status: response.statusCode, headers: response.headers, body: text, json() { try { return JSON.parse(text) } catch { throw new Error('HTTPS response JSON invalid') } } })
      })
    })
    const timer = setTimeout(() => request.destroy(), timeout)
    request.on('close', () => clearTimeout(timer)); request.on('error', () => reject(failure()))
    if (body !== undefined) request.write(body)
    request.end()
  })
}
