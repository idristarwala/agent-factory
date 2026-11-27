import {useState} from 'react'

function Citation({citation:c,request}) {
  const [source,setSource]=useState(null)
  const [error,setError]=useState('')
  async function inspect(e) {
    if (!e.currentTarget.open || source) return
    try {setSource(await request('/api/chat/artifacts/'+c.id+'/'+c.version+'?location='+encodeURIComponent(c.location)));setError('')}
    catch(err){setError(err.message)}
  }
  return <details onToggle={inspect}><summary>{c.name} · {c.location} · revision {c.version}</summary>
    {c.source && <a href={c.source} target="_blank" rel="noreferrer">Open GitHub source</a>}
    {error && <p role="alert">{error}</p>}
    {source?.text && <pre>{source.text}</pre>}
    {source?.image && <img style={{maxWidth:'100%'}} alt={c.name} src={'data:'+source.mime+';base64,'+source.image}/>}
  </details>
}
export default function GoalProposal({draft,busy,approve,revise,request}) {
  const spec=draft.design
  return <section className="definitions"><h2>Review goal proposal · revision {draft.version}</h2>
    <p>{spec.goal}</p>
    <p>Record type: <code>{spec.record_type}</code></p>
    <h3>Editable fields</h3>
    <ul>{Object.entries(spec.fields).map(([name,type])=><li key={name}>{name} · {type}</li>)}</ul>
    <h3>Acceptance criteria</h3>
    <ol>{spec.criteria.map((criterion,i)=><li key={i}>{criterion.text}
      <ul>{criterion.citations.map(c=><li key={JSON.stringify([c.id,c.version,c.location])}>
        <Citation citation={c} request={request}/>
      </li>)}</ul>
    </li>)}</ol>
    <form onSubmit={e=>{e.preventDefault();revise(new FormData(e.currentTarget).get('feedback'))}}>
      <label>Requested changes<textarea name="feedback" required rows={2}/></label>
      <button disabled={busy}>Revise proposal</button>
    </form>
    <button className="ok" disabled={busy} onClick={approve}>Approve goal and create factory</button>
  </section>
}
