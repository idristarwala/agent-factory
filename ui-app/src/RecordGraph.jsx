import { useState } from 'react'

export default function RecordGraph({run,decisionCard}) {
  const records = run?.records || []
  const [selected,setSelected] = useState(records[0]?.id || '')
  const [decisionId,setDecisionId] = useState('')
  if (!records.length) return <p>No source records in this run.</p>
  const id = selected || records[0].id
  const record = records.find(item => item.id === id)
  const incoming = records.flatMap(item => (item.relations || []).filter(rel => rel.target === id)
    .map(rel => ({id:item.id,type:rel.type})))
  const outgoing = (record?.relations || []).map(rel => ({id:rel.target,type:rel.type}))
  const decisions = (run.decisions || []).filter(d => d.entity_id === id)
  const decision = decisions.find(d => d.id === decisionId)
  function navigate(next) { setSelected(next); setDecisionId('') }
  function neighbors(links,direction) {
    return <div><h4>{direction}</h4>{links.length ? links.map((link,i)=>
      <p key={i}><span>{link.type}</span> <button onClick={()=>navigate(link.id)}>{link.id}</button></p>) : <p>None</p>}</div>
  }
  return <><label>Record <select value={id} onChange={e=>navigate(e.target.value)}>
    {!record && <option value={id}>{id}</option>}
    {records.map(item=><option key={item.id}>{item.id}</option>)}</select></label>
    <div className="record-neighborhood" aria-label="Record relationships">
      {neighbors(incoming,'Incoming')}<div><h3>{id}</h3><p>{record?.type || 'Referenced record outside this run'}</p></div>{neighbors(outgoing,'Outgoing')}
    </div>
    {record && <details><summary>Record attributes</summary><pre>{JSON.stringify(record.attributes,null,2)}</pre></details>}
    <h3>Linked decisions</h3>
    {decisions.length ? decisions.map(d=><button key={d.id} onClick={()=>setDecisionId(d.id)}>Open decision {d.id} · {d.status}</button>) : <p>No linked decisions.</p>}
    {decision && decisionCard(decision)}</>
}
