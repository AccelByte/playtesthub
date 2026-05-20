import { useAppUIContext } from '@accelbyte/sdk-extend-app-ui'
import { useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Button,
  Card,
  InputNumber,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
  Upload,
  message
} from 'antd'
import dayjs from 'dayjs'
import { useState } from 'react'
import type { V1Code } from '../playtesthubapi/generated-definitions/V1Code'
import type { V1CodePoolStats } from '../playtesthubapi/generated-definitions/V1CodePoolStats'
import type { V1Playtest } from '../playtesthubapi/generated-definitions/V1Playtest'
import type { V1UploadCodesRejection } from '../playtesthubapi/generated-definitions/V1UploadCodesRejection'
import {
  Key_PlaytesthubServiceAdmin,
  usePlaytesthubServiceAdminApi_CreateCodesSyncFromAg_ByPlaytestIdMutation,
  usePlaytesthubServiceAdminApi_CreateCodesTopUp_ByPlaytestIdMutation,
  usePlaytesthubServiceAdminApi_CreateCodesUpload_ByPlaytestIdMutation,
  usePlaytesthubServiceAdminApi_GetCodes_ByPlaytestId
} from '../playtesthubapi/generated-admin/queries/PlaytesthubServiceAdmin.query'
import { toastError } from '../shared/api-error'
import { DistributionModel } from '../shared/playtesthub-enums'

const POOL_LOW_RATIO = 0.1

const CODE_STATE_TAG: Record<string, { text: string; color: string }> = {
  CODE_STATE_UNUSED: { text: 'Unused', color: 'default' },
  CODE_STATE_RESERVED: { text: 'Reserved', color: 'gold' },
  CODE_STATE_GRANTED: { text: 'Granted', color: 'green' }
}

export function DistributionTab({ playtest }: { playtest: V1Playtest }) {
  const model = playtest.distributionModel ?? ''
  switch (model) {
    case DistributionModel.ADT:
      return <ADTPanel playtest={playtest} />
    case DistributionModel.STEAM_KEYS:
      return <SteamKeysPanel playtest={playtest} />
    case DistributionModel.AGS_CAMPAIGN:
      return <AGSCampaignPanel playtest={playtest} />
    default:
      return (
        <Alert
          type="info"
          showIcon
          message={`Distribution model: ${model || 'unspecified'}`}
          description="No distribution-specific UI is available for this model."
          data-testid="distribution-tab"
        />
      )
  }
}

function ADTPanel({ playtest }: { playtest: V1Playtest }) {
  const linked = Boolean(playtest.adtNamespace)
  return (
    <Space direction="vertical" style={{ width: '100%' }} data-testid="distribution-tab">
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        ADT distribution
      </Typography.Title>
      {!linked ? (
        <Card>
          <Space direction="vertical">
            <Typography.Text strong>🔗 ADT Namespace Not Linked</Typography.Text>
            <Typography.Text type="secondary">
              Link your studio's ADT namespace to surface builds and approve players against this playtest.
            </Typography.Text>
            <Typography.Text type="secondary">
              Linking happens from the Playtests list page → Link new ADT Namespace.
            </Typography.Text>
          </Space>
        </Card>
      ) : (
        <Card>
          <Space direction="vertical" size="small">
            <Tag color="blue">ADT linkage is studio-wide</Tag>
            <FieldRow label="ADT Namespace" value={playtest.adtNamespace ?? '—'} />
            <FieldRow label="Game ID" value={playtest.adtGameId ?? '—'} />
            <FieldRow label="Build ID" value={playtest.adtBuildId ?? '—'} />
            <FieldRow label="Fallback URL" value={playtest.adtFallbackDownloadUrl ?? '(none)'} />
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              ADT linkage is studio-wide; the same ADT namespace covers every playtest under this studio.
            </Typography.Text>
          </Space>
        </Card>
      )}
    </Space>
  )
}

