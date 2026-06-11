import { ArrowLeftOutlined, CopyOutlined } from '@ant-design/icons'
import { useAppUIContext } from '@accelbyte/sdk-extend-app-ui'
import { useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Button,
  Card,
  Modal,
  Space,
  Spin,
  Statistic,
  Tabs,
  Tag,
  Typography,
  message
} from 'antd'
import dayjs from 'dayjs'
import { useNavigate, useParams, useSearchParams } from 'react-router'

import type { V1Playtest } from './playtesthubapi/generated-definitions/V1Playtest'
import {
  Key_PlaytesthubServiceAdmin,
  usePlaytesthubServiceAdminApi_CreatePlaytest_ByPlaytestIdTransitionStatuMutation,
  usePlaytesthubServiceAdminApi_GetParticipants_ByPlaytestId,
  usePlaytesthubServiceAdminApi_GetPlaytests
} from './playtesthubapi/generated-admin/queries/PlaytesthubServiceAdmin.query'
import { usePlaytesthubServiceApi_GetConfig } from './playtesthubapi/generated-public/queries/PlaytesthubService.query'
import { ApplicantStatus, DistributionModel, PlaytestStatus } from './shared/playtesthub-enums'
import { toastError } from './shared/api-error'
import { AuditTab } from './tabs/AuditTab'
import { DiscordBotToolsTab } from './tabs/DiscordBotToolsTab'
import { DistributionTab } from './tabs/DistributionTab'
import { ParticipantsTab } from './tabs/ParticipantsTab'
import { ResponsesTab } from './tabs/ResponsesTab'
import { SurveyTab } from './tabs/SurveyTab'

const STATUS_PILL: Record<string, { text: string; color: string }> = {
  [PlaytestStatus.DRAFT]: { text: 'Draft', color: 'default' },
  [PlaytestStatus.OPEN]: { text: 'Published', color: 'green' },
  [PlaytestStatus.CLOSED]: { text: 'Closed', color: 'red' }
}

const TABS = ['info', 'distribution', 'participants', 'bot-tools', 'survey', 'responses', 'audit'] as const
type TabKey = (typeof TABS)[number]

function isTabKey(v: string): v is TabKey {
  return (TABS as readonly string[]).includes(v)
}

