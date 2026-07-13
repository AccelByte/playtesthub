import { useAppUIContext } from '@accelbyte/sdk-extend-app-ui'
import { InfoCircleFilled } from '@ant-design/icons'
import { useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Card, Checkbox, Form, Input, Popconfirm, Select, Space, Spin, Typography, message } from 'antd'
import { useState } from 'react'
import {
  Key_PlaytesthubServiceAdmin,
  usePlaytesthubServiceAdminApi_CreateSurvey_ByPlaytestIdMutation,
  usePlaytesthubServiceAdminApi_PatchSurvey_ByPlaytestIdMutation
} from '../playtesthubapi/generated-admin/queries/PlaytesthubServiceAdmin.query'
import type { V1MultiChoiceOption } from '../playtesthubapi/generated-definitions/V1MultiChoiceOption'
import type { V1Playtest } from '../playtesthubapi/generated-definitions/V1Playtest'
import type { V1Survey } from '../playtesthubapi/generated-definitions/V1Survey'
import type { V1SurveyQuestion } from '../playtesthubapi/generated-definitions/V1SurveyQuestion'
import { usePlaytesthubServiceApi_GetSurveyPlayer_ByPlaytestId } from '../playtesthubapi/generated-public/queries/PlaytesthubService.query'
import { toastError } from '../shared/api-error'
import { PlaytestStatus } from '../shared/playtesthub-enums'

const QUESTION_TYPE_TEXT = 'SURVEY_QUESTION_TYPE_TEXT'
const QUESTION_TYPE_RATING = 'SURVEY_QUESTION_TYPE_RATING'
const QUESTION_TYPE_MULTI_CHOICE = 'SURVEY_QUESTION_TYPE_MULTI_CHOICE'
const QUESTION_TYPE_LABEL: Record<string, string> = {
  [QUESTION_TYPE_TEXT]: 'Text',
  [QUESTION_TYPE_RATING]: 'Rating (1–5)',
  [QUESTION_TYPE_MULTI_CHOICE]: 'Multi-choice'
}
const MAX_QUESTIONS = 50
const MAX_PROMPT = 1000
const MIN_OPTIONS = 2
const MAX_OPTIONS = 20
const MAX_OPTION_LABEL = 200

type DraftOption = { id?: string; label: string }
type DraftQuestion = {
  key: string
  id?: string
  type: string
  prompt: string
  required: boolean
  allowMultiple: boolean
  options: DraftOption[]
}

let draftKeyCounter = 0
const nextDraftKey = (): string => {
  draftKeyCounter += 1
  return `q-${draftKeyCounter}-${Date.now()}`
}

function questionToDraft(q: V1SurveyQuestion): DraftQuestion {
  return {
    key: nextDraftKey(),
    id: q.id ?? undefined,
    type: typeof q.type === 'string' ? q.type : QUESTION_TYPE_TEXT,
    prompt: q.prompt ?? '',
    required: q.required ?? false,
    allowMultiple: q.allowMultiple ?? false,
    options: (q.options ?? []).map(o => ({ id: o.id ?? undefined, label: o.label ?? '' }))
  }
}

function draftToWire(q: DraftQuestion): V1SurveyQuestion {
  const base: V1SurveyQuestion = {
    type: q.type,
    prompt: q.prompt,
    required: q.required
  }
  if (q.id) base.id = q.id
  if (q.type === QUESTION_TYPE_MULTI_CHOICE) {
    base.allowMultiple = q.allowMultiple
    base.options = q.options.map<V1MultiChoiceOption>(o => (o.id ? { id: o.id, label: o.label } : { label: o.label }))
  }
  return base
}

function freshTextQuestion(): DraftQuestion {
  return { key: nextDraftKey(), type: QUESTION_TYPE_TEXT, prompt: '', required: false, allowMultiple: false, options: [] }
}

