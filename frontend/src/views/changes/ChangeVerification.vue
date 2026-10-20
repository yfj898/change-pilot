<template>
  <ChangeFrame :change="context?.change"><t-alert v-if="error" theme="error" class="notice" :message="error" />
    <t-loading :loading="loading"><template v-if="context">
      <section class="panel"><h2>{{ t('changes.checkPlan') }}</h2><p><code>{{ context.change.input.check_plan.profile_id }} / {{ context.change.input.check_plan.profile_revision }}</code></p><div class="actions"><t-tag v-for="id in context.change.input.check_plan.check_ids" :key="id" variant="light">{{ id }}</t-tag></div><p class="hint">{{ t('changes.confirmHint') }}</p>
        <div v-if="context.current_snapshot_id"><strong>{{ t('changes.currentSnapshot') }}</strong><p><code>{{ context.current_snapshot_id }}</code></p></div><t-alert v-else theme="warning" :message="t('changes.noSnapshot')" class="notice" />
        <div class="actions"><t-button v-if="canEdit && !context.current_snapshot_id" :loading="confirming" @click="confirm">{{ t('changes.confirm') }}</t-button><t-button variant="outline" @click="load">{{ t('changes.refresh') }}</t-button></div>
      </section>
      <VerificationRuns :context="context" @reload="load" />
      <section v-if="context.snapshots.length" class="panel"><h2>{{ t('changes.snapshotHistory') }}</h2><article v-for="snapshot in context.snapshots" :key="snapshot.id" class="snapshot"><t-tag :theme="snapshot.id===context.current_snapshot_id?'primary':'default'" variant="light">{{ snapshot.id===context.current_snapshot_id?t('changes.currentSnapshot'):t('changes.oldSnapshot') }}</t-tag><p><code>{{ snapshot.id }}</code></p><p class="hint">{{ t('changes.inputRevision') }} {{ snapshot.input_revision }} · {{ t('changes.confirmedBy') }} {{ snapshot.confirmed_by }}</p><code>{{ snapshot.input.target_commit }}</code></article></section>
    </template></t-loading>
  </ChangeFrame>
</template>
<script setup lang="ts">
import {computed,onMounted,ref} from 'vue'
import {useRoute} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {useAuthStore} from '@/stores/auth'
import {changeError,confirmChange,getChangeContext,type ChangeContext} from '@/api/changes'
import ChangeFrame from './ChangeFrame.vue'
import VerificationRuns from './VerificationRuns.vue'
const {t}=useI18n(),route=useRoute(),auth=useAuthStore(),context=ref<ChangeContext|null>(null),loading=ref(true),confirming=ref(false),error=ref('')
const canEdit=computed(()=>context.value?.change.stage!=='closed'&&auth.hasRole('contributor')&&(auth.hasRole('admin')||auth.currentUserId===context.value?.change.creator_id))
async function load(){loading.value=true;error.value='';try{context.value=(await getChangeContext(String(route.params.id))).data}catch(e){error.value=changeError(e)}finally{loading.value=false}}
async function confirm(){if(!context.value)return;confirming.value=true;error.value='';try{await confirmChange(context.value.change.id,context.value.change.revision);await load()}catch(e){error.value=changeError(e)}finally{confirming.value=false}}
onMounted(load)
</script>
<style scoped>.snapshot{padding:18px 0;border-bottom:1px solid var(--td-component-stroke)}.snapshot:last-child{border:0}</style>