export function PlaytestDetailPage() {
  const { slug } = useParams<{ slug: string }>()
  const { sdk } = useAppUIContext()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const tabParam = searchParams.get('tab') ?? 'info'
  const activeTab: TabKey = isTabKey(tabParam) ? tabParam : 'info'

  const { data, isLoading, error, refetch } = usePlaytesthubServiceAdminApi_GetPlaytests(sdk, {})
  const playtest = ((data?.playtests ?? []) as V1Playtest[]).find(p => p.slug === slug)

  // Player-app origin is owned by the backend (PLAYER_BASE_URL env). Read it
  // via the unauth GetPublicConfig RPC so the admin AppUI never has to guess
  // — window.location.origin would point at the AGS Admin Portal host.
  const publicConfigQuery = usePlaytesthubServiceApi_GetConfig(sdk, {})
  const playerBaseUrl = publicConfigQuery.data?.playerBaseUrl ?? ''

  const transitionMutation = usePlaytesthubServiceAdminApi_CreatePlaytest_ByPlaytestIdTransitionStatuMutation(sdk, {
    onSuccess: () => {
      message.success('Status updated')
      queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Playtests] })
    },
    onError: toastError('update status')
  })

  if (isLoading) return <Spin size="large" data-testid="playtest-detail-loading" />
  if (error || !data) {
    return (
      <Alert
        type="error"
        message="Failed to load playtest"
        action={
          <Button size="small" onClick={() => refetch()}>
            Retry
          </Button>
        }
      />
    )
  }
  if (!playtest) {
    return (
      <Alert
        type="warning"
        message={`Playtest "${slug}" not found`}
        action={
          <Button size="small" onClick={() => navigate('/')}>
            Back to playtests
          </Button>
        }
      />
    )
  }

  const status = playtest.status ?? ''
  const pill = STATUS_PILL[status] ?? { text: status || '—', color: 'default' }
  const isDraft = status === PlaytestStatus.DRAFT
  const isOpen = status === PlaytestStatus.OPEN

  const handleTabChange = (key: string) => {
    const next = new URLSearchParams(searchParams)
    next.set('tab', key)
    setSearchParams(next, { replace: true })
  }

  const publish = () => {
    Modal.confirm({
      title: 'Publish this playtest?',
      content: 'Players can sign up once published.',
      okText: 'Publish',
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          transitionMutation.mutate(
            { playtestId: playtest.id ?? '', data: { targetStatus: PlaytestStatus.OPEN } },
            { onSuccess: () => resolve(), onError: () => reject() }
          )
        })
    })
  }

  const stop = () => {
    Modal.confirm({
      title: 'Stop this playtest?',
      content:
        'Stopping this playtest will close it for new sign-ups. Approved players keep access to their codes / builds.',
      okText: 'Stop Playtest',
      okButtonProps: { danger: true },
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          transitionMutation.mutate(
            { playtestId: playtest.id ?? '', data: { targetStatus: PlaytestStatus.CLOSED } },
            { onSuccess: () => resolve(), onError: () => reject() }
          )
        })
    })
  }

  const copyShareLink = () => {
    if (!playerBaseUrl) {
      message.error('Player app URL not configured — set PLAYER_BASE_URL on the backend')
      return
    }
    const link = `${playerBaseUrl.replace(/\/$/, '')}/#/playtest/${playtest.slug ?? ''}`
    void navigator.clipboard?.writeText(link).then(
      () => message.success('Playtest link copied'),
      () => message.error('Clipboard unavailable')
    )
  }

  return (
    <>
    <div
      style={{ margin: '-16px -16px 0 -16px', backgroundColor: '#fff', padding: '16px 24px 0', borderTop: '1px solid #f0f0f0' }}
      data-testid="playtest-detail-page"
    >
      <Button
        type="link"
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate('/')}
        style={{ padding: 0, height: 'auto', marginBottom: 8 }}
      >
        Back to list
      </Button>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 16, marginBottom: 8 }}>
        <div>
          <Typography.Title level={2} style={{ margin: 0 }}>
            {playtest.title ?? '—'}
          </Typography.Title>
          <Space size={12} style={{ marginTop: 4 }}>
            <Typography.Text type="secondary">{formatDateRange(playtest.startsAt, playtest.endsAt)}</Typography.Text>
            {!isDraft && (
              <Button
                type="link"
                size="small"
                icon={<CopyOutlined />}
                iconPosition="end"
                onClick={copyShareLink}
                style={{ padding: 0, height: 'auto' }}
              >
                Playtest Link
              </Button>
            )}
          </Space>
        </div>
        <Space wrap data-testid="playtest-header-actions">
          <Tag color={pill.color} data-testid="playtest-status-pill" style={{ marginInlineEnd: 0 }}>
            {pill.text}
          </Tag>
          {isDraft && (
            <Button type="primary" onClick={publish} data-testid="header-publish">
              Publish
            </Button>
          )}
          {isOpen && (
            <Button onClick={stop} data-testid="header-stop">
              Stop Playtest
            </Button>
          )}
        </Space>
      </div>

      <Tabs
        activeKey={activeTab}
        onChange={handleTabChange}
        tabBarStyle={{ marginBottom: 0 }}
        items={[
          { key: 'info', label: 'Playtest Info', children: null },
          { key: 'distribution', label: 'Distribution', children: null },
          { key: 'participants', label: 'Participants', children: null },
          { key: 'bot-tools', label: 'Discord Bot Tools', children: null },
          { key: 'survey', label: 'Survey', children: null },
          { key: 'responses', label: 'Responses', children: null },
          { key: 'audit', label: 'Audit', children: null }
        ]}
      />
    </div>

    <div style={{ margin: '0 -16px -16px -16px', backgroundColor: '#f0f2f5', padding: 16, minHeight: 'calc(100vh - 200px)' }}>
      {activeTab === 'info' && <PlaytestInfoTab playtest={playtest} playerBaseUrl={playerBaseUrl} />}
      {activeTab === 'distribution' && <DistributionTab playtest={playtest} />}
      {activeTab === 'participants' && <ParticipantsTab playtest={playtest} />}
      {activeTab === 'bot-tools' && <DiscordBotToolsTab playtest={playtest} />}
      {activeTab === 'survey' && <SurveyTab playtest={playtest} />}
      {activeTab === 'responses' && <ResponsesTab playtest={playtest} />}
      {activeTab === 'audit' && <AuditTab playtest={playtest} />}
    </div>
    </>
  )
}

