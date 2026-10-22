// SPDX-License-Identifier: Apache-2.0
// Signed protocol fixture ONLY. This is not a deployed IdP/authentication service.
import assert from 'node:assert/strict'
import { randomBytes, createHash, generateKeyPairSync, sign } from 'node:crypto'
import { createServer } from 'node:http'
import { once } from 'node:events'

export async function createOIDCProviderFixture({ origin, clientID, secret, subject }) {
  const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 })
  const jwk = { ...publicKey.export({ format: 'jwk' }), use: 'sig', alg: 'RS256', kid: 'fixture' }
  const codes = new Map()
  let issuer, exchanges = 0, lastCallback = '', mode = 'valid'
  const server = createServer(async (req, res) => {
    try {
      const u = new URL(req.url, issuer); res.setHeader('Cache-Control', 'no-store')
      const json = value => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(value)) }
      if (u.pathname === '/.well-known/openid-configuration') return json({ issuer, authorization_endpoint: `${issuer}/authorize`, token_endpoint: `${issuer}/token`, jwks_uri: `${issuer}/keys`, id_token_signing_alg_values_supported: ['RS256'] })
      if (u.pathname === '/keys') return json({ keys: [jwk] })
      if (u.pathname === '/authorize') {
        assert.equal(u.searchParams.get('client_id'), clientID); assert.equal(u.searchParams.get('response_type'), 'code'); assert.equal(u.searchParams.get('scope'), 'openid')
        assert.equal(u.searchParams.get('redirect_uri'), `${origin}/console/api/v1/oidc/callback`); assert.equal(u.searchParams.get('code_challenge_method'), 'S256')
        const code = randomBytes(32).toString('hex')
        codes.set(code, { nonce: u.searchParams.get('nonce'), challenge: u.searchParams.get('code_challenge') })
        const callback = new URL(`${origin}/console/api/v1/oidc/callback`)
        callback.searchParams.set('code', code); callback.searchParams.set('state', u.searchParams.get('state')); lastCallback = callback.href
        res.setHeader('Content-Type', 'text/html')
        return res.end(`<title>Signed OIDC test issuer</title><h1>Signed OIDC test issuer</h1><p>Protocol fixture only, not production authentication.</p><a href="${callback.href.replaceAll('&', '&amp;')}">Continue as test user</a>`)
      }
      if (u.pathname === '/token') {
        let body = ''; for await (const chunk of req) body += chunk
        const q = new URLSearchParams(body), proof = codes.get(q.get('code'))
        codes.delete(q.get('code')); exchanges++
        assert(proof); assert.equal(req.headers.authorization, `Basic ${Buffer.from(`${clientID}:${secret}`).toString('base64')}`)
        assert.equal(createHash('sha256').update(q.get('code_verifier')).digest('base64url'), proof.challenge)
        assert.equal(q.get('redirect_uri'), `${origin}/console/api/v1/oidc/callback`)
        const jwt = { iss: issuer, aud: clientID, sub: subject, iat: Math.floor(Date.now() / 1000), exp: Math.floor(Date.now() / 1000) + 600, nonce: mode === 'nonce' ? 'forged' : proof.nonce, email: 'administrator@example.test', roles: ['role:admin'] }
        const input = Buffer.from(JSON.stringify({ alg: 'RS256', kid: 'fixture' })).toString('base64url') + '.' + Buffer.from(JSON.stringify(jwt)).toString('base64url')
        return json({ access_token: 'fixture-not-persisted', token_type: 'Bearer', id_token: input + '.' + sign('RSA-SHA256', Buffer.from(input), privateKey).toString('base64url') })
      }
      res.writeHead(404).end()
    } catch { res.writeHead(400).end('Protocol fixture rejected request') }
  })
  // A different literal loopback address exercises actual Chromium cross-site return.
  server.listen(0, '127.0.0.2'); await once(server, 'listening')
  issuer = `http://127.0.0.2:${server.address().port}`
  return {
    issuer,
    get exchanges() { return exchanges },
    get lastCallback() { return lastCallback },
    setMode(value) { assert(['valid', 'nonce'].includes(value)); mode = value },
    close: () => new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())),
  }
}
