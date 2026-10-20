export interface VerificationIntent {
  run_id: string
  snapshot_id: string
}

/** Restore the prior two-key format without assigning its UUID to a newer snapshot. */
export function restoreVerificationIntents(stored: string | null, legacySnapshot: string | null): VerificationIntent[] {
  if (!stored) return []
  if (legacySnapshot) return [{ run_id: stored, snapshot_id: legacySnapshot }]
  const value: unknown = JSON.parse(stored)
  if (!Array.isArray(value) || value.some(item => !item || typeof item.run_id !== 'string' || !item.run_id || typeof item.snapshot_id !== 'string' || !item.snapshot_id)) {
    throw new Error('Saved verification intent must contain run_id and snapshot_id pairs')
  }
  return value.map(item => ({ run_id: item.run_id, snapshot_id: item.snapshot_id }))
}

export function rememberVerificationIntent(intents: VerificationIntent[], intent: VerificationIntent): VerificationIntent[] {
  const previous = intents.find(item => item.run_id === intent.run_id)
  if (previous && previous.snapshot_id !== intent.snapshot_id) {
    throw new Error('A verification run UUID cannot be reassigned to another snapshot')
  }
  return previous ? intents : [...intents, { ...intent }]
}

/** Only authoritative acceptance/rejection resolves an uncertain local intent. */
export function reconcileVerificationIntents(
  intents: VerificationIntent[],
  runs: Array<{ link: { run_id: string; submission_state: string }; run?: unknown }>,
): VerificationIntent[] {
  return intents.filter(intent => !runs.some(view => view.link.run_id === intent.run_id && (view.run || view.link.submission_state === 'rejected')))
}
