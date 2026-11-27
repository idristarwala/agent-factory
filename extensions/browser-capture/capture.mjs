import { chromium } from 'playwright-core'
import { pathToFileURL } from 'node:url'

function checkedURL(raw, origins) {
  const url = new URL(raw)
  if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || !origins.has(url.origin)) {
    throw new Error('Browser capture origin denied')
  }
  url.hash = ''
  return url
}

export async function capture(browser, spec) {
  const allowed = new Set(spec.origins)
  const source = checkedURL(spec.url, allowed)
  const signal = AbortSignal.timeout(spec.timeout_ms)
  let requests = 0
  let bytes = 0
  async function load(url) {
    for (let redirects = 0; redirects <= 5; redirects++) {
      checkedURL(url, allowed)
      if (++requests > 64) throw new Error('Browser request limit reached')
      const response = await fetch(url, { redirect: 'manual', signal, credentials: 'omit' })
      if ([301, 302, 303, 307, 308].includes(response.status)) {
        await response.body?.cancel()
        if (!response.headers.has('location')) throw new Error('Missing redirect location')
        url = checkedURL(new URL(response.headers.get('location'), url),allowed)
        continue
      }
      const chunks = []
      for await (const chunk of response.body ?? []) {
        bytes += chunk.length
        if (bytes > 4 * 1024 * 1024) throw new Error('Browser source exceeds byte limit')
        chunks.push(chunk)
      }
      return { url, status: response.status, contentType: response.headers.get('content-type') || 'application/octet-stream', body: Buffer.concat(chunks) }
    }
    throw new Error('Browser redirect limit reached')
  }

  const first = await load(source)
  signal.throwIfAborted()
  if (first.status !== 200 || !/^text\/html\b/i.test(first.contentType)) throw new Error('Browser capture needs an HTML document')
  const context = await browser.newContext({ serviceWorkers: 'block', acceptDownloads: false, viewport: {width: 1024, height: 768} })
  const close = () => { context.close().catch(() => {}) }
  signal.addEventListener('abort', close, { once: true })
  try {
    signal.throwIfAborted()
    const page = await context.newPage()
    let initial = true
    let denied
    await context.routeWebSocket('**/*', socket => {denied ??= new Error('WebSockets are unavailable in read-only capture');socket.close()})
    await context.route('**/*', async route => {
      try {
        const req = route.request()
        if (req.method() !== 'GET' || (req.isNavigationRequest() && (!initial || req.frame() !== page.mainFrame()))) {
          throw new Error('Browser capture permits only read-only resources of the selected page')
        }
        const url = checkedURL(req.url(), allowed)
        const data = initial && url.href === first.url.href ? first : await load(url)
        initial = false
        await route.fulfill({
          status: data.status, contentType: data.contentType, body: data.body,
          headers: {'content-security-policy': "default-src http: https:; script-src http: https: 'unsafe-inline'; style-src http: https: 'unsafe-inline'; img-src http: https: data:; frame-src 'none'; object-src 'none'; worker-src 'none'; form-action 'none'; base-uri 'none'"}
        })
      } catch (error) {
        denied ??= error
        await route.abort().catch(() => {})
      }
    })
    await page.goto(first.url.href, { waitUntil: 'networkidle', timeout: spec.timeout_ms })
    const text = await page.locator('body').innerText({timeout: spec.timeout_ms})
    if (Buffer.byteLength(text) > spec.max_text_bytes) throw new Error('Rendered text exceeds capture limit')
    const image = spec.screenshot ? await page.screenshot({type:'jpeg', quality:70, animations:'disabled', timeout:spec.timeout_ms}) : undefined
    if (image?.length > 512 * 1024) throw new Error('Screenshot exceeds capture limit')
    if (denied) throw denied
    if (page.url() !== first.url.href) throw new Error('Page navigated during capture')
    return { url:first.url.href, text, image:image?.toString('base64') }
  } finally {
    signal.removeEventListener('abort', close)
    await context.close()
  }
}

export async function runCapture(launchOptions = {}) {
  let input = ''
  for await (const chunk of process.stdin) {
    input += chunk
    if (input.length > 32768) throw new Error('Capture request too large')
  }
  const spec = JSON.parse(input)
  const browser = await chromium.launch({
    executablePath: process.env.FACTORY_CHROMIUM_EXECUTABLE || undefined,
    chromiumSandbox: true,
    proxy: {server:'http://127.0.0.1:9', bypass:'<-loopback>'},
    args:['--force-webrtc-ip-handling-policy=disable_non_proxied_udp'],
    ...launchOptions
  })
  try { process.stdout.write(JSON.stringify(await capture(browser, spec))) }
  finally { await browser.close() }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  runCapture().catch(() => { console.error('Browser capture failed'); process.exitCode = 1 })
}
