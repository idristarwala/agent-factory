// Preserve numeric source tokens when the browser forwards a sample to the API.
export function parseSample(text) {
  return JSON.parse(text, (_key, value, context) => {
    if (typeof value !== 'number' || (Number.isSafeInteger(value) && (!context?.source || context.source === String(value)))) return value
    if (context?.source && JSON.rawJSON) return JSON.rawJSON(context.source)
    if (!Number.isSafeInteger(value)) throw new Error('This browser cannot preserve the precision of the sample. Use a browser with lossless JSON support.')
    return value
  })
}

// The API returns RFC 6901 pointers into the exact submitted sample.
export function valueAtPointer(sample, pointer) {
  if (pointer === '') return sample
  return pointer.slice(1).split('/').reduce((value, token) => {
    const key = token.replaceAll('~1', '/').replaceAll('~0', '~')
    return value[key]
  }, sample)
}

export function runSource(source) {
  const base = {use_goal_artifacts:source.useGoalArtifacts || undefined, factory: source.factory, selected_rules: source.rules, max_tokens: Number(source.budget)}
  if (source.inputMode === 'mapped') {
    if (!source.approved) throw new Error('Approve a source mapping before starting the run.')
    const {mapping, revision, sample, payload, allowPartial, receipt} = source.approved
    return {...base, source: {mapping, revision, payload: parseSample(payload ?? sample), allow_partial: allowPartial, sample_receipt: payload === undefined || payload === sample ? receipt : undefined}}
  }
  return {...base, records: JSON.parse(source.records)}
}
