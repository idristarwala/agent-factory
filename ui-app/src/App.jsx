import { useState } from 'react'
import './App.css'

const INITIAL = [
  {
    id: 'sku-wrong-pt-1',
    image: 'https://placehold.co/400x400/png?text=Nike+Air+Max',
    before: { title: 'Nike Air Max 270', pt: 'Apparel', weight: '12.5 oz' },
    after: { title: "Nike Men's Air Max 270 Running Shoe", pt: 'Footwear', weight: '12.5 oz' },
    status: 'pending',
    reason: 'Product type corrected from Apparel to Footwear',
  },
  {
    id: 'sku-wrong-weight-1',
    image: 'https://placehold.co/400x400/png?text=Nike+Tee',
    before: { title: 'Nike Dri-FIT Tee', pt: 'Apparel', weight: '40 oz' },
    after: { title: 'Nike Dri-FIT Training Tee', pt: 'Apparel', weight: '6 oz' },
    status: 'pending',
    reason: 'Weight corrected to a plausible value',
  },
  {
    id: 'sku-good-1',
    image: 'https://placehold.co/400x400/png?text=Adidas+Ultraboost',
    before: { title: "Adidas Ultraboost 22 Men's Running Shoe", pt: 'Footwear', weight: '11.2 oz' },
    after: { title: "Adidas Ultraboost 22 Men's Running Shoe, Size 10", pt: 'Footwear', weight: '11.2 oz' },
    status: 'approved',
    reason: 'Title enriched with size',
  },
]

export default function App() {
  const [items, setItems] = useState(INITIAL)
  const [note, setNote] = useState({})

  const pending = items.filter(i => i.status === 'pending').length

  function decide(id, status) {
    setItems(items.map(it => it.id === id ? { ...it, status, feedback: note[id] || '' } : it))
  }

  return (
    <div className="app">
      <header>
        <h1>Catalog Review</h1>
        <div className="stats">{pending} to review · {items.length} total</div>
      </header>
      <main>
        {items.map(it => (
          <article key={it.id} className="card">
            <img src={it.image} alt="" className="thumb" />
            <div className="body">
              <div className="meta">{it.id}</div>
              <h2>{it.after.title}</h2>
              <div className="before">
                {it.before.title}
                {it.before.pt && ` · ${it.before.pt}`}
                {it.before.weight && ` · ${it.before.weight}`}
              </div>
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
                  <input
                    className="note"
                    placeholder="Optional: tell us what to fix…"
                    value={note[it.id] || ''}
                    onChange={e => setNote({ ...note, [it.id]: e.target.value })}
                  />
                </>
              )}
              {it.feedback && <div className="feedback">Note: {it.feedback}</div>}
            </div>
          </article>
        ))}
      </main>
    </div>
  )
}
