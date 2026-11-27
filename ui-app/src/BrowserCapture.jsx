import {useState} from 'react'

export default function BrowserCapture({id,request}) {
  const [capture,setCapture]=useState(null)
  const [error,setError]=useState('')
  async function inspect(event) {
    if (!event.currentTarget.open || capture) return
    try {setCapture(await request('/api/browser-captures/'+encodeURIComponent(id)));setError('')}
    catch (err) {setError(err.message)}
  }
  return <details onToggle={inspect}><summary>Rendered page capture</summary>
    {error && <p role="alert">{error}</p>}
    {capture && <><p>{capture.url} · {capture.captured_at}</p>
      <p>{capture.tool} · revision {capture.tool_revision}</p>
      <pre>{capture.text}</pre>
      {capture.image && <img alt="Page at capture time" style={{maxWidth:'100%'}} src={'data:image/jpeg;base64,'+capture.image}/>}</>}
  </details>
}
