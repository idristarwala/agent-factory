export function costLabel(amount,unpriced) {
  if (!amount && unpriced) return 'Pricing unavailable'
  const cost=new Intl.NumberFormat('en-US',{style:'currency',currency:'USD',minimumFractionDigits:2,maximumFractionDigits:6}).format(amount || 0)
  return cost+' estimated'+(unpriced?' · '+unpriced+' unpriced calls':'')
}

export default function RunUsage({run}) {
  const calls=run.calls || []
  const tools=Object.entries(run.tool_calls_by_name || {})
  const cost=calls.reduce((sum,call)=>sum+call.estimated_usd,0)
  return <section><h4>Run usage</h4>
    <p>{calls.length} model calls · {tools.reduce((sum,[,count])=>sum+count,0)} tool calls · {costLabel(cost,calls.filter(call=>!call.priced && !call.failed).length)}</p>
    {calls.length>0 && <table><thead><tr>{['Role','Model','Input tokens','Output tokens','Estimated cost'].map(label=><th key={label}>{label}</th>)}</tr></thead>
      <tbody>{calls.map((call,i)=><tr key={i}><td>{call.role}</td><td>{call.model || 'Unknown'}</td><td>{call.failed?'—':call.input_tokens}</td><td>{call.failed?'—':call.output_tokens}</td><td>{call.failed?'Failed':costLabel(call.estimated_usd,call.priced?0:1)}</td></tr>)}</tbody></table>}
    {tools.length>0 && <ul>{tools.map(([name,count])=><li key={name}>{name} · {count}</li>)}</ul>}
  </section>
}