const DISTRIBUTION_MODEL_LABEL: Record<string, string> = {
  [DistributionModel.ADT]: 'ADT (Direct Download)',
  [DistributionModel.STEAM_KEYS]: 'Steam Keys',
  [DistributionModel.AGS_CAMPAIGN]: 'AGS Campaign'
}

function getPeriodTag(playtest: V1Playtest): { label: string; color: string } {
  const now = dayjs()
  const starts = playtest.startsAt ? dayjs(playtest.startsAt) : null
  const ends   = playtest.endsAt   ? dayjs(playtest.endsAt)   : null
  if (playtest.status === PlaytestStatus.CLOSED) return { label: 'Stopped',  color: 'red' }
  if (playtest.status === PlaytestStatus.DRAFT)  return { label: 'Upcoming', color: 'default' }
  if (starts && now.isBefore(starts)) return { label: 'Upcoming', color: 'blue' }
  if (ends   && now.isAfter(ends))    return { label: 'Ended',    color: 'orange' }
  return { label: 'Running', color: 'green' }
}

function getDaysText(playtest: V1Playtest): string | null {
  const now = dayjs()
  const starts = playtest.startsAt ? dayjs(playtest.startsAt) : null
  const ends   = playtest.endsAt   ? dayjs(playtest.endsAt)   : null

  if (playtest.status === PlaytestStatus.CLOSED) {
    if (!ends) return null
    const n = now.diff(ends, 'day')
    return n === 0 ? 'Ended today' : `Ended ${n} day${n === 1 ? '' : 's'} ago`
  }
  if (playtest.status === PlaytestStatus.DRAFT) {
    if (!starts) return null
    const n = starts.diff(now, 'day')
    if (n <= 0) return 'Starting soon'
    return `Starts in ${n} day${n === 1 ? '' : 's'}`
  }
  // OPEN
  if (ends) {
    const n = ends.diff(now, 'day')
    if (n < 0) {
      const ago = now.diff(ends, 'day')
      return `Ended ${ago} day${ago === 1 ? '' : 's'} ago`
    }
    if (n === 0) return 'Ends today'
    return `${n} day${n === 1 ? '' : 's'} left`
  }
  if (starts && now.isBefore(starts)) {
    const n = starts.diff(now, 'day')
    return n <= 0 ? 'Starting soon' : `Starts in ${n} day${n === 1 ? '' : 's'}`
  }
  return null
}


