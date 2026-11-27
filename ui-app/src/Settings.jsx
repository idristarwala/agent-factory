import { useEffect, useState } from 'react'

export default function Settings({ fetchAPI }) {
  const [state, setState] = useState(null)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const url = '/api/settings'
  async function read() {
    const r = await fetchAPI(url)
    if (!r.ok) throw new Error('Could not load settings')
    return await r.json()
  }
  useEffect(() => {
    let live = true
    read().then(data => { if (live) setState(data) }).catch(e => { if (live) setMessage(e.message) })
    return () => { live = false }
  }, [url,fetchAPI])
  function change(key, value) { setState(s => ({...s, settings: {...s.settings, [key]: value}})) }
  async function save(event) {
    event.preventDefault()
    setBusy(true)
    try {
      const r = await fetchAPI(url, {method: 'POST', headers: {'Content-Type':'application/json'},
        body:JSON.stringify({Expected: state.version, Settings:state.settings})})
      if (r.status === 409) { setState(await read()); setMessage('Settings changed in another session. Review the latest revision before saving.'); return }
      if (!r.ok) throw new Error((await r.text()).trim())
      setState(await read())
      setMessage('Run defaults saved.')
    } catch (e) { setMessage(e.message) } finally { setBusy(false) }
  }
  return <section className="definitions"><h2>Workspace settings</h2>
    {state && <form onSubmit={save}><p>Revision {state.version} · Gateway {state.gateway || 'configured by environment'}</p>
      {[['max_rounds','Maximum rounds'],['max_tokens','Token budget'],['max_minutes','Time limit in minutes']].map(([k,label]) =>
        <label key={k}>{label}<input type="number" min="1" required value={state.settings[k]} onChange={e=>change(k,Number(e.target.value))}/></label>)}
      {[['replay_gate','Hold learned rules for review'],['observe','Observe proposed changes']].map(([k,label]) =>
        <label key={k}><input type="checkbox" checked={state.settings[k]} onChange={e=>change(k,e.target.checked)}/> {label}</label>)}
      <button className="ok" disabled={busy}>Save run defaults</button></form>}
    <p role="status">{message}</p></section>
}
