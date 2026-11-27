import {useState} from 'react'

function fileData(file) {
  return new Promise((resolve,reject)=>{
    const reader=new FileReader()
    reader.onload=()=>resolve(reader.result.split(',')[1])
    reader.onerror=()=>reject(new Error('Unable to read file'))
    reader.readAsDataURL(file)
  })
}

export default function ChatAttachments({files,setFiles,busy,setBusy,request,setMessage}) {
  const [link,setLink]=useState('')
  async function attachLink() {
    const url=link.trim()
    const index=files.findIndex(file=>file.url===url)
    if(index<0 && files.length>=4) {setMessage('Select up to four attachments');return}
    setBusy(true)
    try {
      const {id,version,name,source}=await request('/api/chat/github',{...files[index]?.ref,url})
      const item={name,ref:{id,version},url,source}
      setFiles(index<0?[...files,item]:files.map((old,i)=>i===index?item:old))
      setLink('')
      setMessage('')
    } catch(error) {setMessage(error.message)} finally {setBusy(false)}
  }
  async function upload(selection) {
    const next=[...files]
    const errors=[]
    setBusy(true)
    try {
      for (const file of selection) {
        try {
          const index=next.findIndex(item=>!item.url && item.name===file.name)
          if (index<0 && next.length>=4) throw new Error('Select up to four files')
          if (file.size>512*1024) throw new Error('Maximum file size is 512 KiB')
          const ref=await request('/api/chat/artifacts',{...next[index]?.ref,name:file.name,data:await fileData(file)})
          const item={name:file.name,ref}
          if (index<0) next.push(item)
          else next[index]=item
        } catch (error) {errors.push(file.name+' — '+error.message)}
      }
      setFiles(next)
      setMessage(errors.join(' · '))
    } finally {setBusy(false)}
  }
  return <fieldset disabled={busy} className="chat-attachments"><legend>Goal attachments</legend>
    <label>Attach documents or screenshots<input type="file" multiple accept=".pdf,.docx,.txt,.md,.png,.jpg,.jpeg,.webp"
      onChange={e=>{const selected=Array.from(e.target.files);e.target.value='';upload(selected)}}/></label>
    <label>GitHub link<input type="url" value={link} onChange={e=>setLink(e.target.value)} placeholder="https://github.com/…"/></label>
    <button type="button" disabled={!link.trim()} onClick={attachLink}>Attach GitHub link</button>
    {files.map(({name,ref,source})=><div key={ref.id}>{source?<a href={source} target="_blank" rel="noreferrer">{name}</a>:name} · revision {ref.version}{' '}
      <button type="button" aria-label={'Remove '+name} onClick={()=>setFiles(files.filter(file=>file.ref.id!==ref.id))}>Remove</button></div>)}
  </fieldset>
}