function SteamKeysPanel({ playtest }: { playtest: V1Playtest }) {
  const { sdk } = useAppUIContext()
  const queryClient = useQueryClient()
  const playtestId = playtest.id ?? ''

  const codesQuery = usePlaytesthubServiceAdminApi_GetCodes_ByPlaytestId(sdk, { playtestId }, { enabled: !!playtestId })
  const invalidateCodes = () => queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Codes_ByPlaytestId] })

  const [csvText, setCsvText] = useState('')
  const [csvFilename, setCsvFilename] = useState('')
  const [rejections, setRejections] = useState<V1UploadCodesRejection[]>([])

  const uploadMutation = usePlaytesthubServiceAdminApi_CreateCodesUpload_ByPlaytestIdMutation(sdk, {
    onSuccess: response => {
      const r = (response.rejections ?? []) as V1UploadCodesRejection[]
      setRejections(r)
      if (r.length === 0) {
        message.success(`Inserted ${response.inserted ?? 0} codes`)
        setCsvText('')
        setCsvFilename('')
      } else {
        message.warning(`Upload rejected: ${r.length} invalid line${r.length === 1 ? '' : 's'}`)
      }
      invalidateCodes()
    },
    onError: toastError('upload codes')
  })

  const handleFileChosen = (file: File) => {
    const reader = new FileReader()
    reader.onload = () => {
      setCsvText(typeof reader.result === 'string' ? reader.result : '')
      setCsvFilename(file.name ?? '')
      setRejections([])
    }
    reader.readAsText(file)
    return false
  }

  const stats = codesQuery.data?.stats
  const codes = (codesQuery.data?.codes ?? []) as V1Code[]
  const total = stats?.total ?? 0

  return (
    <Space direction="vertical" style={{ width: '100%' }} data-testid="distribution-tab">
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        Steam keys
      </Typography.Title>

      <LowPoolBanner stats={stats} />
      <PoolStatsRow stats={stats} />

      <div>
        <Typography.Title level={5}>Upload Steam keys</Typography.Title>
        <Typography.Paragraph type="secondary">
          One code per line. UTF-8, max 10 MB, max 50,000 lines, charset <code>[A-Za-z0-9._-]</code>, length 1–128. Any
          invalid line rejects the whole file.
        </Typography.Paragraph>
        <Upload accept=".csv,.txt,text/plain,text/csv" beforeUpload={handleFileChosen} maxCount={1} showUploadList={false}>
          <Button>Choose file</Button>
        </Upload>
        {csvFilename && (
          <Typography.Paragraph style={{ marginTop: 8 }}>
            Selected: <code>{csvFilename}</code>
          </Typography.Paragraph>
        )}
        <Button
          type="primary"
          disabled={!csvText}
          loading={uploadMutation.isPending}
          style={{ marginTop: 8 }}
          onClick={() => uploadMutation.mutate({ playtestId, data: { csvContent: csvText, filename: csvFilename || undefined } })}>
          Upload
        </Button>
        {rejections.length > 0 && (
          <Alert
            type="error"
            style={{ marginTop: 12 }}
            message={`Upload rejected — ${rejections.length} invalid line${rejections.length === 1 ? '' : 's'}`}
            description={
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {rejections.slice(0, 50).map((rej, i) => (
                  <li key={i}>
                    Line {rej.lineNumber}: {rej.reason}
                    {rej.value ? ` — ${rej.value}` : ''}
                  </li>
                ))}
                {rejections.length > 50 && <li>…and {rejections.length - 50} more.</li>}
              </ul>
            }
          />
        )}
      </div>

      {total === 0 && !codesQuery.isLoading && (
        <Alert type="info" showIcon message="No codes uploaded yet" description="Upload a CSV above to start approving applicants." />
      )}

      <CodesTable query={codesQuery} codes={codes} />
    </Space>
  )
}

