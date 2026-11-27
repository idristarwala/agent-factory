import BrowserCapture from "./BrowserCapture"
import RunUsage from './RunUsage'
import SourceMapping from './SourceMapping'
import {runSource} from './sourceInput'
import RecordGraph from './RecordGraph'
import { useEffect, useState } from 'react'

export default function Runs({runs, source, setSource, request, refresh, setMessage, page}) {
  const [availableRules,setAvailableRules]=useState([])
  useEffect(()=>{
    if(page!=='Goal builder') return
    let live=true
    request('/api/rules').then(rows=>{if(live)setAvailableRules(rows)}).catch(e=>{if(live)setMessage(e.message)})
    return ()=>{live=false}
  },[page,request,setMessage])
  const goal=source?.goal || ''
  const setGoal=value=>setSource(previous=>({...previous,goal:value}))
  const definition=source?.definition || '', expected=source?.expected || 0
  const [selected,setSelected] = useState('')
  const [notes,setNotes] = useState({})
  const [busy,setBusy] = useState(false)
  const run = runs.find(item=>item.id===selected) || runs[0]
  const selectRun = <label>Run <select value={run?.id || ''} onChange={e=>{setSelected(e.target.value)}}>
    {runs.map(r=><option key={r.id} value={r.id}>{r.description} · {r.id.slice(0,8)}</option>)}</select></label>
  async function action(fn) { setBusy(true); try { await fn(); await refresh() } catch(e) {setMessage(e.message)} finally {setBusy(false)} }
  if(page==='Goal builder') return <form className="definitions" onSubmit={e=>{e.preventDefault();action(async()=>{
    const input=runSource(source)
    if(!input.source && (!Array.isArray(input.records)||!input.records.length)) throw new Error('Supply at least one source record.')
    const r=await request('/api/goals',{description:goal,...input})
    setSelected(r.id);setMessage('Started run '+r.id)
  })}}><h2>Create a goal</h2><label>Outcome<input required value={goal} onChange={e=>setGoal(e.target.value)}/></label>
    <label>Factory definition ID<input required value={source.factory} onChange={e=>setSource({...source,factory:e.target.value,approved:null,useGoalArtifacts:false,expected:0,definition:''})}/></label>
    <details><summary>Create or revise this factory definition</summary>
      <label>Expected revision<input type="number" min="0" value={expected} onChange={e=>setSource({...source,expected:Number(e.target.value)})}/></label>
      <label>Definition JSON<textarea rows={4} value={definition} placeholder='{"record_type":"request","fields":{"assigned_team":"string"}}' onChange={e=>setSource({...source,definition:e.target.value})}/></label>
      <button type="button" disabled={busy||!source.factory} onClick={()=>action(async()=>{const d=await request('/api/definitions/factory/'+encodeURIComponent(source.factory),{Expected:expected,Data:JSON.parse(definition)});setSource(previous=>({...previous,expected:d.version}));setMessage('Saved factory revision '+d.version)})}>Save definition</button></details>
    <label>Input format<select value={source.inputMode || 'records'} onChange={e=>setSource({...source,inputMode:e.target.value})}><option value="records">Factory records</option><option value="mapped">Source API response</option></select></label>
    {source.inputMode==='mapped' ? <SourceMapping key={source.factory} factory={source.factory} goal={goal} approved={source.approved} request={request} setMessage={setMessage} onApprove={approved=>setSource(prev=>prev.factory===source.factory?{...prev,approved}:prev)}/> :
<label>Source records (JSON array)<textarea required rows={6} value={source.records} onChange={e=>setSource({...source,records:e.target.value})}/></label>}
    <label><input type="checkbox" checked={source.useGoalArtifacts || false} onChange={e=>setSource({...source,useGoalArtifacts:e.target.checked})}/> Include pinned goal attachments in worker and judge evidence</label>
    <label>Token budget (0 uses workspace settings)<input type="number" min="0" value={source.budget} onChange={e=>setSource({...source,budget:e.target.value})}/></label>
    <fieldset><legend>Run rules</legend><label><input type="checkbox" checked={source.rules===undefined}
      onChange={e=>setSource({...source,rules:e.target.checked?undefined:Object.fromEntries(availableRules.map(r=>[r.ID,r.Version]))})}/> Use all active rules</label>
      {source.rules!==undefined && availableRules.map(r=><label key={r.ID}><input type="checkbox" checked={source.rules[r.ID]===r.Version}
        onChange={e=>{const rules={...source.rules};if(e.target.checked)rules[r.ID]=r.Version;else delete rules[r.ID];setSource({...source,rules})}}/> {r.Description} · r{r.Version}</label>)}</fieldset>
    <button className="ok" disabled={busy || (source.inputMode==='mapped' && !source.approved)}>Start run</button></form>
  const decisionCard = d => <article key={d.id}><h3>{d.entity_id} · {d.status}</h3><p>{d.reason}</p>
      <div className="decision-layout"><div><table><thead><tr><th>Field</th><th>Before</th><th>Proposed</th></tr></thead><tbody>{Object.entries(d.after).map(([k,v])=><tr key={k}><td>{k}</td><td>{JSON.stringify(d.before?.[k]) ?? 'Unset'}</td><td>{JSON.stringify(v)}</td></tr>)}</tbody></table></div>
      <aside aria-label="Decision evidence"><h4>Evidence</h4>
        {d.browser_capture && <BrowserCapture key={d.browser_capture} id={d.browser_capture} request={request}/>}
        {d.proposal?.tool_proposal && <section aria-label="External change proposal">
          <h4>External change proposal</h4>
          <p>{d.proposal.tool_proposal.name} · r{d.proposal.tool_proposal.tool_revision} · {d.proposal.tool_proposal.side_effect}</p>
          <p>Pack {d.proposal.tool_proposal.pack} · r{d.proposal.tool_proposal.pack_revision}</p>
          <pre>{JSON.stringify(d.proposal.tool_proposal.arguments,null,2)}</pre>
          <p>Approval records your decision. External delivery is a separate operation.</p>
        </section>}
        <details><summary>Source record</summary><pre>{JSON.stringify(d.before,null,2)}</pre></details>
        {d.lineage && <details><summary>Source lineage</summary>
          <p>Mapping {d.lineage.mapping} · revision {d.lineage.revision}</p>
          <p>{d.lineage.feed ? 'Feed '+d.lineage.feed : 'Submitted payload'}{d.lineage.import && ' · import '+d.lineage.import.id+' · page '+d.lineage.import.page}</p>
          <p>Record path <code>{d.lineage.pointer || '(root)'}</code></p>
          <p>Source SHA-256 <code>{d.lineage.sha256}</code></p>
        </details>}
        <ul>{d.proposal?.Evidence?.filter(item=>!d.lineage || !item.startsWith('Source lineage: ')).map((item,i)=><li key={i}>{item}</li>)}</ul>
        {d.proposal?.AgentName && <p>{d.proposal.AgentName} · r{d.proposal.AgentRevision}</p>}
        {Object.entries(d.proposal?.RuleRevisions || {}).map(([id,version])=><p key={id}>{id} · r{version}</p>)}
        {d.feedback && <p>Reviewer feedback · {d.feedback}</p>}
      </aside></div>
      {d.status==='pending' && run.status==='in_review' && <><label>Reviewer note<input value={notes[d.id] || ''} onChange={e=>setNotes({...notes,[d.id]:e.target.value})}/></label>
        {['approved','rejected'].map(s=><button disabled={busy} key={s} onClick={()=>action(async()=>{await request('/api/decisions',{Run:run.id,ID:d.id,Expected:run.version,Status:s,Feedback:notes[d.id] || ''});setNotes(previous=>({...previous,[d.id]:''}))})}>{s==='approved'?'Approve':'Reject'}</button>)}</>}</article>
  if(page==='Graph') return <section className="definitions"><h2>Record graph</h2>{selectRun}
    <RecordGraph key={run?.id} run={run} decisionCard={decisionCard}/>
  </section>
  if(page==='Decisions') return <section className="definitions"><h2>Decision review</h2>{selectRun}
    {!run?.decisions?.length && <p>No decisions for this run.</p>}
    {run?.decisions?.map(decisionCard)}</section>
  const recent = runs.flatMap(r => (r.decisions || []).map(d => ({ decision: d, run: r })))
    .sort((a,b) => decisionTime(b) - decisionTime(a)).slice(0,5)
  function decisionTime({decision,run}) {
    const updated = Date.parse(decision.updated_at)
    return updated > 0 ? updated : Date.parse(run.created_at) || 0
  }
  return <section className="definitions"><h2>Workspace overview</h2>
    <p>{runs.filter(r=>['queued','running'].includes(r.status)).length} active runs · {runs.filter(r=>r.status==='in_review').length} awaiting review</p>
    <h3>Recent decisions</h3>
    {recent.length === 0 ? <p>No decisions yet.</p> : <ul>{recent.map(({decision,run}) =>
      <li key={decision.id}><strong>{decision.entity_id}</strong> · {decision.status} · {run.description}</li>)}</ul>}
    <h3>Runs</h3>
    {!runs.length && <p>No runs yet. Open Goal builder to start one.</p>}
    {runs.map(r=><article key={r.id}><strong>{r.description}</strong> · {r.status}<p>{r.id}</p>
    <p>{r.runner === "continuous" ? "Continuous runner" : `${r.decisions?.length || 0} of ${r.records?.length || 0} records evaluated`} · {r.calls?.reduce((n,c)=>n+c.tokens,0) || 0} tokens</p>
    {r.import_problems?.length > 0 && <details><summary>{r.import_problems.length} rejected source {r.import_problems.length===1 ? 'row' : 'rows'}</summary>
      <ul>{r.import_problems.map(p=><li key={p.index}>Row {p.index + 1}: {p.reason}</li>)}</ul></details>}
    {r.error && <p role="alert">{r.error}</p>}<details><summary>Run details</summary><RunUsage run={r}/><pre>{JSON.stringify({factory:r.plan.factory,revision:r.plan.revision,settings_revision:r.settings_revision},null,2)}</pre></details>
    {['queued','running','created','in_review'].includes(r.status)&&<button disabled={busy} onClick={()=>action(()=>request('/api/goals/'+r.id,undefined,'DELETE'))}>Cancel run</button>}</article>)}</section>
}
