function fileData(file) {
  return new Promise((resolve,reject)=>{
    const reader=new FileReader()
    reader.onload=()=>resolve(reader.result.split(',')[1])
    reader.onerror=()=>reject(new Error('Unable to read file'))
    reader.readAsDataURL(file)
  })
}

export default function ChatAttachments({files,setFiles,busy,setBusy,request,setMessage}) {
  async function upload(selection) {
    const next=[...files]
    const errors=[]
    setBusy(true)
    try {
      for (const file of selection) {
        try {
          const index=next.findIndex(item=>item.name===file.name)
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
    {files.map(({name,ref})=><div key={ref.id}>{name} · revision {ref.version}{' '}
      <button type="button" aria-label={'Remove '+name} onClick={()=>setFiles(files.filter(file=>file.ref.id!==ref.id))}>Remove</button></div>)}
  </fieldset>
}
