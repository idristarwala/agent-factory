import { useCallback, useEffect, useState } from 'react'
import {parseSample, runSource} from './sourceInput'
import GoalProposal from './GoalProposal'
import ChatAttachments from './ChatAttachments'
import {costLabel} from './RunUsage'
import Settings from './Settings'
import Definitions from './Definitions'
import Runs from './Runs'
import './App.css'

const API = (import.meta.env.VITE_API_URL || 'http://localhost:8080').replace(/\/+$/, '')
function apiFetch(path, token, options = {}) {
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', 'Bearer ' + token)
  return fetch(API + path, {...options, headers, redirect: 'error'})
}

function Connect({onConnect}) {
  const [token,setToken]=useState('')
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  async function connect(event) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      const value=token.trim()
      const response=await apiFetch('/api/settings',value)
      if(!response.ok) throw new Error(response.status===401?'The API rejected this token.':'Could not connect to the workspace.')
      onConnect(value)
    } catch(err) {setError(err.message)}
    finally {setBusy(false)}
  }
  return <main className="definitions connect"><h1>Connect to Agent Factory</h1>
    <form onSubmit={connect}><label>Workspace API token<input type="password" autoComplete="off"
      value={token} onChange={e=>setToken(e.target.value)} required autoFocus/></label>
      <button disabled={busy || !token.trim()}>Connect</button></form>
    {error && <p role="alert">{error}</p>}</main>
}

export default function App(){
  const [session,setSession]=useState({token:'',locked:false})
  const disconnect=useCallback(()=>setSession(current=>current===session?{token:'',locked:true}:current),[session])
  const fetchAPI=useCallback(async(path,options)=>{
    const response=await apiFetch(path,session.token,options)
    if(response.status===401) disconnect()
    return response
  },[session,disconnect])
  if(session.locked) return <Connect onConnect={token=>setSession({token,locked:false})}/>
  return <Workspace fetchAPI={fetchAPI} disconnect={session.token?disconnect:null}/>
}

function Workspace({fetchAPI,disconnect}){
  const [page,setPage]=useState('Workspace'), [runs,setRuns]=useState([]), [message,setMessage]=useState('')
  const [source,setSource]=useState({factory:'',records:'[]',budget:0})
  const [chat,setChat]=useState([]),[input,setInput]=useState(''),[sending,setSending]=useState(false)
  const [files,setFiles]=useState([]),[uploading,setUploading]=useState(false)
  const [draft,setDraft]=useState(null)
  const [usage,setUsage]=useState({tokens_used:0})
  const request=useCallback(async(path,body,method,lossless=false)=>{
    const r=await fetchAPI(path,{method:method || (body?'POST':'GET'),headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined})
    if(!r.ok)throw new Error((await r.text()).trim() || 'Request failed')
    return lossless ? parseSample(await r.text()) : await r.json()
  },[fetchAPI])
  const refresh=useCallback(async()=>{const [r,u]=await Promise.all([request('/api/goals?view=review'),request('/api/usage')]);setRuns(r);setUsage(u)},[request])
  useEffect(()=>{refresh().catch(e=>setMessage(e.message));const id=setInterval(()=>refresh().catch(e=>setMessage(e.message)),5000);return ()=>clearInterval(id)},[refresh])
  async function send(e){e.preventDefault();if(!input.trim()||sending||uploading)return;setSending(true)
    try{const text=input.trim();const body={message:text,artifacts:files.map(file=>file.ref)}
      if(/^(?:please\s+)?(?:(?:can|could) you\s+)?start\b/i.test(text)&&source.factory) body.source=runSource(source)
      const reply=await request('/api/chat',body)
      if(reply.proposal){setDraft(reply);setPage('Goal proposal')}
      if(reply.factory){setSource({factory:reply.factory,definition:reply.definition,expected:1,records:'[]',budget:0});setPage('Goal builder')}
      setChat(c=>[...c,{role:'user',text,files},{role:'bot',text:reply.reply}]);setInput('');await refresh()
    }catch(err){setMessage(err.message)}finally{setSending(false)}}
  async function review(feedback) {
    setSending(true)
    try {
      if(feedback){
        const next=await request('/api/chat',{message:'Create a factory to '+feedback,proposal:draft.proposal,expected:draft.version,artifacts:files.map(f=>f.ref)})
        setDraft(next)
      } else {
        const reply=await request('/api/chat/proposals/'+draft.proposal+'/approve',{version:draft.version})
        setSource({factory:reply.factory,definition:reply.definition,expected:reply.version,records:'[]',budget:0,goal:reply.goal})
        setDraft(null);setPage('Goal builder')
      }
      setMessage('')
    } catch(error){setMessage(error.message)} finally {setSending(false)}
  }
  return <div className="app"><header><h1>Agent Factory</h1><span>{usage.run_tokens || 0} run tokens · {usage.model_calls || 0} model calls · {usage.tool_calls || 0} tool calls · {costLabel(usage.estimated_usd,usage.unpriced_calls)}</span>{disconnect && <button onClick={disconnect}>Disconnect</button>}</header>
    <div className="layout"><main><nav aria-label="Workspace navigation">{['Workspace',...(draft?['Goal proposal']:[]),'Goal builder','Decisions','Graph','Rules','Agents','Settings'].map(n=>
      <button key={n} aria-pressed={n===page} onClick={()=>{setPage(n);setMessage('')}}>{n}</button>)}</nav>
    {page==='Goal proposal'&&draft?<GoalProposal request={request} draft={draft} busy={sending||uploading} approve={()=>review()} revise={review}/>:page==='Settings'?<Settings fetchAPI={fetchAPI}/>:['Rules','Agents'].includes(page)?<Definitions fetchAPI={fetchAPI} kind={page.toLowerCase()} key={page}/>:
      <Runs {...{page,runs,source,setSource,request,refresh,setMessage}}/>}<p role="status">{message}</p></main>
    <aside className="chat"><h2 className="chat-header">Factory chat</h2><div className="messages">{chat.map((m,i)=><p className={'msg '+m.role} key={i}>{m.text}{m.files?.length>0 && <small className="chat-citations">{m.files.map(f=>f.name+' · r'+f.ref.version).join(', ')}</small>}</p>)}</div>
    <ChatAttachments files={files} setFiles={setFiles} busy={uploading||sending} setBusy={setUploading} request={request} setMessage={setMessage}/>
    <form className="chat-input" onSubmit={send}><input aria-label="Message" placeholder="Create a factory to…, start a run, or check status" value={input} onChange={e=>setInput(e.target.value)}/><button disabled={sending||uploading}>Send</button></form></aside></div></div>
}
