import test from 'node:test'
import assert from 'node:assert/strict'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { createServer } from 'vite'

test('decision review displays typed values and actions only for pending changes', async () => {
  const vite = await createServer({ server: { middlewareMode: true, hmr: false }, appType: 'custom' })
  try {
    const { default: Runs } = await vite.ssrLoadModule('/src/Runs.jsx')
    const decision = { id: 'd1', entity_id: 'request-1', status: 'pending', before: { priority: 0 }, after: { priority: 2 } }
    const run = { id: 'r1', status: 'in_review', decisions: [decision] }
    const render = () => renderToStaticMarkup(createElement(Runs, { runs: [run], page: 'Decisions' }))
    assert.match(render(), /<td>priority<\/td><td>0<\/td><td>2<\/td>/)
    assert.match(render(), />Approve<\/button>/)
    decision.status = 'approved'
    decision.feedback = 'Use the source priority'
    decision.proposal = { Evidence: ['Source register'], AgentName: 'triage', AgentRevision: 2 }
    assert.match(render(), /Source register/)
    assert.match(render(), /triage · r2/)
    assert.match(render(), /Use the source priority/)
    assert.doesNotMatch(render(), />Approve<\/button>/)
  } finally {
    await vite.close()
  }
})
