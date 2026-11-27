import { useState, useEffect } from 'react'
import './App.css'
import Definitions from './Definitions'

const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

export default function App() {
  const [page, setPage] = useState("Workspace")
  const [goals, setGoals] = useState([])
  const [usage, setUsage] = useState({ tokens_used: 0, tool_calls: 0, estimated_usd: 0 })
  const [newGoal, setNewGoal] = useState('')
  const [messages, setMessages] = useState([
    { role: 'bot', text: 'Hi. Start a goal or ask about the factory.' },
  ])
  const [input, setInput] = useState('')
  const [chatOpen, setChatOpen] = useState(true)

  useEffect(() => {
    fetch(`${API}/api/goals`).then(r => r.json()).then(setGoals).catch(() => {})
    fetch(`${API}/api/usage`).then(r => r.json()).then(setUsage).catch(() => {})
  }, [])

  function createGoal() {
    if (!newGoal.trim()) return
    fetch(`${API}/api/goals`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ description: newGoal }),
    })
      .then(r => r.json())
      .then(g => {
        setGoals([...goals, g])
        setNewGoal('')
      })
      .catch(() => {})
  }

  function send() {
    const text = input.trim()
    if (!text) return
    fetch(`${API}/api/chat`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: text }),
    })
      .then(r => r.json())
      .then(data => {
        setMessages([...messages, { role: 'user', text }, { role: 'bot', text: data.reply || 'Done.' }])
      })
      .catch(() => {
        setMessages([...messages, { role: 'user', text }, { role: 'bot', text: 'API unavailable.' }])
      })
    setInput('')
  }

  return (
    <div className="app">
      <header>
        <h1>Agent Factory</h1>
        <div className="stats">
          {goals.length} goals · {usage.tokens_used} tokens · ${usage.estimated_usd.toFixed(4)}
        </div>
      </header>
      <div className="layout">
        <main>
          <nav aria-label="Factory navigation">{["Workspace","Rules","Agents"].map(name => <button aria-pressed={page===name} key={name} onClick={()=>setPage(name)}>{name}</button>)}</nav>
          {page !== "Workspace" ? <Definitions api={API} kind={page.toLowerCase()} key={page} /> : <>
          <section className="card">
            <h2>Start a goal</h2>
            <div className="actions">
              <input className="note" placeholder="Describe the outcome you want" value={newGoal} onChange={e => setNewGoal(e.target.value)} />
              <button className="ok" onClick={createGoal}>Start</button>
            </div>
          </section>
          <section>
            <h2>Goals</h2>
            {goals.length === 0 && <p>No goals yet.</p>}
            {goals.map(g => (
              <article key={g.id} className="card">
                <div className="meta">{g.id}</div>
                <h3>{g.description}</h3>
                <span className={`badge ${g.status}`}>{g.status}</span>
              </article>
            ))}
          </section>
        </>}
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
