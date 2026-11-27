import { useState } from 'react'

export default function Runs({runs, source, setSource, request, refresh, setMessage, page}) {
  const [goal,setGoal] = useState('')
  const [definition,setDefinition]=useState(''), [expected,setExpected]=useState(0)
  const [selected,setSelected] = useState('')
  const [record,setRecord] = useState('')
  const [note,setNote] = useState('')
  const [busy,setBusy] = useState(false)
  const run = runs.find(item=>item.id===selected) || runs[0]
  const selectRun = <label>Run <select value={run?.id || ''} onChange={e=>{setSelected(e.target.value);setRecord('');setNote('')}}>
    {runs.map(r=><option key={r.id} value={r.id}>{r.description} · {r.id.slice(0,8)}</option>)}</select></label>
  async function action(fn) { setBusy(true); try { await fn(); await refresh() } catch(e) {setMessage(e.message)} finally {setBusy(false)} }
  if(page==='Goal builder') return <form className="definitions" onSubmit={e=>{e.preventDefault();action(async()=>{
    const records=JSON.parse(source.records)
    if(!Array.isArray(records)||!records.length) throw new Error('Supply at least one source record.')
    const r=await request('/api/goals',{description:goal, factory:source.factory,records,max_tokens:Number(source.budget)})
    setSelected(r.id);setMessage('Started run '+r.id)
  })}}><h2>Create a goal</h2><label>Outcome<input required value={goal} onChange={e=>setGoal(e.target.value)}/></label>
    <label>Factory definition ID<input required value={source.factory} onChange={e=>setSource({...source,factory:e.target.value})}/></label>
    <details><summary>Create or revise this factory definition</summary>
      <label>Expected revision<input type="number" min="0" value={expected} onChange={e=>setExpected(Number(e.target.value))}/></label>
      <label>Definition JSON<textarea rows={4} value={definition} placeholder='{"record_type":"request","fields":{"assigned_team":"string"}}' onChange={e=>setDefinition(e.target.value)}/></label>
      <button type="button" disabled={busy||!source.factory} onClick={()=>action(async()=>{const d=await request('/api/definitions/factory/'+encodeURIComponent(source.factory),{Expected:expected,Data:JSON.parse(definition)});setExpected(d.version);setMessage('Saved factory revision '+d.version)})}>Save definition</button></details>
    <label>Source records (JSON array)<textarea required rows={6} value={source.records} onChange={e=>setSource({...source,records:e.target.value})}/></label>
    <label>Token budget (0 uses workspace settings)<input type="number" min="0" value={source.budget} onChange={e=>setSource({...source,budget:e.target.value})}/></label>
    <p>Uses the selected factory revision and its active workspace rules.</p>
    <button className="ok" disabled={busy}>Start run</button></form>
  if(page==='Graph') return <section className="definitions"><h2>Record graph</h2>{selectRun}
    {run?.records?.map(e=><article key={e.id}><button onClick={()=>setRecord(e.id)}>{e.id}</button> <span>{e.type}</span>
    {e.relations?.map((rel,i)=><p key={i}>↳ {rel.type} → <button onClick={()=>setRecord(rel.target)}>{rel.target}</button></p>)}</article>)}
    {record && <aside><h3>{record}</h3><pre>{JSON.stringify(run?.records?.find(e=>e.id===record)?.attributes || {},null,2)}</pre>
    {run?.decisions?.filter(d=>d.entity_id===record).map(d=><p key={d.id}>{d.status} · {d.reason}</p>)}</aside>}</section>
  if(page==='Decisions') return <section className="definitions"><h2>Decision review</h2>{selectRun}
    {!run?.decisions?.length && <p>No decisions for this run.</p>}
    {run?.decisions?.map(d=><article key={d.id}><h3>{d.entity_id} · {d.status}</h3><p>{d.reason}</p>
      <table><thead><tr><th>Field</th><th>Before</th><th>Proposed</th></tr></thead><tbody>{Object.entries(d.after).map(([k,v])=><tr key={k}><td>{k}</td><td>{JSON.stringify(d.before?.[k]) ?? 'Unset'}</td><td>{JSON.stringify(v)}</td></tr>)}</tbody></table>
      <details><summary>Evidence and revisions</summary><pre>{JSON.stringify(d.proposal,null,2)}</pre></details>
      {d.status==='pending' && run.status==='in_review' && <><label>Reviewer note<input value={note} onChange={e=>setNote(e.target.value)}/></label>
        {['approved','rejected'].map(s=><button disabled={busy} key={s} onClick={()=>action(async()=>{await request('/api/decisions',{Run:run.id,ID:d.id,Expected:run.version,Status:s,Feedback:note});setNote('')})}>{s==='approved'?'Approve':'Reject'}</button>)}</>}</article>)}</section>
  return <section className="definitions"><h2>Workspace overview</h2>
    <p>{runs.filter(r=>['queued','running'].includes(r.status)).length} active runs · {runs.filter(r=>r.status==='in_review').length} awaiting review</p>
    {!runs.length && <p>No runs yet. Open Goal builder to start one.</p>}
    {runs.map(r=><article key={r.id}><strong>{r.description}</strong> · {r.status}<p>{r.id}</p>
    <p>{r.decisions?.length || 0} of {r.records?.length || 0} records evaluated · {r.calls?.reduce((n,c)=>n+c.tokens,0) || 0} tokens</p>
    {r.error && <p role="alert">{r.error}</p>}<details><summary>Run details</summary><pre>{JSON.stringify({factory:r.plan.factory,revision:r.plan.revision,settings_revision:r.settings_revision,model_calls:r.calls},null,2)}</pre></details>
    {['queued','running','created','in_review'].includes(r.status)&&<button disabled={busy} onClick={()=>action(()=>request('/api/goals/'+r.id,undefined,'DELETE'))}>Cancel run</button>}</article>)}</section>
}
