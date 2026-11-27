import {test} from 'node:test'
import assert from 'node:assert/strict'
import http from 'node:http'
import {chromium} from 'playwright-core'
import {capture} from './capture.mjs'

async function serve(handler) {
  const server=http.createServer(handler)
  await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve))
  return {origin:'http://127.0.0.1:'+server.address().port,close:()=>{server.closeAllConnections();server.close()}}
}

test('renders an isolated page, follows allowed redirects and rejects disallowed GETs and writes',async()=>{
  let blockedHits=0,writes=0
  const denied=await serve((_req,res)=>{blockedHits++;res.end('forbidden')})
  const source=await serve((req,res)=>{
    if(req.method!=='GET'){writes++;res.end();return}
    if(req.url==='/redirect'||req.url==='/denied'){res.writeHead(302,{location:req.url==='/redirect'?'/page':denied.origin});res.end();return}
    res.setHeader('content-type','text/html')
    if(req.url==='/asset'){res.end('<img src="'+denied.origin+'/image">');return}
    if(req.url==='/post'){res.end('<script>fetch("/save",{method:"POST"})</script>');return}
    res.end('<main></main><script>document.querySelector("main").textContent=localStorage.seen||"Rendered with JavaScript";localStorage.seen="leaked context"</script>')
  })
  let browser
  try {
    browser=await chromium.launch({executablePath:process.env.FACTORY_CHROMIUM_EXECUTABLE||undefined,proxy:{server:'http://127.0.0.1:9',bypass:'<-loopback>'}})
    const spec={origins:[source.origin],timeout_ms:8000,max_text_bytes:1024,screenshot:true}
    const run=path=>capture(browser,{...spec,url:source.origin+path})
    const result=await run('/redirect')
    assert.equal(result.url,source.origin+'/page');assert.equal(result.text,'Rendered with JavaScript')
    assert.ok(Buffer.from(result.image,'base64').length>100)
    assert.equal((await run('/page')).text,result.text)
    for(const path of ['/denied','/asset','/post']) await assert.rejects(run(path))
    assert.equal(blockedHits,0);assert.equal(writes,0)
  } finally {await browser?.close();source.close();denied.close()}
})

test('production executable captures a page with the Chromium sandbox', {skip:process.env.CI !== 'true'},async()=>{
  const source=await serve((_req,res)=>{res.setHeader('content-type','text/html');res.end('<p>Sandboxed capture</p>')})
  try {
    const {spawn}=await import('node:child_process')
    const child=spawn(process.execPath,['capture.mjs'],{stdio:['pipe','pipe','inherit'],signal:AbortSignal.timeout(20000)})
    const result=new Promise((resolve,reject)=>{
      let output=''
      child.stdout.on('data',chunk=>{output+=chunk})
      child.on('error',reject)
      child.on('close',code=>code===0?resolve(JSON.parse(output)):reject(new Error('capture exit '+code)))
    })
    child.stdin.end(JSON.stringify({url:source.origin,origins:[source.origin],timeout_ms:10000,max_text_bytes:1024,screenshot:false}))
    assert.equal((await result).text,'Sandboxed capture')
  } finally {source.close()}
})
