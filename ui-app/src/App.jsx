import { useState } from 'react'
import './App.css'

const INITIAL = [
  {
    id: 'sku-wrong-pt-1',
    image: 'https://placehold.co/400x400/e8e8e8/333?text=Nike+Air+Max',
    before: { title: 'Nike Air Max 270', pt: 'Apparel', weight: '12.5 oz' },
    after: { title: "Nike Men's Air Max 270 Running Shoe", pt: 'Footwear', weight: '12.5 oz' },
    status: 'pending',
    reason: 'Product type corrected from Apparel to Footwear',
  },
  {
    id: 'sku-wrong-weight-1',
    image: 'https://placehold.co/400x400/e8e8e8/333?text=Nike+Tee',
    before: { title: 'Nike Dri-FIT Tee', pt: 'Apparel', weight: '40 oz' },
    after: { title: 'Nike Dri-FIT Training Tee', pt: 'Apparel', weight: '6 oz' },
    status: 'pending',
    reason: 'Weight corrected to a plausible value',
  },
  {
    id: 'sku-good-1',
    image: 'https://placehold.co/400x400/e8e8e8/333?text=Adidas',
    before: { title: "Adidas Ultraboost 22 Men's Running Shoe", pt: 'Footwear', weight: '11.2 oz' },
    after: { title: "Adidas Ultraboost 22 Men's Running Shoe, Size 10", pt: 'Footwear', weight: '11.2 oz' },
    status: 'approved',
    reason: 'Title enriched with size',
  },
]

function localReply(text, items) {
  const t = text.toLowerCase()
  if (t.includes('pending') || t.includes('review')) {
    return `There are ${items.filter(i => i.status === 'pending').length} items waiting for review.`
  }
  if (t.includes('approved')) {
    return `${items.filter(i => i.status === 'approved').length} items approved.`
  }
  if (t.includes('goal') || t.includes('fix') || t.includes('start')) {
    return `Got it. Starting work on: "${text}". Decisions will appear as they are ready.`
  }
  return `Ask about pending items, or give a goal like "Fix product types and weights".`
}

export default function App() {
  const [items, setItems] = useState(INITIAL)
  const [note, setNote] = useState({})
  const [messages, setMessages] = useState([
    { role: 'bot', text: 'Hi. I can answer questions about these decisions or start work if you give me a goal.' },
  ])
  const [input, setInput] = useState('')
  const [chatOpen, setChatOpen] = useState(true)

  const pending = items.filter(i => i.status === 'pending').length

  function decide(id, status) {
    setItems(items.map(it => it.id === id ? { ...it, status, feedback: note[id] || '' } : it))
  }

  function send() {
    const text = input.trim()
    if (!text) return
    const reply = localReply(text, items)
    setMessages([...messages, { role: 'user', text }, { role: 'bot', text: reply }])
    setInput('')
  }

  return (
    <div className="app">
      <header>
        <h1>Catalog Review</h1>
        <div className="stats">{pending} to review · {items.length} total</div>
      </header>
      <div className="layout">
        <main>
          {items.map(it => (
            <article key={it.id} className="card">
              <img src={it.image} alt="" className="thumb" />
              <div>
                <div className="meta">{it.id}</div>
                <h2>{it.after.title}</h2>
                <div className="before">{it.before.title} · {it.before.pt} · {it.before.weight}</div>
                <div className="row">
                  <span className={`badge ${it.status}`}>{it.status}</span>
                  <span className="reason">{it.reason}</span>
                </div>
                {it.status === 'pending' && (
                  <>
                    <div className="actions">
                      <button className="ok" onClick={() => decide(it.id, 'approved')}>Looks good</button>
                      <button className="no" onClick={() => decide(it.id, 'rejected')}>Needs change</button>
                    </div>
                    <input className="note" placeholder="Optional note…" value={note[it.id] || ''} onChange={e => setNote({ ...note, [it.id]: e.target.value })} />
                  </>
                )}
                {it.feedback && <div className="feedback">Note: {it.feedback}</div>}
              </div>
            </article>
          ))}
        </main>
        {chatOpen && (
          <aside className="chat">
            <div className="chat-header">Assistant <button className="close" onClick={() => setChatOpen(false)}>×</button></div>
            <div className="messages">
              {messages.map((m, i) => <div key={i} className={`msg ${m.role}`}>{m.text}</div>)}
            </div>
            <div className="chat-input">
              <input value={input} onChange={e => setInput(e.target.value)} onKeyDown={e => e.key === 'Enter' && send()} placeholder="Ask or give a goal…" />
              <button onClick={send}>Send</button>
            </div>
          </aside>
        )}
      </div>
      {!chatOpen && <button className="fab" onClick={() => setChatOpen(true)}>Chat</button>}
    </div>
  )
}
