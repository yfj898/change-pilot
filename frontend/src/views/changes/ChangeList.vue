<template>
  <ChangeFrame>
    <template #actions><t-button v-if="auth.hasRole('contributor')" :disabled="!projects.length" @click="creating = true">{{ t('changes.create') }}</t-button></template>
    <t-alert v-if="error" theme="error" class="notice" :message="error" />
    <t-loading :loading="loading">
      <div v-if="!loading && !projects.length" class="panel hint">{{ t('changes.noProjects') }}</div>
      <section v-if="creating && selected" class="panel">
        <h2>{{ t('changes.create') }}</h2>
        <label class="field">{{ t('changes.businessPack') }}<t-select v-model="packKey" :options="projectOptions" @change="selectProject" /></label>
        <label class="field">{{ t('changes.name') }}<t-input v-model="title" /></label>
        <label class="field">{{ t('changes.objective') }}<t-textarea v-model="objective" :autosize="{ minRows: 3 }" /></label>
        <div class="actions"><t-button :loading="saving" :disabled="!title.trim() || !objective.trim()" @click="create">{{ t('changes.create') }}</t-button><t-button variant="outline" @click="creating=false">{{ t('changes.cancel') }}</t-button></div>
      </section>
      <div v-if="!records.length && !loading" class="empty">{{ t('changes.empty') }}</div>
      <section v-for="change in records" :key="change.id" class="change-row panel">
        <router-link :to="`/platform/changes/${change.id}`"><h2>{{ change.title }}</h2></router-link>
        <p class="hint">{{ change.input.objective }}</p><div class="actions"><t-tag variant="light">{{ t(`changes.${change.stage}`) }}</t-tag><code>{{ change.input.business_pack_id }}</code><span class="hint">{{ t('changes.revision') }} {{ change.revision }}</span></div>
      </section>
      <t-pagination v-if="total > 20" :current="page" :total="total" :page-size="20" :show-page-size="false" @current-change="changePage" />
    </t-loading>
  </ChangeFrame>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { changeError, createChange, emptyChangeInput, listChangeProjects, listChanges, type ChangeProject, type ChangeRecord } from '@/api/changes'
import ChangeFrame from './ChangeFrame.vue'
const { t } = useI18n(), router = useRouter(), auth = useAuthStore()
const records = ref<ChangeRecord[]>([]), projects = ref<ChangeProject[]>([]), loading = ref(true), saving = ref(false), error = ref(''), creating = ref(false)
const page = ref(1), total = ref(0), title = ref(''), objective = ref(''), packKey = ref('')
const key = (p: ChangeProject) => `${p.id}/${p.business_pack_id}`
const selected = computed(() => projects.value.find(p => key(p) === packKey.value))
const projectOptions = computed(() => projects.value.map(p => ({ label: `${p.name || p.id} · ${p.business_pack_id} ${p.business_pack_revision}`, value: key(p) })))
const selectProject = () => { error.value = '' }
async function load() { loading.value = true; error.value=''; try { const [a,b] = await Promise.all([listChanges(page.value), listChangeProjects()]); records.value=a.data;total.value=a.total;projects.value=b.data;if(!packKey.value)packKey.value=b.data[0]?key(b.data[0]):'' } catch(e) { error.value=changeError(e) } finally { loading.value=false } }
async function create() { if(!selected.value)return;saving.value=true;error.value='';try { const input=emptyChangeInput(selected.value);input.objective=objective.value;const result=await createChange({title:title.value,input,assignee_id:'',notes:''});await router.push(`/platform/changes/${result.data.id}`) } catch(e){error.value=changeError(e)}finally{saving.value=false} }
function changePage(value:number){page.value=value;void load()}
onMounted(load)
</script>
