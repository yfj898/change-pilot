<template>
  <section class="panel"><div class="run-heading"><h2>{{ t('changes.verification') }}</h2><div class="actions"><t-button v-if="canSubmit && context.current_snapshot_id" :loading="submitting" @click="submitNew">{{ t('changes.newRun') }}</t-button><t-button variant="outline" :loading="loading" @click="refresh">{{ t('changes.refresh') }}</t-button></div></div>
    <t-alert v-if="error" theme="error" :message="error" class="notice" />
    <article v-for="intent in unlistedIntents" :key="intent.run_id" class="run-record"><p>{{ t('changes.submission.unknown') }} · <code>{{ intent.run_id }}</code></p><p class="hint">{{ t('changes.submittedSnapshot') }} <code>{{ intent.snapshot_id }}</code></p><t-button v-if="canSubmit" variant="outline" :loading="submitting" @click="submitIntent(intent)">{{ t('changes.recoverRun') }}</t-button></article>
    <p v-if="!runs.length" class="empty">{{ t('changes.noRuns') }}</p>
    <article v-for="view in runs" :key="view.link.run_id" class="run-record">
      <div class="actions"><t-tag :theme="view.run?.status==='success'?'success':view.run?.status==='failed'?'danger':'default'" variant="light">{{ view.run ? t(`changes.runState.${view.run.status}`) : t(`changes.submission.${view.link.submission_state}`) }}</t-tag><t-tag v-if="!view.applicable" theme="warning" variant="light">{{ t('changes.oldReport') }}</t-tag><code>{{ view.link.run_id }}</code></div>
      <t-alert v-if="view.execution_error || view.link.submission_error" theme="warning" class="notice" :message="view.execution_error || view.link.submission_error" />
      <p v-if="view.run" class="hint">{{ t('changes.targetCommit') }} <code>{{ view.run.target_commit }}</code></p>
      <t-button v-if="canSubmit && !view.run && ['unknown','pending'].includes(view.link.submission_state)" variant="outline" :loading="submitting" @click="submitIntent({run_id:view.link.run_id,snapshot_id:view.link.snapshot_id})">{{ t('changes.recoverRun') }}</t-button>
      <section v-for="attempt in view.run?.attempts || []" :key="attempt.generation" class="attempt">
        <h3>{{ t('changes.attempt') }} {{ attempt.generation }} · {{ attempt.verdict ? t(`changes.verdict.${attempt.verdict}`) : t('changes.runState.running') }}</h3><p v-if="attempt.error" class="hint">{{ attempt.error }}</p>
        <div v-for="check in attempt.checks" :key="check.id" class="check-row"><div><strong>{{ check.id }}</strong><span class="hint"> · {{ check.required ? t('changes.mandatory') : t('changes.optional') }}</span><p v-if="check.error" class="check-error">{{ check.error }}</p></div><div class="actions"><t-tag :theme="check.status==='passed'?'success':check.status==='failed'?'danger':'default'" variant="light">{{ t(`changes.checkState.${check.status}`) }}</t-tag><t-button v-if="check.artifact_id" size="small" variant="text" @click="download(view,attempt.generation,check.artifact_id)">{{ t('changes.downloadLog') }}</t-button></div></div>
      </section>
      <div v-if="context.change.stage==='active' && auth.hasRole('contributor') && view.applicable" class="handoff-form">
        <label class="field">{{ t('changes.handoffNotes') }}<t-textarea v-model="notes[view.link.run_id]" :autosize="{minRows:2}" /></label><div class="actions"><t-button :disabled="!passing(view) || !notes[view.link.run_id]?.trim()" :loading="handingOff" @click="handoff(view,'accepted')">{{ t('changes.accept') }}</t-button><t-button variant="outline" :disabled="!notes[view.link.run_id]?.trim()" :loading="handingOff" @click="handoff(view,'rejected')">{{ t('changes.reject') }}</t-button></div>
      </div>
    </article>
    <section v-if="handoffs.length" class="handoff-history"><h2>{{ t('changes.handoffHistory') }}</h2><article v-for="item in handoffs" :key="item.id"><t-tag variant="light">{{ item.conclusion==='accepted'?t('changes.accepted'):t('changes.rejected') }}</t-tag><p>{{ item.notes }}</p><p class="hint">{{ item.actor_id }} · {{ item.created_at }}</p></article></section>
  </section>
