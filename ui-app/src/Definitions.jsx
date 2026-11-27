import { useEffect, useState } from 'react'

const blankRule = { Description: '', Condition: '', Action: '' }
const blankAgent = { Name: '', Role: 'worker', ModelTier: 'cheap', Version: '', instructions: '', Retired: false }

export default function Definitions({ fetchAPI, kind }) {
  const isRule = kind === 'rules'
  const idOf = row => isRule ? row.ID : row.Name
  const versionOf = row => isRule ? row.Version : row.Revision
  const [rows, setRows] = useState([])
  const [selected, setSelected] = useState('')
  const [draft, setDraft] = useState(isRule ? blankRule : blankAgent)
  const [base, setBase] = useState(0)
  const [message, setMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const [ready, setReady] = useState(false)
  const url = '/api/' + kind

  async function read() {
    const response = await fetchAPI(url + '?history=true')
    if (!response.ok) throw new Error('Could not load revision history')
    return await response.json()
  }
  useEffect(() => {
    let live = true
    read().then(data => { if (live) { setRows(data); setReady(true) } })
      .catch(err => { if (live) setMessage(err.message) })
    return () => { live = false }
  }, [url,fetchAPI])

  const heads = new Map(rows.map(row => [idOf(row), row]))
  const history = rows.filter(row => idOf(row) === selected).slice().reverse()
  const active = isRule ? history.find(row => row.Active) : null
  function select(id, head = heads.get(id)) {
    setSelected(id)
    setDraft(head ? { ...head, instructions: head.Config?.instructions || '' } : isRule ? blankRule : blankAgent)
    setBase(head ? versionOf(head) : 0)
    setMessage('')
  }
  async function save(promote) {
    setSaving(true)
    let body
    if (isRule) {
      body = promote ? { ID: selected, Promote: promote } : {
        ID: selected, Description: draft.Description, Condition: draft.Condition, Action: draft.Action
      }
    } else {
      body = { Name: draft.Name, Role: draft.Role, ModelTier: draft.ModelTier, Version: draft.Version,
        Config: { ...draft.Config, instructions: draft.instructions }, Tools: draft.Tools, Retired: draft.Retired }
    }
    try {
      const response = await fetchAPI(url, { method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...body, Expected: base }) })
      if (response.status === 409) {
        setRows(await read())
        setMessage('Another edit was saved first. Select the definition again to load its latest revision.')
        return
      }
      if (!response.ok) throw new Error((await response.text()).trim() || 'Could not save')
      const saved = await response.json()
      const data = await read()
      setRows(data)
      const id = idOf(saved)
      select(id, data.filter(row => idOf(row) === id).at(-1))
      setMessage('Saved revision ' + versionOf(saved))
    } catch (err) { setMessage(err.message) }
    finally { setSaving(false) }
  }
  const field = (key, label, locked = false) => <label key={key}>{label}
    <input required={['Description','Name','Role'].includes(key)} value={draft[key] || ''}
      disabled={locked || saving} onChange={event => setDraft({ ...draft, [key]: event.target.value })} /></label>
  function limitInput(key, label, fallback) {
    return <label>{label}<input type="number" min="1" max={key === 'max_attempts' ? 16 : undefined}
      placeholder={String(fallback)} value={draft.Config?.[key] ?? ''} disabled={saving}
      onChange={event => {
        const Config = {...draft.Config}
        if (event.target.value) Config[key] = event.target.value
        else delete Config[key]
        setDraft({...draft, Config})
      }}/></label>
  }
  return <section className="definitions" aria-label={isRule ? 'Rule manager' : 'Agent definitions'}>
    <h2>{isRule ? 'Rule manager' : 'Agent definitions'}</h2>
    <div className="definition-layout"><div><button disabled={!ready || saving} onClick={() => select('')}>New {isRule ? 'rule' : 'agent'}</button>
      {Array.from(heads, ([id,row]) => <button disabled={saving} key={id} aria-pressed={selected === id}
        onClick={() => select(id)}>{isRule ? row.Description : row.Name} · r{versionOf(row)}</button>)}</div>
      <div><form onSubmit={event => { event.preventDefault(); save() }}>
        {isRule ? [field('Description','Rule'), field('Condition','Condition'), field('Action','Action')] :
          [field('Name','Agent name',Boolean(selected)),field('Role','Role'),field('ModelTier','Model tier'),
            field('Version','Build version'),field('instructions','Instructions')]}
        {!isRule && ['worker', 'judge'].includes(draft.Role) &&
          limitInput('max_output_tokens', 'Maximum output tokens per call', draft.Role === 'judge' ? 300 : 16000)}
        {!isRule && draft.Role === 'worker' && limitInput('max_attempts', 'Maximum worker attempts', 2)}
        {!isRule && selected && <label><input type="checkbox" checked={draft.Retired} onChange={e => setDraft({...draft, Retired:e.target.checked})} /> Retired</label>}
        <button className="ok" disabled={!ready || saving}>{isRule ? 'Save proposal' : 'Save definition'}</button>
      </form>
      {selected && <><h3>Revision history</h3>{history.map(row => <article key={versionOf(row)}>
        <strong>r{versionOf(row)}</strong> · {row.CreatedBy} · {row.ChangedAt || row.CreatedAt ? new Date(row.ChangedAt || row.CreatedAt).toLocaleString() : 'Legacy revision'}
        <p>{isRule ? row.Description : row.Role + ' · ' + row.ModelTier}</p>
        {isRule && (row.Version === active?.Version ? <span>Active</span> :
          <button disabled={saving} onClick={() => save(row.Version)}>Activate r{row.Version}</button>)}
        <details><summary>Snapshot</summary><pre>{JSON.stringify(row, null, 2)}</pre></details>
      </article>)}</>}</div></div>
    <p role="status">{message}</p>
  </section>
}
