<template>
  <section class="panel"><h2>{{ t('changes.advice') }}</h2><p class="hint">{{ t('changes.adviceHint') }}</p>
    <t-alert v-if="error" theme="error" class="notice" :message="error" />
    <div v-if="canEdit" class="actions"><t-select v-model="modelID" :options="models" :placeholder="t('changes.selectModel')" style="max-width:360px" clearable /><t-button variant="outline" :loading="generating" @click="generate">{{ t('changes.generateAdvice') }}</t-button></div>
    <t-textarea v-model="content" :disabled="!canEdit" :autosize="{minRows:5,maxRows:20}" style="margin-top:18px" />
    <p v-if="change.draft" class="hint">{{ t('changes.inputRevision') }} {{ change.draft.input_revision }} · {{ change.draft.model_id||t('changes.humanEdit') }}</p><div v-if="canEdit" class="actions" style="margin-top:16px"><t-button :disabled="!content.trim()" :loading="saving" @click="save">{{ t('changes.saveAdvice') }}</t-button><t-button variant="outline" theme="danger" :loading="clearing" @click="clear">{{ t('changes.clearAdvice') }}</t-button></div>
  </section>
</template>
<script setup lang="ts">
import {computed,onMounted,ref,watch} from 'vue'
import {useI18n} from 'vue-i18n'
import {useAuthStore} from '@/stores/auth'
import {listModels} from '@/api/model'
import {changeError,prepareChangeDraft,type ChangeRecord} from '@/api/changes'
const props=defineProps<{change:ChangeRecord}>(),emit=defineEmits<{updated:[]}>(),{t}=useI18n(),auth=useAuthStore()
const content=ref(props.change.draft?.content||''),modelID=ref(''),models=ref<Array<{label:string;value:string}>>([]),error=ref(''),saving=ref(false),generating=ref(false),clearing=ref(false)
const canEdit=computed(()=>props.change.stage!=='closed'&&auth.hasRole('contributor')&&(auth.hasRole('admin')||auth.currentUserId===props.change.creator_id))
watch(()=>props.change.draft?.content,value=>{content.value=value||''})
onMounted(async()=>{try{models.value=(await listModels('KnowledgeQA')).filter(m=>m.id).map(m=>({label:m.display_name||m.name,value:m.id!}))}catch(e){error.value=changeError(e)}})
async function generate(){generating.value=true;error.value='';try{await prepareChangeDraft(props.change.id,{expected_revision:props.change.revision,model_id:modelID.value});emit('updated')}catch(e){error.value=changeError(e)}finally{generating.value=false}}
async function save(){saving.value=true;error.value='';try{await prepareChangeDraft(props.change.id,{expected_revision:props.change.revision,content:content.value});emit('updated')}catch(e){error.value=changeError(e)}finally{saving.value=false}}
async function clear(){clearing.value=true;error.value='';try{await prepareChangeDraft(props.change.id,{expected_revision:props.change.revision,content:''});emit('updated')}catch(e){error.value=changeError(e)}finally{clearing.value=false}}
</script>