</template>
<script setup lang="ts">
import {computed,onMounted,onUnmounted,reactive,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {useAuthStore} from '@/stores/auth'
import {changeArtifact,changeError,createChangeHandoff,listChangeHandoffs,listChangeRuns,submitChangeRun,type ChangeContext,type ChangeHandoff,type ChangeRunView} from '@/api/changes'
import {reconcileVerificationIntents,rememberVerificationIntent,restoreVerificationIntents,type VerificationIntent} from './verificationIntent'
const props=defineProps<{context:ChangeContext}>(),emit=defineEmits<{reload:[]}>(),{t}=useI18n(),auth=useAuthStore()
const runs=ref<ChangeRunView[]>([]),handoffs=ref<ChangeHandoff[]>([]),error=ref(''),loading=ref(false),submitting=ref(false),handingOff=ref(false),notes=reactive<Record<string,string>>({})
const storageKey=`weknora-change-run-intent:${props.context.change.id}`,pendingIntents=ref<VerificationIntent[]>([])
let savedIntentError=''
try{pendingIntents.value=restoreVerificationIntents(sessionStorage.getItem(storageKey),sessionStorage.getItem(storageKey+':snapshot'))}catch(e){savedIntentError=changeError(e);error.value=savedIntentError}
const unlistedIntents=computed(()=>pendingIntents.value.filter(intent=>!runs.value.some(view=>view.link.run_id===intent.run_id)))
const canSubmit=computed(()=>props.context.change.stage!=='closed'&&auth.hasRole('contributor')&&(auth.hasRole('admin')||auth.currentUserId===props.context.change.creator_id))
let timer:ReturnType<typeof setTimeout>|undefined,disposed=false
function stop(){if(timer)clearTimeout(timer);timer=undefined}
function persistIntents() {
  if (pendingIntents.value.length) sessionStorage.setItem(storageKey, JSON.stringify(pendingIntents.value))
  else sessionStorage.removeItem(storageKey)
  sessionStorage.removeItem(storageKey + ':snapshot')
  savedIntentError = ''
}
async function refresh() {
  stop()
  loading.value = true
  error.value = savedIntentError
  try {
    const [a, b] = await Promise.all([listChangeRuns(props.context.change.id), listChangeHandoffs(props.context.change.id)])
    if (disposed) return
    runs.value = a.data
    handoffs.value = b.data
    pendingIntents.value = reconcileVerificationIntents(pendingIntents.value, runs.value)
    if (!savedIntentError) persistIntents()
    if (runs.value.some(view => view.run && ['pending', 'running'].includes(view.run.status))) timer = setTimeout(refresh, 3000)
  } catch (e) {
    error.value = changeError(e)
  } finally {
    loading.value = false
  }
}
async function submitNew() {
  if (!props.context.current_snapshot_id) return
  await submitIntent({ run_id: crypto.randomUUID(), snapshot_id: props.context.current_snapshot_id })
}
async function submitIntent(intent: VerificationIntent) {
  submitting.value = true
  error.value = ''
  try {
    pendingIntents.value = rememberVerificationIntent(pendingIntents.value, intent)
    persistIntents()
    const result = await submitChangeRun(props.context.change.id, { ...intent, expected_revision: props.context.change.revision })
    pendingIntents.value = reconcileVerificationIntents(pendingIntents.value, [result.data])
    persistIntents()
    await refresh()
  } catch (e) {
    error.value = changeError(e)
  } finally {
    submitting.value = false
  }
}
function passing(view:ChangeRunView){const run=view.run;if(!view.applicable||run?.status!=='success')return false;const a=run.attempts.find(a=>a.generation===run.generation);return a?.verdict==='passed'&&props.context.change.input.check_plan.check_ids.every(id=>a.checks.some(c=>c.id===id&&c.required&&c.status==='passed'))}
async function handoff(view:ChangeRunView,conclusion:'accepted'|'rejected'){handingOff.value=true;error.value='';try{await createChangeHandoff(props.context.change.id,{run_id:view.link.run_id,conclusion,notes:notes[view.link.run_id]||'',expected_revision:props.context.change.revision});await refresh();emit('reload')}catch(e){error.value=changeError(e)}finally{handingOff.value=false}}
async function download(view:ChangeRunView,generation:number,artifact:string){try{const data=await changeArtifact(props.context.change.id,view.link.run_id,generation,artifact);const url=URL.createObjectURL(data),a=document.createElement('a');a.href=url;a.download=`${view.link.run_id}-${generation}-${artifact}.log`;a.click();URL.revokeObjectURL(url)}catch(e){error.value=changeError(e)}}
onMounted(refresh);onUnmounted(()=>{disposed=true;stop()})
</script>
<style scoped>.run-heading{display:flex;justify-content:space-between;gap:16px;align-items:center}.run-record{padding:24px 0;border-top:1px solid var(--td-component-stroke)}.attempt{margin:20px 0}.attempt h3{font-size:var(--app-text-lg);font-weight:600}.check-row{display:flex;justify-content:space-between;gap:16px;padding:14px 0;border-bottom:1px solid var(--td-component-stroke)}.check-error{font-size:var(--app-text-md);color:var(--td-error-color);margin:8px 0;overflow-wrap:anywhere}.handoff-form{margin-top:20px}.handoff-history{border-top:1px solid var(--td-component-stroke);padding-top:24px}.handoff-history article{padding:12px 0}@media(max-width:850px){.run-heading,.check-row{align-items:flex-start;flex-direction:column}}</style>
