import { get, getDown, patch, post } from '@/utils/request'

export interface ChangeReference {
  repository_id: string
  commit: string
  path: string
  start_line?: number
  end_line?: number
  knowledge_id?: string
}
export interface ChangeInput {
  objective: string
  project_id: string
  business_pack_id: string
  business_pack_revision: string
  base_commit: string
  target_commit: string
  references: ChangeReference[]
  check_plan: { profile_id: string; profile_revision: string; check_ids: string[] }
}
export interface ChangeRecord {
  id: string
  tenant_id: number
  creator_id: string
  title: string
  assignee_id: string
  notes: string
  input: ChangeInput
  revision: number
  input_revision: number
  /** Advice body is returned only by the authorized context endpoint, not metadata/edit responses. */
  draft?: { content: string; model_id?: string; input_revision: number; author_id: string; updated_at: string }
  stage: 'draft' | 'active' | 'closed'
  created_at: string
  updated_at: string
}
export interface ChangeProject {
  id: string
  name: string
  business_pack_id: string
  business_pack_revision: string
  check_profile_id: string
  check_profile_revision: string
  check_ids: string[]
  required_reference_paths: string[]
}
export interface ChangeSnapshot {
  id: string
  change_id: string
  revision: number
  input_revision: number
  input: ChangeInput
  confirmed_by: string
  created_at: string
}
export interface ChangeContext {
  change: ChangeRecord
  sources: Array<{ reference: ChangeReference; content: string }>
  snapshots: ChangeSnapshot[]
  current_snapshot_id?: string
}
interface Response<T> { success: boolean; data: T }
const changePath = (id: string) => `/api/v1/changes/${encodeURIComponent(id)}`
export const listChanges = (page = 1) => get<Response<ChangeRecord[]> & { total: number }>(`/api/v1/changes?page=${page}&page_size=20`)
export const listChangeProjects = () => get<Response<ChangeProject[]>>('/api/v1/changes/projects')
export const getChange = (id: string) => get<Response<ChangeRecord>>(changePath(id))
export const createChange = (data: Pick<ChangeRecord, 'title' | 'input' | 'notes' | 'assignee_id'>) => post<Response<ChangeRecord>>('/api/v1/changes', data)
export const updateChange = (id: string, data: Partial<Pick<ChangeRecord, 'title' | 'input' | 'notes' | 'assignee_id'>> & { expected_revision: number }) => patch<Response<ChangeRecord>>(changePath(id), data)
export const getChangeContext = (id: string) => get<Response<ChangeContext>>(`${changePath(id)}/context`)
export const confirmChange = (id: string, expected_revision: number) => post<Response<ChangeSnapshot>>(`${changePath(id)}/snapshots`, { expected_revision })
export interface ChangeRunLink { run_id: string; change_id: string; snapshot_id: string; submission_state: 'pending'|'accepted'|'unknown'|'rejected'; submission_error?: string; submitted_by: string; created_at: string }
export interface VerificationCheck { id: string; required: boolean; status: 'passed'|'failed'|'not_run'; exit_code?: number; artifact_id?: string; error?: string }
export interface VerificationAttempt { generation: number; verdict: ''|'passed'|'failed'|'incomplete'; checks: VerificationCheck[]; error?: string; started_at: string; finished_at?: string }
export interface VerificationRun { run_id: string; target_commit: string; status: 'pending'|'running'|'success'|'failed'; generation: number; attempts: VerificationAttempt[]; last_error?: string }
export interface ChangeRunView { link: ChangeRunLink; run?: VerificationRun; applicable: boolean; execution_error?: string }
export interface ChangeHandoff { id: string; run_id: string; snapshot_id: string; conclusion: 'accepted'|'rejected'; notes: string; actor_id: string; created_at: string }
export const listChangeRuns = (id: string) => get<Response<ChangeRunView[]>>(`${changePath(id)}/runs`)
export const submitChangeRun = (id: string, data: { snapshot_id: string; run_id: string; expected_revision: number }) => post<Response<ChangeRunView>>(`${changePath(id)}/runs`, data)
export const listChangeHandoffs = (id: string) => get<Response<ChangeHandoff[]>>(`${changePath(id)}/handoffs`)
export const createChangeHandoff = (id: string, data: { run_id: string; conclusion: 'accepted'|'rejected'; notes: string; expected_revision: number }) => post<Response<ChangeHandoff>>(`${changePath(id)}/handoffs`, data)
export const changeArtifact = (id: string, runID: string, generation: number, artifactID: string) => getDown(`${changePath(id)}/runs/${encodeURIComponent(runID)}/artifacts/${encodeURIComponent(artifactID)}?generation=${generation}`)
export const prepareChangeDraft = (id: string, data: { model_id?: string; content?: string; expected_revision: number }) => post<Response<ChangeRecord>>(`${changePath(id)}/context-draft`, data)

export function changeError(error: unknown): string {
  if (typeof error === 'object' && error !== null) {
    const e = error as { response?: { data?: { error?: { message?: string } } }; message?: string }
    return e.response?.data?.error?.message || e.message || String(error)
  }
  return String(error)
}

export function emptyChangeInput(project: ChangeProject): ChangeInput {
  return {
    objective: '', project_id: project.id, business_pack_id: project.business_pack_id,
    business_pack_revision: project.business_pack_revision, base_commit: '', target_commit: '', references: [],
    check_plan: { profile_id: project.check_profile_id, profile_revision: project.check_profile_revision, check_ids: [...project.check_ids] },
  }
}