function AGSCampaignPanel({ playtest }: { playtest: V1Playtest }) {
  const { sdk } = useAppUIContext()
  const queryClient = useQueryClient()
  const playtestId = playtest.id ?? ''

  const codesQuery = usePlaytesthubServiceAdminApi_GetCodes_ByPlaytestId(sdk, { playtestId }, { enabled: !!playtestId })
  const invalidateCodes = () => queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Codes_ByPlaytestId] })

  const [topUpQty, setTopUpQty] = useState<number | null>(100)

  const topUpMutation = usePlaytesthubServiceAdminApi_CreateCodesTopUp_ByPlaytestIdMutation(sdk, {
    onSuccess: response => {
      message.success(`Generated ${response.added ?? 0} new codes`)
      invalidateCodes()
    },
    onError: toastError('top up')
  })

  const syncMutation = usePlaytesthubServiceAdminApi_CreateCodesSyncFromAg_ByPlaytestIdMutation(sdk, {
    onSuccess: response => {
      message.success(`Synced ${response.added ?? 0} new codes from AGS`)
      invalidateCodes()
    },
    onError: toastError('sync from AGS')
  })

  const stats = codesQuery.data?.stats
  const codes = (codesQuery.data?.codes ?? []) as V1Code[]

  return (
    <Space direction="vertical" style={{ width: '100%' }} data-testid="distribution-tab">
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        AGS Campaign codes
      </Typography.Title>

      <Card>
        <Space direction="vertical" size="small">
          <FieldRow label="AGS Item ID" value={playtest.agsItemId ?? '—'} />
          <FieldRow label="AGS Campaign ID" value={playtest.agsCampaignId ?? '—'} />
          <FieldRow label="Initial Quantity" value={playtest.initialCodeQuantity ?? '—'} />
        </Space>
      </Card>

      <LowPoolBanner stats={stats} />
      <PoolStatsRow stats={stats} />

      <div>
        <Typography.Title level={5}>Generate / sync AGS Campaign codes</Typography.Title>
        <Typography.Paragraph type="secondary">
          Top-up calls AGS to generate fresh codes. Sync re-fetches from AGS to recover from a previous failure (idempotent).
        </Typography.Paragraph>
        <Space>
          <InputNumber min={1} max={50000} value={topUpQty} onChange={v => setTopUpQty(typeof v === 'number' ? v : null)} />
          <Button
            type="primary"
            disabled={!topUpQty || topUpQty < 1}
            loading={topUpMutation.isPending}
            onClick={() => topUpQty && topUpMutation.mutate({ playtestId, data: { quantity: topUpQty } })}>
            Generate more codes
          </Button>
          <Button loading={syncMutation.isPending} onClick={() => syncMutation.mutate({ playtestId, data: {} })}>
            Sync from AGS
          </Button>
        </Space>
      </div>

      <CodesTable query={codesQuery} codes={codes} />
    </Space>
  )
}

function PoolStatsRow({ stats }: { stats: V1CodePoolStats | null | undefined }) {
  return (
    <Space wrap size="large">
      <Statistic title="Total" value={stats?.total ?? 0} />
      <Statistic title="Unused" value={stats?.unused ?? 0} />
      <Statistic title="Reserved" value={stats?.reserved ?? 0} />
      <Statistic title="Granted" value={stats?.granted ?? 0} />
    </Space>
  )
}

function LowPoolBanner({ stats }: { stats: V1CodePoolStats | null | undefined }) {
  const total = stats?.total ?? 0
  const unused = stats?.unused ?? 0
  if (total <= 0) return null
  if (unused / total > POOL_LOW_RATIO) return null
  return (
    <Alert
      type="warning"
      showIcon
      message="Code pool is low"
      description={`Only ${unused} of ${total} codes remain unused. Top up before approving more applicants.`}
    />
  )
}

function CodesTable({
  query,
  codes
}: {
  query: { isLoading: boolean; error: unknown; refetch: () => void }
  codes: V1Code[]
}) {
  const columns = [
    { title: 'Value', dataIndex: 'value', key: 'value', render: (v: string | null | undefined) => v ?? '—' },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      render: (v: string | null | undefined) => {
        const info = CODE_STATE_TAG[v ?? ''] ?? { text: v ?? '—', color: 'default' }
        return <Tag color={info.color}>{info.text}</Tag>
      }
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (v: string | null | undefined) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '—')
    }
  ]
  return (
    <div>
      <Typography.Title level={5}>Codes</Typography.Title>
      {query.isLoading && <Spin />}
      {!query.isLoading && !!query.error && (
        <Alert
          type="error"
          message="Failed to load codes."
          action={
            <Button size="small" onClick={() => query.refetch()}>
              Retry
            </Button>
          }
        />
      )}
      {!query.isLoading && !query.error && (
        <Table<V1Code> rowKey={row => row.id ?? ''} dataSource={codes} columns={columns} pagination={{ pageSize: 50 }} />
      )}
    </div>
  )
}

function FieldRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', gap: 16, alignItems: 'baseline' }}>
      <Typography.Text strong style={{ width: 160 }}>
        {label}
      </Typography.Text>
      <Typography.Text>{value}</Typography.Text>
    </div>
  )
}
