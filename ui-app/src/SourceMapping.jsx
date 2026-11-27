import {useState} from 'react'
import {parseSample} from './sourceInput'

export default function SourceMapping({factory, goal, approved, onApprove, request, setMessage}) {
  const [sample,setSample] = useState(approved?.sample || '')
  const [mapping,setMapping] = useState(approved?.mapping || factory)
  const [feed,setFeed] = useState('')
  const [spec,setSpec] = useState({document:'',path:'',method:'GET',status:'200'})
  function updateSpec(name,value) {
    reset()
    setSpec(previous=>({...previous,[name]:value}))
  }
  const [collection,setCollection] = useState(undefined)
  const [identity,setIdentity] = useState([])
  const [result,setResult] = useState(null)
  const [busy,setBusy] = useState(false)
  const [expected,setExpected] = useState(0)
  const base = '/api/sources/'
  const id = encodeURIComponent(mapping)

  function reset() {
    setResult(null)
    setCollection(undefined)
    setIdentity([])
    onApprove(null)
  }

  async function perform(fn) {
    setBusy(true)
    try { await fn() } catch (error) { setMessage(error.message) }
    finally { setBusy(false) }
  }

  async function propose() {
    const payload = parseSample(sample)
    const [defs, drafts, mappings] = await Promise.all([
      request('/api/definitions/factory/' + encodeURIComponent(factory)),
      request(base + 'proposals/' + id), request(base + 'mappings/' + id)
    ])
    const recordType = defs.at(-1)?.data?.record_type
    if (!recordType) throw new Error('Save a factory definition before proposing a source mapping.')
    const response = await request(base + 'proposals/' + id, {
      sample: payload, goal, record_type: recordType, expected: drafts.length, collection, identity, openapi: spec.document.trim() ? spec : undefined
    },undefined,true)
    setResult(response)
    setExpected(mappings.length)
    if (response.status === 'choose_identity') setCollection(response.collection.path)
  }

  async function approve() {
    const saved = await request(base + 'mappings/' + id, {
      sample: parseSample(sample), expected, proposal_revision: result.proposal.version
    })
    onApprove({mapping, revision:saved.version, sample})
    setMessage('Approved mapping revision ' + saved.version)
  }

  return <fieldset disabled={busy}><legend>API source mapping</legend>
    <label>Mapping ID<input value={mapping} onChange={e=>{reset();setMapping(e.target.value)}}/></label>
    <details><summary>Fetch a configured API source</summary>
      <label>Source name<input value={feed} onChange={e=>setFeed(e.target.value)}/></label>
      <button type="button" disabled={!factory || !feed} onClick={()=>perform(async()=>{
        const response = await request(base+encodeURIComponent(factory)+'/'+encodeURIComponent(feed)+'/sample',{include_sample:true},undefined,true)
        reset();setSample(JSON.stringify(response.sample,null,2))
      })}>Fetch sample</button></details>
    <label>Source API response<textarea rows={6} value={sample} onChange={e=>{reset();setSample(e.target.value)}}/></label>
    <details><summary>Compare with an OpenAPI response</summary>
      <label>OpenAPI document (JSON or YAML)<textarea rows={5} value={spec.document} onChange={e=>updateSpec('document',e.target.value)}/></label>
      <label>Operation path<input value={spec.path} onChange={e=>updateSpec('path',e.target.value)} placeholder="/records"/></label>
      <label>Operation method<input value={spec.method} onChange={e=>updateSpec('method',e.target.value)}/></label>
      <label>Response status<input value={spec.status} onChange={e=>updateSpec('status',e.target.value)}/></label>
    </details>
    {result?.declared_fields && <details><summary>Declared and observed fields</summary>
      <table><thead><tr><th>Path</th><th>Declared type</th><th>Observed types</th></tr></thead><tbody>
        {Array.from(new Set([...result.declared_fields,...(result.observed_fields || result.collection?.fields || [])].map(f=>f.path))).sort().map(path=>{
          const declared=result.declared_fields.find(f=>f.path===path)
          const observed=(result.observed_fields || result.collection?.fields || []).find(f=>f.path===path)
          return <tr key={path}><td>{path}</td><td>{declared ? (declared.types?.join(', ') || 'Unspecified')+(declared.required?' · required':'') : 'Undeclared'}</td><td>{observed ? Object.keys(observed.types).join(', ') : 'Absent in sample'}</td></tr>
        })}
      </tbody></table>
    </details>}
    {result?.status==='choose_collection' && <label>Record collection<select value={collection ?? '__choose__'} onChange={e=>setCollection(e.target.value)}>
      <option disabled value="__choose__">Choose a collection</option>
      {result.collections.map(c=><option key={c.path} value={c.path}>{c.path || '(root)'} · {c.records} records</option>)}
    </select></label>}
    {result?.status==='choose_identity' && <fieldset><legend>Identity fields, in key order</legend>
      {result.problem && <p role="alert">Row {result.row}: {result.problem}</p>}
      {result.collection.fields.filter(f=>Object.hasOwn(f.types,'string') || Object.hasOwn(f.types,'number')).map(f=><label key={f.path}>
        <input type="checkbox" checked={identity.includes(f.path)} onChange={e=>setIdentity(current=>e.target.checked?[...current,f.path]:current.filter(p=>p!==f.path))}/>
        {f.path}{identity.includes(f.path) && ' · '+(identity.indexOf(f.path)+1)}
      </label>)}
    </fieldset>}
    {!approved && <button type="button" disabled={!factory || !mapping || !goal || !sample || (result?.status==='choose_collection' && collection===undefined) || (result?.status==='choose_identity' && !identity.length)} onClick={()=>perform(propose)}>Propose mapping</button>}
    {result?.status==='review' && <section><h3>Mapping preview · draft {result.proposal.version}</h3>
      <table><thead><tr><th>Source path</th><th>Record field</th></tr></thead><tbody>
        {Object.entries(result.proposal.data.fields).map(([name,path])=><tr key={name}><td>{path}</td><td>{name}</td></tr>)}
      </tbody></table>
      <p>Identity: {result.proposal.data.identity.join(', ')}</p>
      <details><summary>Preview {result.rows.length} mapped records</summary><pre>{JSON.stringify(result.rows,null,2)}</pre></details>
      {!approved && <button type="button" onClick={()=>perform(approve)}>Approve mapping for this run</button>}
    </section>}
    {approved && <>
      <p role="status">Using {approved.mapping} · revision {approved.revision}</p>
      <details><summary>Import another payload with this mapping</summary>
        <label>Run payload<textarea rows={6} value={approved.payload ?? approved.sample}
          onChange={e=>onApprove({...approved,payload:e.target.value})}/></label>
        <label><input type="checkbox" checked={approved.allowPartial || false}
          onChange={e=>onApprove({...approved,allowPartial:e.target.checked})}/> Process valid rows and report rejected rows</label>
      </details>
    </>}
  </fieldset>
}
