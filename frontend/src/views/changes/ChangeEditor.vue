<template>
  <ChangeFrame :change="change">
    <t-alert v-if="error" theme="error" class="notice" :message="error" />
    <t-loading :loading="loading">
      <section v-if="change && form" class="panel">
        <t-alert v-if="!canEdit" theme="info" :message="t('changes.readonly')" class="notice" />
        <label class="field">{{ t('changes.name') }}<t-input v-model="form.title" :disabled="!canEdit" /></label>
        <label class="field">{{ t('changes.objective') }}<t-textarea v-model="form.input.objective" :disabled="!canEdit" :autosize="{ minRows: 3 }" /></label>
        <div class="two-cols"><label class="field">{{ t('changes.baseCommit') }}<t-input v-model="form.input.base_commit" :disabled="!canEdit" /></label><label class="field">{{ t('changes.targetCommit') }}<t-input v-model="form.input.target_commit" :disabled="!canEdit" /></label></div>
        <p class="hint">{{ t('changes.commitHint') }}</p>
        <label class="field">{{ t('changes.assignee') }}<t-input v-model="form.assignee_id" :disabled="!canEdit" /></label>
        <label class="field">{{ t('changes.notes') }}<t-textarea v-model="form.notes" :disabled="!canEdit" /></label>
        <div class="actions"><t-button v-if="canEdit" :loading="saving" @click="save">{{ t('changes.save') }}</t-button><t-button variant="outline" @click="load">{{ t('changes.refresh') }}</t-button><span v-if="saved" class="hint">{{ t('changes.saved') }}</span></div>
      </section>
    </t-loading>
  </ChangeFrame>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { changeError, getChange, updateChange, type ChangeRecord } from '@/api/changes'
import ChangeFrame from './ChangeFrame.vue'
const {t}=useI18n(), route=useRoute(),auth=useAuthStore()
const change=ref<ChangeRecord|null>(null),form=ref<ChangeRecord|null>(null),loading=ref(true),saving=ref(false),saved=ref(false),error=ref('')
const canEdit=computed(()=>!!change.value&&change.value.stage!=='closed'&&auth.hasRole('contributor')&&(auth.hasRole('admin')||auth.currentUserId===change.value.creator_id))
async function load(){loading.value=true;error.value='';saved.value=false;try{change.value=(await getChange(String(route.params.id))).data;form.value=JSON.parse(JSON.stringify(change.value)) as ChangeRecord}catch(e){error.value=changeError(e)}finally{loading.value=false}}
async function save(){if(!change.value||!form.value)return;saving.value=true;error.value='';saved.value=false;try{change.value=(await updateChange(change.value.id,{expected_revision:change.value.revision,title:form.value.title,notes:form.value.notes,assignee_id:form.value.assignee_id,input:form.value.input})).data;form.value=JSON.parse(JSON.stringify(change.value)) as ChangeRecord;saved.value=true}catch(e){error.value=`${t('changes.conflict')} ${changeError(e)}`}finally{saving.value=false}}
onMounted(load)
</script>