function validateDraft(questions: DraftQuestion[]): string | null {
  if (questions.length === 0) return 'Add at least one question'
  if (questions.length > MAX_QUESTIONS) return `At most ${MAX_QUESTIONS} questions`
  for (const [i, q] of questions.entries()) {
    const label = `Question ${i + 1}`
    if (!q.prompt.trim()) return `${label}: prompt is required`
    if (q.prompt.length > MAX_PROMPT) return `${label}: prompt exceeds ${MAX_PROMPT} chars`
    if (q.type === QUESTION_TYPE_MULTI_CHOICE) {
      if (q.options.length < MIN_OPTIONS || q.options.length > MAX_OPTIONS) {
        return `${label}: multi-choice needs ${MIN_OPTIONS}–${MAX_OPTIONS} options`
      }
      for (const [j, opt] of q.options.entries()) {
        if (!opt.label.trim()) return `${label} option ${j + 1}: label is required`
        if (opt.label.length > MAX_OPTION_LABEL) return `${label} option ${j + 1}: label exceeds ${MAX_OPTION_LABEL} chars`
      }
    }
  }
  return null
}

export function SurveyTab({ playtest }: { playtest: V1Playtest }) {
  const { sdk } = useAppUIContext()
  const playtestId = playtest.id ?? ''
  const hasSurvey = Boolean(playtest.surveyId)

  // Player GetSurvey is the authoritative read path (no admin GET in proto).
  // Returns NotFound for DRAFT playtests — render the warning + blank form in
  // that case so first-version edits still work.
  const surveyQuery = usePlaytesthubServiceApi_GetSurveyPlayer_ByPlaytestId(sdk, { playtestId }, { enabled: hasSurvey, retry: false })

  if (hasSurvey && surveyQuery.isLoading) return <Spin description="Loading existing survey..." />

  const initialSurvey = (hasSurvey && surveyQuery.data?.survey ? surveyQuery.data.survey : null) as V1Survey | null
  const draftPreloadFailed = hasSurvey && surveyQuery.isError && playtest.status === PlaytestStatus.DRAFT
  // Mounting a fresh form per data shape avoids the cascading-effect anti-pattern.
  const formKey = `${playtestId}-${initialSurvey?.id ?? 'new'}-${draftPreloadFailed ? 'draft-blank' : 'ok'}`

  return (
    <SurveyTabForm
      key={formKey}
      playtestId={playtestId}
      initialSurvey={initialSurvey}
      hasSurvey={hasSurvey}
      draftPreloadFailed={draftPreloadFailed}
    />
  )
}

type SurveyTabFormProps = {
  playtestId: string
  initialSurvey: V1Survey | null
  hasSurvey: boolean
  draftPreloadFailed: boolean
}

