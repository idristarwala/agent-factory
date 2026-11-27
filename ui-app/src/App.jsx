import { useCallback, useEffect, useState } from 'react'
import {parseSample, runSource} from './sourceInput'
import ChatAttachments from './ChatAttachments'
import Settings from './Settings'
import Definitions from './Definitions'
import Runs from './Runs'
import './App.css'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'
export default function App(){
  const [page,setPage]=useState('Workspace'), [runs,setRuns]=useState([]), [message,setMessage]=useState('')
  const [source,setSource]=useState({factory:'',records:'[]',budget:0})
  const [chat,setChat]=useState([]),[input,setInput]=useState(''),[sending,setSending]=useState(false)
  const [files,setFiles]=useState([]),[uploading,setUploading]=useState(false)
  const [usage,setUsage]=useState({tokens_used:0})
  const request=useCallback(async(path,body,method,lossless=false)=>{
    const r=await fetch(API+path,{method:method || (body?'POST':'GET'),headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined})
    if(!r.ok)throw new Error((await r.text()).trim() || 'Request failed')
    return lossless ? parseSample(await r.text()) : await r.json()
  },[])
  const refresh=useCallback(async()=>{const [r,u]=await Promise.all([request('/api/goals'),request('/api/usage')]);setRuns(r);setUsage(u)},[request])
  useEffect(()=>{refresh().catch(e=>setMessage(e.message));const id=setInterval(()=>refresh().catch(e=>setMessage(e.message)),5000);return ()=>clearInterval(id)},[refresh])
  async function send(e){e.preventDefault();if(!input.trim()||sending||uploading)return;setSending(true)
    try{const text=input.trim();const body={message:text,artifacts:files.map(file=>file.ref)}
      if(/^(?:please\s+)?(?:(?:can|could) you\s+)?start\b/i.test(text)&&source.factory) body.source=runSource(source)
      const reply=await request('/api/chat',body)
      if(reply.factory){setSource({factory:reply.factory,definition:reply.definition,expected:1,records:'[]',budget:0});setPage('Goal builder')}
      setChat(c=>[...c,{role:'user',text,files},{role:'bot',text:reply.reply}]);setInput('');await refresh()
    }catch(err){setMessage(err.message)}finally{setSending(false)}}
  return <div className="app"><header><h1>Agent Factory</h1><span>{usage.tokens_used} tokens · {usage.model_calls || 0} run model calls</span></header>
    <div className="layout"><main><nav aria-label="Workspace navigation">{['Workspace','Goal builder','Decisions','Graph','Rules','Agents','Settings'].map(n=>
      <button key={n} aria-pressed={n===page} onClick={()=>{setPage(n);setMessage('')}}>{n}</button>)}</nav>
    {page==='Settings'?<Settings api={API}/>:['Rules','Agents'].includes(page)?<Definitions api={API} kind={page.toLowerCase()} key={page}/>:
      <Runs {...{page,runs,source,setSource,request,refresh,setMessage}}/>}<p role="status">{message}</p></main>
    <aside className="chat"><h2 className="chat-header">Factory chat</h2><div className="messages">{chat.map((m,i)=><p className={'msg '+m.role} key={i}>{m.text}{m.files?.length>0 && <small className="chat-citations">{m.files.map(f=>f.name+' · r'+f.ref.version).join(', ')}</small>}</p>)}</div>
    <ChatAttachments files={files} setFiles={setFiles} busy={uploading||sending} setBusy={setUploading} request={request} setMessage={setMessage}/>
    <form className="chat-input" onSubmit={send}><input aria-label="Message" placeholder="Create a factory to…, start a run, or check status" value={input} onChange={e=>setInput(e.target.value)}/><button disabled={sending||uploading}>Send</button></form></aside></div></div>
}
