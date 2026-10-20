<template>
  <main class="change-page">
    <header class="page-header">
      <div><router-link to="/platform/changes" class="back-link">{{ t('changes.list') }}</router-link>
        <h1>{{ change?.title || t('changes.title') }}</h1>
        <p v-if="change" class="meta">{{ change.input.business_pack_id }} · {{ t('changes.revision') }} {{ change.revision }} · {{ t('changes.inputRevision') }} {{ change.input_revision }}
          <t-tag variant="light">{{ t(`changes.${change.stage}`) }}</t-tag>
        </p><p v-else class="meta">{{ t('changes.subtitle') }}</p>
      </div><slot name="actions" />
    </header>
    <nav v-if="change" class="tabs" :aria-label="t('changes.title')">
      <router-link :to="`/platform/changes/${change.id}`" exact-active-class="selected">{{ t('changes.overview') }}</router-link>
      <router-link :to="`/platform/changes/${change.id}/context`" exact-active-class="selected">{{ t('changes.context') }}</router-link>
      <router-link :to="`/platform/changes/${change.id}/verification`" exact-active-class="selected">{{ t('changes.verification') }}</router-link>
    </nav>
    <slot />
  </main>
</template>
<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ChangeRecord } from '@/api/changes'
defineProps<{ change?: ChangeRecord | null }>()
const { t } = useI18n()
</script>
<style>
.change-page{max-width:1240px;margin:0 auto;padding:32px;width:100%;box-sizing:border-box;color:var(--td-text-color-primary);overflow-y:auto;height:100%}
.change-page .page-header{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:24px}.change-page h1{font-size:26px;font-weight:600;margin:10px 0;line-height:1.35}.change-page h2{font-size:18px;font-weight:600;margin:0 0 16px}.change-page .meta,.change-page .hint{color:var(--td-text-color-secondary);font-size:14px;line-height:1.7}.change-page .meta .t-tag{margin-left:8px}.change-page .back-link{font-size:14px;color:var(--td-brand-color)}
.change-page .tabs{display:flex;gap:24px;border-bottom:1px solid var(--td-component-stroke);margin-bottom:24px;overflow-x:auto}.change-page .tabs a{white-space:nowrap;color:var(--td-text-color-secondary);padding:12px 0;text-decoration:none}.change-page .tabs .selected{color:var(--td-brand-color);border-bottom:2px solid var(--td-brand-color);font-weight:600}
.change-page .panel{background:var(--td-bg-color-container);padding:24px;border-radius:8px;margin:18px 0;border:1px solid var(--td-component-stroke)}.change-page .field{display:flex;flex-direction:column;gap:8px;margin-bottom:20px;font-size:14px;font-weight:500}.change-page .field .hint{margin:0;font-weight:400}.change-page .two-cols{display:grid;grid-template-columns:1fr 1fr;gap:20px}.change-page .actions{display:flex;align-items:center;gap:12px;flex-wrap:wrap}.change-page .notice{margin:16px 0}.change-page code{font-size:13px;overflow-wrap:anywhere}.change-page pre{white-space:pre-wrap;overflow-wrap:anywhere;background:var(--td-bg-color-secondarycontainer);border-radius:6px;padding:16px;line-height:1.7;font-size:13px}.change-page .empty{padding:48px 16px;text-align:center;color:var(--td-text-color-secondary)}
@media(max-width:850px){.change-page{padding:20px}.change-page .two-cols{grid-template-columns:1fr;gap:0}.change-page .panel{padding:18px}.change-page .page-header{align-items:flex-start;flex-direction:column}}
</style>