function SurveyTabForm({ playtestId, initialSurvey, hasSurvey, draftPreloadFailed }: SurveyTabFormProps) {
  const { sdk } = useAppUIContext()
  const queryClient = useQueryClient()

  const [questions, setQuestions] = useState<DraftQuestion[]>(() => {
    if (initialSurvey?.questions?.length) return initialSurvey.questions.map(questionToDraft)
    return [freshTextQuestion()]
  })
  const [showNotifyDetail, setShowNotifyDetail] = useState(false)
  const version = initialSurvey?.version ?? null

  const createMutation = usePlaytesthubServiceAdminApi_CreateSurvey_ByPlaytestIdMutation(sdk, {
    onSuccess: () => {
      message.success('Survey created')
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId] })
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Playtests] })
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId] })
    },
    onError: toastError('create survey')
  })
  const editMutation = usePlaytesthubServiceAdminApi_PatchSurvey_ByPlaytestIdMutation(sdk, {
    onSuccess: () => {
      message.success('Survey updated (new version)')
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId] })
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Playtests] })
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId] })
    },
    onError: toastError('update survey')
  })

  const updateQuestion = (key: string, patch: Partial<DraftQuestion>) => {
    setQuestions(prev => prev.map(q => (q.key === key ? { ...q, ...patch } : q)))
  }
  const moveQuestion = (key: string, direction: -1 | 1) => {
    setQuestions(prev => {
      const idx = prev.findIndex(q => q.key === key)
      if (idx < 0) return prev
      const target = idx + direction
      if (target < 0 || target >= prev.length) return prev
      const next = prev.slice()
      const tmp = next[idx]
      next[idx] = next[target]
      next[target] = tmp
      return next
    })
  }
  const removeQuestion = (key: string) => setQuestions(prev => prev.filter(q => q.key !== key))
  const addQuestion = () => setQuestions(prev => [...prev, freshTextQuestion()])
  const setQuestionType = (key: string, type: string) =>
    updateQuestion(key, {
      type,
      options: type === QUESTION_TYPE_MULTI_CHOICE ? [{ label: '' }, { label: '' }] : [],
      allowMultiple: false
    })
  const updateOption = (qKey: string, oIndex: number, label: string) => {
    setQuestions(prev =>
      prev.map(q => {
        if (q.key !== qKey) return q
        const next = q.options.slice()
        next[oIndex] = { ...next[oIndex], label }
        return { ...q, options: next }
      })
    )
  }
  const addOption = (qKey: string) => {
    setQuestions(prev =>
      prev.map(q => (q.key === qKey && q.options.length < MAX_OPTIONS ? { ...q, options: [...q.options, { label: '' }] } : q))
    )
  }
  const removeOption = (qKey: string, oIndex: number) => {
    setQuestions(prev =>
      prev.map(q => (q.key === qKey && q.options.length > MIN_OPTIONS ? { ...q, options: q.options.filter((_, i) => i !== oIndex) } : q))
    )
  }

  const onSave = () => {
    const error = validateDraft(questions)
    if (error) {
      message.error(error)
      return
    }
    const wireQuestions = questions.map(draftToWire)
    if (hasSurvey) {
      editMutation.mutate({ playtestId, data: { questions: wireQuestions } })
      return
    }
    createMutation.mutate({ playtestId, data: { questions: wireQuestions } })
  }

  const saving = createMutation.isPending || editMutation.isPending

  return (
    <>
      <Alert
        type="info"
        showIcon
        icon={<InfoCircleFilled style={{ marginTop: 3 }} />}
        style={{ marginBottom: 16, alignItems: 'flex-start' }}
        data-testid="survey-notify-notice"
        message={
          <div>
            <Typography.Text>When you create this survey, your approved testers are notified automatically.</Typography.Text>
            {!showNotifyDetail && (
              <>
                {' '}
                <Typography.Link onClick={() => setShowNotifyDetail(true)}>Learn more</Typography.Link>
              </>
            )}
            {showNotifyDetail && (
              <>
                <ul style={{ margin: '8px 0 0', paddingInlineStart: 20 }}>
                  <li>
                    <Typography.Text strong>Already approved</Typography.Text> (NDA accepted, if required): a one-time DM with the link.
                  </li>
                  <li>
                    <Typography.Text strong>Approved later</Typography.Text>: the link is added to their approval DM automatically.
                  </li>
                  <li>
                    <Typography.Text strong>Anytime</Typography.Text>: it also appears on the Playtest sign-up page (shown after the player
                    signs up).
                  </li>
                </ul>
                <Typography.Paragraph type="secondary" style={{ margin: '8px 0 0' }}>
                  Editing the survey afterward won't re-send any DMs.
                </Typography.Paragraph>
                <Typography.Link onClick={() => setShowNotifyDetail(false)} style={{ display: 'inline-block', marginTop: 8 }}>
                  Show less
                </Typography.Link>
              </>
            )}
          </div>
        }
      />
      <Card
        data-testid="survey-tab"
        title="Post-Playtest Survey"
        extra={
          <Button type="primary" onClick={onSave} loading={saving} disabled={questions.length === 0}>
            {hasSurvey ? 'Save new version' : 'Create survey'}
          </Button>
        }>
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Typography.Text type="secondary">
            {hasSurvey ? 'Editing existing survey.' : 'Configure the post-playtest survey for approved players.'}
            {version != null && ` Current version v${version} — saving creates v${version + 1}.`}
          </Typography.Text>

          {draftPreloadFailed && (
            <Alert
              type="warning"
              showIcon
              message="DRAFT playtest survey can't be previewed"
              description="Loading existing survey questions requires the playtest to be OPEN. Saving here will create a new version that won't preserve question/option ids — only safe before any responses exist."
            />
          )}

          <Space direction="vertical" size="middle" style={{ display: 'flex' }}>
            {questions.map((q, i) => (
              <div
                key={q.key}
                data-testid="survey-question"
                style={{ background: '#fafafa', border: '1px solid #f0f0f0', borderRadius: 8, padding: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <Typography.Text strong>Question {i + 1}</Typography.Text>
                  <Space size={4}>
                    <Button
                      size="small"
                      onClick={() => moveQuestion(q.key, -1)}
                      disabled={i === 0}
                      aria-label={`Move question ${i + 1} up`}>
                      ↑
                    </Button>
                    <Button
                      size="small"
                      onClick={() => moveQuestion(q.key, 1)}
                      disabled={i === questions.length - 1}
                      aria-label={`Move question ${i + 1} down`}>
                      ↓
                    </Button>
                    <Popconfirm
                      title="Remove this question?"
                      okText="Remove"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => removeQuestion(q.key)}>
                      <Button size="small" danger aria-label={`Remove question ${i + 1}`}>
                        Remove
                      </Button>
                    </Popconfirm>
                  </Space>
                </div>
                <Form layout="vertical">
                  <Form.Item label="Type" style={{ marginBottom: 12 }}>
                    <Select
                      value={q.type}
                      onChange={val => setQuestionType(q.key, val)}
                      options={Object.entries(QUESTION_TYPE_LABEL).map(([value, label]) => ({ value, label }))}
                    />
                  </Form.Item>
                  <Form.Item label="Prompt" style={{ marginBottom: 12 }}>
                    <Input.TextArea
                      value={q.prompt}
                      maxLength={MAX_PROMPT}
                      showCount
                      onChange={e => updateQuestion(q.key, { prompt: e.target.value })}
                      rows={2}
                      placeholder="What did you think of the build?"
                    />
                  </Form.Item>
                  <Form.Item style={{ marginBottom: q.type === QUESTION_TYPE_MULTI_CHOICE ? 12 : 0 }}>
                    <Checkbox checked={q.required} onChange={e => updateQuestion(q.key, { required: e.target.checked })}>
                      Required
                    </Checkbox>
                  </Form.Item>
                  {q.type === QUESTION_TYPE_MULTI_CHOICE && (
                    <>
                      <Form.Item style={{ marginBottom: 12 }}>
                        <Checkbox checked={q.allowMultiple} onChange={e => updateQuestion(q.key, { allowMultiple: e.target.checked })}>
                          Allow multiple selections
                        </Checkbox>
                      </Form.Item>
                      <Form.Item label={`Options (${q.options.length}/${MAX_OPTIONS})`} style={{ marginBottom: 0 }}>
                        <Space direction="vertical" style={{ display: 'flex' }}>
                          {q.options.map((opt, oIdx) => (
                            <Space key={oIdx} style={{ width: '100%' }}>
                              <Input
                                value={opt.label}
                                maxLength={MAX_OPTION_LABEL}
                                onChange={e => updateOption(q.key, oIdx, e.target.value)}
                                placeholder={`Option ${oIdx + 1}`}
                              />
                              <Button onClick={() => removeOption(q.key, oIdx)} disabled={q.options.length <= MIN_OPTIONS}>
                                ×
                              </Button>
                            </Space>
                          ))}
                          <Button onClick={() => addOption(q.key)} disabled={q.options.length >= MAX_OPTIONS}>
                            Add option
                          </Button>
                        </Space>
                      </Form.Item>
                    </>
                  )}
                </Form>
              </div>
            ))}
            <Button onClick={addQuestion} disabled={questions.length >= MAX_QUESTIONS}>
              + Add question
            </Button>
          </Space>
        </Space>
      </Card>
    </>
  )
}
