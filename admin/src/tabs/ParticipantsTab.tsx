import { useAppUIContext } from '@accelbyte/sdk-extend-app-ui'
import { useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Select, Space, Table, Tag, Typography, message } from 'antd'
import dayjs from 'dayjs'
import { useState } from 'react'
import type { V1ParticipantRow } from '../playtesthubapi/generated-definitions/V1ParticipantRow'
import type { V1Playtest } from '../playtesthubapi/generated-definitions/V1Playtest'
import {
  Key_PlaytesthubServiceAdmin,
  usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdApproveMutation,
  usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdRejectMutation,
  usePlaytesthubServiceAdminApi_GetParticipants_ByPlaytestId
} from '../playtesthubapi/generated-admin/queries/PlaytesthubServiceAdmin.query'
import { ApplicantStatus, type ApplicantStatusValue } from '../shared/playtesthub-enums'
import { toastError } from '../shared/api-error'

const STATUS_TAG: Record<string, { text: string; color: string }> = {
  [ApplicantStatus.PENDING]: { text: 'Pending', color: 'blue' },
  [ApplicantStatus.APPROVED]: { text: 'Approved', color: 'green' },
  [ApplicantStatus.REJECTED]: { text: 'Rejected', color: 'red' }
}

export function ParticipantsTab({ playtest }: { playtest: V1Playtest }) {
  const { sdk } = useAppUIContext()
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState<ApplicantStatusValue | ''>('')

  const { data, isLoading, error, refetch } = usePlaytesthubServiceAdminApi_GetParticipants_ByPlaytestId(
    sdk,
    {
      playtestId: playtest.id ?? '',
      queryParams: statusFilter ? { statusFilter } : undefined
    }
  )

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: [Key_PlaytesthubServiceAdmin.Participants_ByPlaytestId] })
  }

  const approveMutation = usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdApproveMutation(sdk, {
    onSuccess: () => {
      message.success('Applicant approved')
      invalidate()
    },
    onError: toastError('approve')
  })
  const rejectMutation = usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdRejectMutation(sdk, {
    onSuccess: () => {
      message.success('Applicant rejected')
      invalidate()
    },
    onError: toastError('reject')
  })

  const rows = (data?.participants ?? []) as V1ParticipantRow[]
  const cap = playtest.autoApproveLimit ?? null
  const enrolled = rows.length

  const columns = [
    { title: 'Discord Handle', dataIndex: 'discordHandle', key: 'discordHandle', render: (v: string) => v || '—' },
    { title: 'AGS User ID', dataIndex: 'userId', key: 'userId', render: (v: string) => v || '—' },
    {
      title: 'Sign-up Date',
      dataIndex: 'signupAt',
      key: 'signupAt',
      render: (v: string | null | undefined) => (v ? dayjs(v).format('YYYY-MM-DD') : '—')
    },
    {
      title: 'NDA Accepted',
      dataIndex: 'ndaAcceptedAt',
      key: 'ndaAcceptedAt',
      render: (v: string | null | undefined) => (v ? '✓' : '—')
    },
    {
      title: 'Code Sent Date',
      dataIndex: 'codeSentAt',
      key: 'codeSentAt',
      render: (v: string | null | undefined) => (v ? dayjs(v).format('YYYY-MM-DD') : '—')
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (v: string) => {
        const tag = STATUS_TAG[v] ?? { text: v ?? '—', color: 'default' }
        return <Tag color={tag.color}>{tag.text}</Tag>
      }
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: unknown, row: V1ParticipantRow) => {
        if (row.status === ApplicantStatus.PENDING) {
          return (
            <Space>
              <Button size="small" type="primary" onClick={() => approveMutation.mutate({ applicantId: row.applicantId ?? '', data: {} })}>
                Approve
              </Button>
              <Button size="small" danger onClick={() => rejectMutation.mutate({ applicantId: row.applicantId ?? '', data: {} })}>
                Reject
              </Button>
            </Space>
          )
        }
        return null
      }
    }
  ]

  if (error) {
    return (
      <Alert
        type="error"
        message="Failed to load participants"
        action={
          <Button size="small" onClick={() => refetch()}>
            Retry
          </Button>
        }
      />
    )
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }} data-testid="participants-tab">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography.Text>
          {enrolled} / {cap ?? '∞'} enrolled
        </Typography.Text>
        <Select
          allowClear
          placeholder="Filter by status"
          style={{ width: 200 }}
          value={statusFilter || undefined}
          onChange={v => setStatusFilter((v ?? '') as ApplicantStatusValue | '')}
          options={[
            { value: ApplicantStatus.PENDING, label: 'Pending' },
            { value: ApplicantStatus.APPROVED, label: 'Approved' },
            { value: ApplicantStatus.REJECTED, label: 'Rejected' }
          ]}
          data-testid="participants-status-filter"
        />
      </div>
      <Table<V1ParticipantRow>
        rowKey={row => row.applicantId ?? ''}
        loading={isLoading}
        dataSource={rows}
        columns={columns}
        pagination={{ pageSize: 25 }}
      />
    </Space>
  )
}