function OverviewCard({ playtest }: { playtest: V1Playtest }) {
  const { sdk } = useAppUIContext()
  const playtestId = playtest.id ?? ''
  const { data } = usePlaytesthubServiceAdminApi_GetParticipants_ByPlaytestId(sdk, { playtestId }, { retry: false })
  const participants = data?.participants ?? []
  const total = participants.length
  const cap = playtest.autoApproveLimit ?? null
  const isManual = !playtest.autoApprove
  const pending = isManual ? participants.filter(p => p.status === ApplicantStatus.PENDING).length : 0
  const periodTag = getPeriodTag(playtest)
  const daysText = getDaysText(playtest)

  const subsectionStyle: React.CSSProperties = {
    flex: 1,
    background: '#fafafa',
    border: '1px solid #f0f0f0',
    borderRadius: 8,
    padding: 16,
  }

  return (
    <Card title="Overview" data-testid="playtest-overview-card">
      <div style={subsectionStyle}>
        <div style={{ display: 'flex', gap: 32, alignItems: 'flex-start' }}>
          <Statistic
            title="Total Participants"
            value={total}
            suffix={cap != null ? `/ ${cap}` : undefined}
            valueStyle={{ fontSize: 20, fontWeight: 600 }}
          />
          {isManual && (
            <Statistic title="Pending Approval" value={pending} valueStyle={{ fontSize: 20, fontWeight: 600 }} />
          )}
          <div>
            <div style={{ marginBottom: 4, fontSize: 14, color: 'rgba(0,0,0,0.45)' }}>Playtest Period</div>
            <Space size={8} align="center">
              <Typography.Text style={{ fontSize: 20, fontWeight: 600 }}>{formatDateRange(playtest.startsAt, playtest.endsAt)}</Typography.Text>
              <Tag color={periodTag.color}>{periodTag.label}</Tag>
            </Space>
            {daysText && (
              <div style={{ marginTop: 4, fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>{daysText}</div>
            )}
          </div>
        </div>
      </div>
    </Card>
  )
}

function PlaytestInfoTab({ playtest, playerBaseUrl }: { playtest: V1Playtest; playerBaseUrl: string }) {
  const navigate = useNavigate()
  const isDraft = playtest.status === PlaytestStatus.DRAFT

  const distributionLabel = playtest.distributionModel
    ? (DISTRIBUTION_MODEL_LABEL[playtest.distributionModel] ?? playtest.distributionModel)
    : '—'

  const shareLink = playerBaseUrl ? `${playerBaseUrl.replace(/\/$/, '')}/#/playtest/${playtest.slug ?? ''}` : ''

  const copyShareLink = () => {
    if (!shareLink) {
      message.error('Player app URL not configured — set PLAYER_BASE_URL on the backend')
      return
    }
    void navigator.clipboard?.writeText(shareLink).then(
      () => message.success('Playtest link copied'),
      () => message.error('Clipboard unavailable')
    )
  }

  const rows: Array<[string, React.ReactNode]> = [
    ['Title', playtest.title ?? '—'],
    ['Slug', <Typography.Text code>{playtest.slug ?? '—'}</Typography.Text>],
    ['Description', playtest.description ?? '—'],
    ['Start Date', playtest.startsAt ? dayjs(playtest.startsAt).format('MMMM D, YYYY') : '—'],
    ['End Date', playtest.endsAt ? dayjs(playtest.endsAt).format('MMMM D, YYYY') : '—'],
    ['Platforms', (playtest.platforms ?? []).join(', ') || '—'],
    ['NDA Required', playtest.ndaRequired ? 'Yes' : 'No'],
    ['Distribution Model', distributionLabel],
    ['Approval Method', playtest.autoApprove ? 'Auto-Approve' : 'Manual'],
    ['Max Participants', playtest.autoApproveLimit ?? '—'],
    ['Sign-Up Link', isDraft
      ? <Typography.Text type="secondary">Available once published</Typography.Text>
      : shareLink ? (
        <span>
          <span style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace', wordBreak: 'break-all' }}>
            {shareLink}
          </span>
          <Button type="link" size="small" icon={<CopyOutlined />} onClick={copyShareLink} style={{ paddingInline: 0, marginLeft: 8, height: 'auto' }} />
        </span>
      ) : '—'
    ]
  ]

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }} data-testid="playtest-info-tab">
      <OverviewCard playtest={playtest} />
      <Card
        title="Playtest Information"
        extra={
          <Button
            onClick={() =>
              navigate(`/${playtest.id ?? ''}/edit`, {
                state: { from: `/playtest/${playtest.slug ?? ''}` }
              })
            }
            data-testid="playtest-info-edit"
          >
            Edit
          </Button>
        }
        styles={{ body: { padding: 0 } }}
      >
        {rows.map(([label, value], idx) => (
          <div
            key={label}
            style={{
              display: 'flex',
              padding: '14px 24px',
              borderTop: idx === 0 ? 'none' : '1px solid #f0f0f0',
              alignItems: 'center',
              fontSize: 14
            }}
          >
            <div style={{ width: 200, color: 'rgba(0, 0, 0, 0.65)', flexShrink: 0 }}>{label}</div>
            <div style={{ flex: 1, color: 'rgba(0, 0, 0, 0.88)', minWidth: 0 }}>{value}</div>
          </div>
        ))}
      </Card>
    </Space>
  )
}

function formatDateRange(starts?: string | null, ends?: string | null): string {
  if (!starts && !ends) return 'No dates set'
  if (!starts || !ends) {
    const only = starts ? dayjs(starts) : dayjs(ends!)
    return only.format('MMM D, YYYY')
  }
  const s = dayjs(starts)
  const e = dayjs(ends)
  if (s.year() === e.year()) {
    return `${s.format('MMM D')} – ${e.format('MMM D, YYYY')}`
  }
  return `${s.format('MMM D, YYYY')} – ${e.format('MMM D, YYYY')}`
}
