<template>
  <section class="panel"><h2>{{ t('changes.searchSources') }}</h2><p class="hint">{{ t('changes.searchHint') }}</p>
    <t-alert v-if="error" theme="error" class="notice" :message="error" />
    <div class="search-controls"><t-select v-model="kbID" :options="knowledgeBases" :placeholder="t('changes.chooseKB')" /><t-input v-model="query" :placeholder="t('changes.searchQuery')" @enter="search" /><t-button :loading="searching" :disabled="!kbID || !query.trim()" @click="search">{{ t('changes.search') }}</t-button></div>
    <p v-if="searched && !results.length" class="hint">{{ t('changes.noSearchResults') }}</p>
    <article v-for="item in results" :key="item.id" class="search-result"><strong>{{ item.knowledge_title || item.knowledge_filename || item.knowledge_id }}</strong><p class="hint">{{ item.content }}</p><t-button v-if="canEdit" variant="outline" size="small" @click="emit('select',item.knowledge_id)">{{ t('changes.linkCandidate') }}</t-button></article>
  </section>
</template>
<script setup lang="ts">
import {onMounted,ref} from 'vue'
import {useI18n} from 'vue-i18n'
import {listKnowledgeBases,knowledgeSemanticSearch} from '@/api/knowledge-base'
import {changeError} from '@/api/changes'
const props=defineProps<{tenantId:number;canEdit:boolean}>(),emit=defineEmits<{select:[id:string]}>(),{t}=useI18n()
interface SearchItem{id:string;knowledge_id:string;knowledge_title?:string;knowledge_filename?:string;content:string}
const knowledgeBases=ref<Array<{label:string;value:string}>>([]),kbID=ref(''),query=ref(''),results=ref<SearchItem[]>([]),error=ref(''),searching=ref(false),searched=ref(false)
onMounted(async()=>{try{const response=await listKnowledgeBases();knowledgeBases.value=(response.data||[]).filter((kb:{tenant_id:number})=>kb.tenant_id===props.tenantId).map((kb:{id:string;name:string})=>({label:kb.name,value:kb.id}));kbID.value=knowledgeBases.value[0]?.value||''}catch(e){error.value=changeError(e)}})
async function search(){if(!kbID.value||!query.value.trim())return;searching.value=true;error.value='';searched.value=false;try{const response=await knowledgeSemanticSearch({query:query.value,knowledge_base_ids:[kbID.value]});results.value=response.data;searched.value=true}catch(e){error.value=changeError(e)}finally{searching.value=false}}
</script>
<style scoped>.search-controls{display:grid;grid-template-columns:240px 1fr auto;gap:12px}.search-result{padding:18px 0;border-bottom:1px solid var(--td-component-stroke)}.search-result p{white-space:pre-wrap}@media(max-width:850px){.search-controls{grid-template-columns:1fr}}</style>
