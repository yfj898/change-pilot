import assert from 'node:assert/strict'
import test from 'node:test'
import { reconcileVerificationIntents, rememberVerificationIntent, restoreVerificationIntents } from './verificationIntent'

test('unknown A keeps its UUID and snapshot when current B starts a new run', () => {
  const a = { run_id: 'uuid-a', snapshot_id: 'snapshot-a' }
  const b = { run_id: 'uuid-b', snapshot_id: 'snapshot-b' }
  const pending = rememberVerificationIntent([a], b)
  assert.deepEqual(pending, [a, b])
  assert.deepEqual(restoreVerificationIntents(JSON.stringify(pending), null), [a, b])
  assert.throws(() => rememberVerificationIntent(pending, { run_id: a.run_id, snapshot_id: b.snapshot_id }), /cannot be reassigned/)
})

test('recovering A reuses the original pair and does not add another intent', () => {
  const a = { run_id: 'uuid-a', snapshot_id: 'snapshot-a' }
  const pending = rememberVerificationIntent([a], { ...a })
  assert.deepEqual(pending, [a])
})

test('legacy pending UUID and snapshot are migrated together', () => {
  assert.deepEqual(restoreVerificationIntents('uuid-a', 'snapshot-a'), [{ run_id: 'uuid-a', snapshot_id: 'snapshot-a' }])
  assert.deepEqual(restoreVerificationIntents(null, null), [])
})

test('unknown historical A does not block B and is not guessed to have failed', () => {
  const intents = [{ run_id: 'uuid-a', snapshot_id: 'snapshot-a' }, { run_id: 'uuid-b', snapshot_id: 'snapshot-b' }]
  assert.deepEqual(reconcileVerificationIntents(intents, [
    { link: { run_id: 'uuid-a', submission_state: 'unknown' } },
    { link: { run_id: 'uuid-b', submission_state: 'accepted' }, run: { status: 'pending' } },
  ]), [intents[0]])
})

test('only an actual report or a recorded rejection clears the matching intent', () => {
  const intents = [{ run_id: 'uuid-a', snapshot_id: 'snapshot-a' }, { run_id: 'uuid-b', snapshot_id: 'snapshot-b' }]
  assert.deepEqual(reconcileVerificationIntents(intents, [{ link: { run_id: 'uuid-a', submission_state: 'rejected' } }]), [intents[1]])
  assert.deepEqual(reconcileVerificationIntents(intents, [{ link: { run_id: 'uuid-a', submission_state: 'accepted' } }]), intents)
})

test('incomplete saved intent is reported rather than silently reassigned', () => {
  assert.throws(() => restoreVerificationIntents('[{"run_id":"uuid-a"}]', null), /must contain/)
})
