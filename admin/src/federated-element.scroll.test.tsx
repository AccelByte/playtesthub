import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// PM fix #1: the Create button sits far below the required fields, so antd's
// default validation jump-to-top leaves the error off-screen. We pass
// scrollToFirstError to both create + edit Forms. Scroll behavior can't be
// observed in jsdom, so we partially mock antd to capture the props the
// <Form> is constructed with and assert the prop is wired. This mock is
// isolated to this file so the main suite still renders the real antd Form.
const formProps: Array<Record<string, unknown>> = []

vi.mock('antd', async () => {
  const actual = await vi.importActual<typeof import('antd')>('antd')
  const RealForm = actual.Form as unknown as React.ComponentType<Record<string, unknown>> & Record<string, unknown>
  const FormSpy = (props: Record<string, unknown>) => {
    formProps.push(props)
    return <RealForm {...props} />
  }
  // Preserve the statics antd hangs off Form (useForm, useWatch, Item, ...).
  Object.assign(FormSpy, RealForm)
  return { ...actual, Form: FormSpy }
})

vi.mock('@accelbyte/sdk-extend-app-ui', () => ({
  useAppUIContext: () => ({ sdk: {}, isCurrentUserHasPermission: () => true }),
  CrudType: { READ: 'READ', CREATE: 'CREATE', UPDATE: 'UPDATE', DELETE: 'DELETE' }
}))

const noopQuery = { data: undefined, isLoading: false, error: null, refetch: vi.fn() }
const noopMutation = { mutate: vi.fn(), isPending: false, isError: false, error: null }

vi.mock('./playtesthubapi/generated-public/queries/PlaytesthubService.query', () => ({
  usePlaytesthubServiceApi_GetConfig: () => ({ data: { playerBaseUrl: 'https://play.example.com' } })
}))

vi.mock('./playtesthubapi/generated-admin/queries/PlaytesthubServiceAdmin.query', () => ({
  Key_PlaytesthubServiceAdmin: { Playtests: 'playtests', Playtest_ByPlaytestId: 'playtest-by-id', AdtLinkages: 'adt-linkages' },
  usePlaytesthubServiceAdminApi_GetPlaytests: () => ({ data: { playtests: [] }, isLoading: false, error: null, refetch: vi.fn() }),
  usePlaytesthubServiceAdminApi_GetPlaytest_ByPlaytestId: () => ({
    data: {
      playtest: {
        id: 'pt_1',
        slug: 'summer-alpha',
        title: 'Summer Alpha',
        platforms: ['PLATFORM_STEAM'],
        distributionModel: 'DISTRIBUTION_MODEL_STEAM_KEYS',
        ndaRequired: false
      }
    },
    isLoading: false,
    error: null
  }),
  usePlaytesthubServiceAdminApi_CreatePlaytestMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_DeletePlaytest_ByPlaytestIdMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_PatchPlaytest_ByPlaytestIdMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_CreatePlaytest_ByPlaytestIdTransitionStatuMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_GetWorkersHealth: () => ({ data: { workers: [] }, isLoading: false, error: null }),
  usePlaytesthubServiceAdminApi_GetAdtLinkages: () => ({ data: { linkages: [] }, isLoading: false, error: null }),
  usePlaytesthubServiceAdminApi_GetGamesAdt_ByAdtLinkageId: () => noopQuery,
  usePlaytesthubServiceAdminApi_CreateAdtLinkagesStartMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_CreateAdtLinkagesCompleteMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_CreateAdtLinkagesRecoverMutation: () => noopMutation,
  usePlaytesthubServiceAdminApi_DeleteAdtLinkage_ByAdtLinkageIdMutation: () => noopMutation
}))

import { FederatedElement } from './federated-element'

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <FederatedElement />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

beforeEach(() => {
  formProps.length = 0
})

describe('scrollToFirstError (PM fix #1)', () => {
  it('passes scrollToFirstError on the create Form', () => {
    renderAt('/new')
    const withScroll = formProps.find(p => p.scrollToFirstError)
    expect(withScroll?.scrollToFirstError).toEqual({ behavior: 'smooth', block: 'center' })
  })

  it('passes scrollToFirstError on the edit Form', () => {
    renderAt('/pt_1/edit')
    const withScroll = formProps.find(p => p.scrollToFirstError)
    expect(withScroll?.scrollToFirstError).toEqual({ behavior: 'smooth', block: 'center' })
  })
})
