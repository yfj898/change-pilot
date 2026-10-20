<template>
  <ChangeFrame :change="change || undefined">
    <t-alert v-if="error" theme="error" class="notice" :message="error" />
    <t-loading :loading="loading"><template v-if="change">
      <section class="panel"><h2>{{ t('changes.sources') }}</h2><p class="hint">{{ t('changes.sourceHint') }}</p>
        <p v-if="project?.required_reference_paths?.length" class="hint">{{ t('changes.requiredSources') }} <code>{{ project.required_reference_paths.join(', ') }}</code></p>
        <p v-if="!references.length" class="empty">{{ t('changes.noSources') }}</p>
        <div v-for="(source,index) in references" :key="index" class="source-form">
          <div class="two-cols"><label class="field">{{ t('changes.sourcePath') }}<t-input v-model="source.path" :disabled="!canEdit" /></label><label class="field">{{ t('changes.sourceCommit') }}<t-input v-model="source.commit" :disabled="!canEdit" /></label></div>
          <div class="two-cols"><label class="field">{{ t('changes.startLine') }}<t-input-number v-model="source.start_line" :min="1" :disabled="!canEdit" /></label><label class="field">{{ t('changes.endLine') }}<t-input-number v-model="source.end_line" :min="1" :disabled="!canEdit" /></label></div>
          <label class="field">{{ t('changes.knowledgeID') }}<t-input v-model="source.knowledge_id" :disabled="!canEdit" /></label><t-button v-if="canEdit" variant="text" theme="danger" @click="references.splice(index,1)">{{ t('changes.remove') }}</t-button>
        </div>
        <div v-if="canEdit" class="actions"><t-button variant="outline" @click="add">{{ t('changes.addSource') }}</t-button><t-button :loading="saving" @click="save">{{ t('changes.saveSources') }}</t-button></div><p class="hint">{{ t('changes.dirtyEvidence') }}</p>
      </section>
      <section v-for="source in context?.sources || []" :key="`${source.reference.commit}:${source.reference.path}:${source.reference.start_line}`" class="panel"><h2>{{ source.reference.path }}</h2><code>{{ source.reference.commit }}</code><pre>{{ source.content }}</pre></section>
      <ChangeSearch :tenant-id="change.tenant_id" :can-edit="canEdit" @select="linkCandidate" />
      <ChangeAdvice :change="context?.change || change" @updated="load" />
    </template></t-loading>
  </ChangeFrame>
</template>
<script setup lang="ts">
import {computed,onMounted,ref} from 'vue'
import {useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {useAuthStore} from '@/stores/auth'
import {changeError,getChange,getChangeContext,listChangeProjects,updateChange,type ChangeContext,type ChangeProject,type ChangeRecord,type ChangeReference} from '@/api/changes'
import ChangeFrame from './ChangeFrame.vue'
import ChangeAdvice from './ChangeAdvice.vue'
import ChangeSearch from './ChangeSearch.vue'
const {t}=useI18n(),route=useRoute(),auth=useAuthStore(),change=ref<ChangeRecord|null>(null),context=ref<ChangeContext|null>(null),references=ref<ChangeReference[]>([]),project=ref<ChangeProject>(),loading=ref(true),saving=ref(false),error=ref('')
const canEdit=computed(()=>!!change.value&&change.value.stage!=='closed'&&auth.hasRole('contributor')&&(auth.hasRole('admin')||auth.currentUserId===change.value.creator_id))
async function load() {
  loading.value = true
  error.value = ''
  context.value = null
  change.value = null
  try {
    const [record, projects] = await Promise.all([getChange(String(route.params.id)), listChangeProjects()])
    change.value = record.data
    references.value = structuredClone(record.data.input.references || [])
    project.value = projects.data.find(p => p.id === record.data.input.project_id && p.business_pack_id === record.data.input.business_pack_id)
    try {
      context.value = (await getChangeContext(record.data.id)).data
    } catch (e) {
      error.value = `${t('changes.contextUnavailable')} ${changeError(e)}`
    }
  } catch (e) {
    error.value = changeError(e)
  } finally {
    loading.value = false
  }
}
function add(){if(!change.value)return;references.value.push({repository_id:change.value.input.project_id,commit:change.value.input.target_commit,path:''})}
async function save(){if(!change.value)return;saving.value=true;error.value='';try{const c=change.value;const refs=references.value.map(r=>({...r,start_line:r.start_line||undefined,end_line:r.end_line||undefined}));await updateChange(c.id,{expected_revision:c.revision,input:{...c.input,references:refs}});await load()}catch(e){error.value=`${t('changes.conflict')} ${changeError(e)}`}finally{saving.value=false}}
function linkCandidate(id:string){add();const source=references.value[references.value.length-1];if(source)source.knowledge_id=id}
onMounted(load)
</script>
<style scoped>.source-form{padding:20px 0;border-bottom:1px solid var(--td-component-stroke);margin-bottom:20px}.source-form .field{margin-bottom:12px}</style>
