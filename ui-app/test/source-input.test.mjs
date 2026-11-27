import test from 'node:test'
import assert from 'node:assert/strict'
import {parseSample,runSource} from '../src/sourceInput.js'

test('mapped run forwards exact source numbers and the approved revision',()=>{
  const sample='{"key":9007199254740993,"amount":1.123456789012345678}'
  assert.equal(JSON.stringify(parseSample(sample)),sample)
  const input=runSource({factory:'telemetry',inputMode:'mapped',approved:{mapping:'devices',revision:2,sample}})
  assert.equal(JSON.stringify(input.source.payload),sample)
  assert.equal(input.source.revision,2)
  assert.throws(()=>runSource({inputMode:'mapped'}),/Approve/)
})
