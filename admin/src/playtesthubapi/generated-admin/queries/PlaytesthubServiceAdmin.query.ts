/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
/**
 * AUTO GENERATED
 */
import type { AccelByteSDK, ApiError, SdkSetConfigParam } from '@accelbyte/sdk'
import type { UseMutationOptions, UseMutationResult, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'
import { useMutation, useQuery } from '@tanstack/react-query'
import type { AxiosError, AxiosResponse } from 'axios'
import { PlaytesthubServiceAdminApi } from '../PlaytesthubServiceAdminApi.js'

import { PlaytesthubServiceApproveApplicantBody } from '../../generated-definitions/PlaytesthubServiceApproveApplicantBody.js'
import { PlaytesthubServiceCreatePlaytestBody } from '../../generated-definitions/PlaytesthubServiceCreatePlaytestBody.js'
import { PlaytesthubServiceCreateSurveyBody } from '../../generated-definitions/PlaytesthubServiceCreateSurveyBody.js'
import { PlaytesthubServiceEditPlaytestBody } from '../../generated-definitions/PlaytesthubServiceEditPlaytestBody.js'
import { PlaytesthubServiceEditSurveyBody } from '../../generated-definitions/PlaytesthubServiceEditSurveyBody.js'
import { PlaytesthubServiceRejectApplicantBody } from '../../generated-definitions/PlaytesthubServiceRejectApplicantBody.js'
import { PlaytesthubServiceRetryDmBody } from '../../generated-definitions/PlaytesthubServiceRetryDmBody.js'
import { PlaytesthubServiceRetryFailedDmsBody } from '../../generated-definitions/PlaytesthubServiceRetryFailedDmsBody.js'
import { PlaytesthubServiceSyncFromAgsBody } from '../../generated-definitions/PlaytesthubServiceSyncFromAgsBody.js'
import { PlaytesthubServiceTopUpCodesBody } from '../../generated-definitions/PlaytesthubServiceTopUpCodesBody.js'
import { PlaytesthubServiceTransitionPlaytestStatusBody } from '../../generated-definitions/PlaytesthubServiceTransitionPlaytestStatusBody.js'
import { PlaytesthubServiceUploadCodesBody } from '../../generated-definitions/PlaytesthubServiceUploadCodesBody.js'
import { V1AdminGetPlaytestResponse } from '../../generated-definitions/V1AdminGetPlaytestResponse.js'
import { V1ApproveApplicantResponse } from '../../generated-definitions/V1ApproveApplicantResponse.js'
import { V1CreatePlaytestResponse } from '../../generated-definitions/V1CreatePlaytestResponse.js'
import { V1CreateSurveyResponse } from '../../generated-definitions/V1CreateSurveyResponse.js'
import { V1EditPlaytestResponse } from '../../generated-definitions/V1EditPlaytestResponse.js'
import { V1EditSurveyResponse } from '../../generated-definitions/V1EditSurveyResponse.js'
import { V1GetCodePoolResponse } from '../../generated-definitions/V1GetCodePoolResponse.js'
import { V1ListApplicantsResponse } from '../../generated-definitions/V1ListApplicantsResponse.js'
import { V1ListAuditLogResponse } from '../../generated-definitions/V1ListAuditLogResponse.js'
import { V1ListPlaytestsResponse } from '../../generated-definitions/V1ListPlaytestsResponse.js'
import { V1ListSurveyResponsesResponse } from '../../generated-definitions/V1ListSurveyResponsesResponse.js'
import { V1RejectApplicantResponse } from '../../generated-definitions/V1RejectApplicantResponse.js'
import { V1RetryDmResponse } from '../../generated-definitions/V1RetryDmResponse.js'
import { V1RetryFailedDmsResponse } from '../../generated-definitions/V1RetryFailedDmsResponse.js'
import { V1SoftDeletePlaytestResponse } from '../../generated-definitions/V1SoftDeletePlaytestResponse.js'
import { V1SyncFromAgsResponse } from '../../generated-definitions/V1SyncFromAgsResponse.js'
import { V1TopUpCodesResponse } from '../../generated-definitions/V1TopUpCodesResponse.js'
import { V1TransitionPlaytestStatusResponse } from '../../generated-definitions/V1TransitionPlaytestStatusResponse.js'
import { V1UploadCodesResponse } from '../../generated-definitions/V1UploadCodesResponse.js'

export const Key_PlaytesthubServiceAdmin = {
  Playtests: 'Playtesthubapi.PlaytesthubServiceAdmin.Playtests',
  Playtest: 'Playtesthubapi.PlaytesthubServiceAdmin.Playtest',
  Playtest_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.Playtest_ByPlaytestId',
  Codes_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.Codes_ByPlaytestId',
  Survey_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.Survey_ByPlaytestId',
  Applicant_ByApplicantIdReject: 'Playtesthubapi.PlaytesthubServiceAdmin.Applicant_ByApplicantIdReject',
  AuditLog_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.AuditLog_ByPlaytestId',
  Applicant_ByApplicantIdApprove: 'Playtesthubapi.PlaytesthubServiceAdmin.Applicant_ByApplicantIdApprove',
  Applicant_ByApplicantIdRetryDm: 'Playtesthubapi.PlaytesthubServiceAdmin.Applicant_ByApplicantIdRetryDm',
  Applicants_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.Applicants_ByPlaytestId',
  CodesTopUp_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.CodesTopUp_ByPlaytestId',
  CodesUpload_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.CodesUpload_ByPlaytestId',
  Playtest_ByPlaytestIdTransitionStatu: 'Playtesthubapi.PlaytesthubServiceAdmin.Playtest_ByPlaytestIdTransitionStatu',
  SurveyResponses_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.SurveyResponses_ByPlaytestId',
  CodesSyncFromAg_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.CodesSyncFromAg_ByPlaytestId',
  ApplicantsRetryFailedDm_ByPlaytestId: 'Playtesthubapi.PlaytesthubServiceAdmin.ApplicantsRetryFailedDm_ByPlaytestId'
} as const

export const usePlaytesthubServiceAdminApi_GetPlaytests = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam,
  options?: Omit<UseQueryOptions<V1ListPlaytestsResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1ListPlaytestsResponse>) => void
): UseQueryResult<V1ListPlaytestsResponse, AxiosError<ApiError>> => {
  const queryFn = (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetPlaytests>[1]) => async () => {
    const response = await PlaytesthubServiceAdminApi(sdk, { coreConfig: input.coreConfig, axiosConfig: input.axiosConfig }).getPlaytests()
    callback?.(response)
    return response.data
  }

  return useQuery<V1ListPlaytestsResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.Playtests, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

/**
 * STEAM_KEYS only in M1; distribution_model=AGS_CAMPAIGN returns Unimplemented until M2.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Playtest, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreatePlaytestMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<V1CreatePlaytestResponse, AxiosError<ApiError>, SdkSetConfigParam & { data: PlaytesthubServiceCreatePlaytestBody }>,
    'mutationKey'
  >,
  callback?: (data: V1CreatePlaytestResponse) => void
): UseMutationResult<
  V1CreatePlaytestResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { data: PlaytesthubServiceCreatePlaytestBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { data: PlaytesthubServiceCreatePlaytestBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, { coreConfig: input.coreConfig, axiosConfig: input.axiosConfig }).createPlaytest(
      input.data
    )
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Playtest],
    mutationFn,
    ...options
  })
}

export const usePlaytesthubServiceAdminApi_GetPlaytest_ByPlaytestId = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam & { playtestId: string },
  options?: Omit<UseQueryOptions<V1AdminGetPlaytestResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1AdminGetPlaytestResponse>) => void
): UseQueryResult<V1AdminGetPlaytestResponse, AxiosError<ApiError>> => {
  const queryFn = (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetPlaytest_ByPlaytestId>[1]) => async () => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).getPlaytest_ByPlaytestId(input.playtestId)
    callback?.(response)
    return response.data
  }

  return useQuery<V1AdminGetPlaytestResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

export const usePlaytesthubServiceAdminApi_DeletePlaytest_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<V1SoftDeletePlaytestResponse, AxiosError<ApiError>, SdkSetConfigParam & { playtestId: string }>,
    'mutationKey'
  >,
  callback?: (data: V1SoftDeletePlaytestResponse) => void
): UseMutationResult<V1SoftDeletePlaytestResponse, AxiosError<ApiError>, SdkSetConfigParam & { playtestId: string }> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).deletePlaytest_ByPlaytestId(input.playtestId)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * Editable: title, description, bannerImageUrl, platforms, startsAt, endsAt, ndaRequired, ndaText. Immutable fields → InvalidArgument.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_PatchPlaytest_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1EditPlaytestResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditPlaytestBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1EditPlaytestResponse) => void
): UseMutationResult<
  V1EditPlaytestResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditPlaytestBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditPlaytestBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).patchPlaytest_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * Returns aggregate counts plus the full code list including raw values — admin surfaces are exempt from the §6 log-redaction rule (PRD §5.7).
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Codes_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_GetCodes_ByPlaytestId = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam & { playtestId: string },
  options?: Omit<UseQueryOptions<V1GetCodePoolResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1GetCodePoolResponse>) => void
): UseQueryResult<V1GetCodePoolResponse, AxiosError<ApiError>> => {
  const queryFn = (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetCodes_ByPlaytestId>[1]) => async () => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).getCodes_ByPlaytestId(input.playtestId)
    callback?.(response)
    return response.data
  }

  return useQuery<V1GetCodePoolResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.Codes_ByPlaytestId, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

/**
 * Natural-key on playtest_id. Server mints question UUIDs and multi-choice option UUIDs. Bounds: ≤50 questions, prompt ≤1,000 chars, multi-choice 2–20 options with label ≤200 chars (schema.md §"Survey entity spec").
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateSurvey_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1CreateSurveyResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceCreateSurveyBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1CreateSurveyResponse) => void
): UseMutationResult<
  V1CreateSurveyResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceCreateSurveyBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceCreateSurveyBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createSurvey_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * Always creates a new Survey row with version = previous + 1. Question UUIDs are preserved for kept questions (client passes the existing id) and minted for new ones (id empty). Multi-choice option ids likewise — keeps histogram aggregation keys stable across edits per schema.md.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_PatchSurvey_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1EditSurveyResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditSurveyBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1EditSurveyResponse) => void
): UseMutationResult<
  V1EditSurveyResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditSurveyBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceEditSurveyBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).patchSurvey_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Survey_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * Re-reject returns the existing row (natural-key idempotency). rejection_reason is admin-visible (max 500 chars per schema.md).
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdReject, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdRejectMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1RejectApplicantResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRejectApplicantBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1RejectApplicantResponse) => void
): UseMutationResult<
  V1RejectApplicantResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRejectApplicantBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRejectApplicantBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createApplicant_ByApplicantIdReject(input.applicantId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdReject],
    mutationFn,
    ...options
  })
}

/**
 * actor_filter='system' maps to actorUserId IS NULL per PRD §4.7. action_filter is exact-match on the action string. before_json / after_json carry the JSONB columns verbatim — the client renders the diff.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.AuditLog_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_GetAuditLog_ByPlaytestId = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam & {
    playtestId: string
    queryParams?: { actorFilter?: string | null; actionFilter?: string | null; pageToken?: string | null; pageSize?: number }
  },
  options?: Omit<UseQueryOptions<V1ListAuditLogResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1ListAuditLogResponse>) => void
): UseQueryResult<V1ListAuditLogResponse, AxiosError<ApiError>> => {
  const queryFn = (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetAuditLog_ByPlaytestId>[1]) => async () => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).getAuditLog_ByPlaytestId(input.playtestId, input.queryParams)
    callback?.(response)
    return response.data
  }

  return useQuery<V1ListAuditLogResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.AuditLog_ByPlaytestId, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

/**
 * Re-approve on an already-APPROVED applicant returns the existing row (natural-key idempotency). Errors per docs/errors.md ApproveApplicant rows.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdApprove, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdApproveMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1ApproveApplicantResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceApproveApplicantBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1ApproveApplicantResponse) => void
): UseMutationResult<
  V1ApproveApplicantResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceApproveApplicantBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceApproveApplicantBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createApplicant_ByApplicantIdApprove(input.applicantId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdApprove],
    mutationFn,
    ...options
  })
}

/**
 * No cooldown — double-click sends two DMs (PRD §5.4). Returns the updated Applicant row with refreshed DM fields.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdRetryDm, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateApplicant_ByApplicantIdRetryDmMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1RetryDmResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRetryDmBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1RetryDmResponse) => void
): UseMutationResult<
  V1RetryDmResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRetryDmBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { applicantId: string; data: PlaytesthubServiceRetryDmBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createApplicant_ByApplicantIdRetryDm(input.applicantId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Applicant_ByApplicantIdRetryDm],
    mutationFn,
    ...options
  })
}

/**
 * Order: createdAt DESC. Filters: status_filter (UNSPECIFIED → no filter), dm_failed_filter (true → only lastDmStatus='failed'). page_token is opaque; absent → start of stream. page_size 0 → server default (50).
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.Applicants_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_GetApplicants_ByPlaytestId = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam & {
    playtestId: string
    queryParams?: {
      statusFilter?: 'APPLICANT_STATUS_UNSPECIFIED' | 'APPLICANT_STATUS_PENDING' | 'APPLICANT_STATUS_APPROVED' | 'APPLICANT_STATUS_REJECTED'
      dmFailedFilter?: boolean | null
      pageToken?: string | null
      pageSize?: number
    }
  },
  options?: Omit<UseQueryOptions<V1ListApplicantsResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1ListApplicantsResponse>) => void
): UseQueryResult<V1ListApplicantsResponse, AxiosError<ApiError>> => {
  const queryFn =
    (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetApplicants_ByPlaytestId>[1]) => async () => {
      const response = await PlaytesthubServiceAdminApi(sdk, {
        coreConfig: input.coreConfig,
        axiosConfig: input.axiosConfig
      }).getApplicants_ByPlaytestId(input.playtestId, input.queryParams)
      callback?.(response)
      return response.data
    }

  return useQuery<V1ListApplicantsResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.Applicants_ByPlaytestId, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

/**
 * Each call generates a fresh batch via the AGS Campaign API. Per docs/ags-failure-modes.md the call is not transactional; partial fulfillment commits the codes received. STEAM_KEYS playtests reject with FailedPrecondition.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.CodesTopUp_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateCodesTopUp_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1TopUpCodesResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTopUpCodesBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1TopUpCodesResponse) => void
): UseMutationResult<
  V1TopUpCodesResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTopUpCodesBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTopUpCodesBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createCodesTopUp_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.CodesTopUp_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * PRD §4.3: UTF-8, charset [A-Za-z0-9._-], 1–128 chars/code, file ≤10 MB, ≤50,000 codes, file-level + cross-row dedup. On any violation the response carries per-line rejection details and 0 codes are inserted.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.CodesUpload_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateCodesUpload_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1UploadCodesResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceUploadCodesBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1UploadCodesResponse) => void
): UseMutationResult<
  V1UploadCodesResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceUploadCodesBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceUploadCodesBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createCodesUpload_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.CodesUpload_ByPlaytestId],
    mutationFn,
    ...options
  })
}

export const usePlaytesthubServiceAdminApi_CreatePlaytest_ByPlaytestIdTransitionStatuMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1TransitionPlaytestStatusResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTransitionPlaytestStatusBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1TransitionPlaytestStatusResponse) => void
): UseMutationResult<
  V1TransitionPlaytestStatusResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTransitionPlaytestStatusBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceTransitionPlaytestStatusBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createPlaytest_ByPlaytestIdTransitionStatu(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.Playtest_ByPlaytestIdTransitionStatu],
    mutationFn,
    ...options
  })
}

/**
 * Default page_size 50, max 200. Optional survey_id_filter narrows to a single Survey version for per-version aggregate split.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.SurveyResponses_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_GetSurveyResponses_ByPlaytestId = (
  sdk: AccelByteSDK,
  input: SdkSetConfigParam & {
    playtestId: string
    queryParams?: { surveyIdFilter?: string | null; pageToken?: string | null; pageSize?: number }
  },
  options?: Omit<UseQueryOptions<V1ListSurveyResponsesResponse, AxiosError<ApiError>>, 'queryKey'>,
  callback?: (data: AxiosResponse<V1ListSurveyResponsesResponse>) => void
): UseQueryResult<V1ListSurveyResponsesResponse, AxiosError<ApiError>> => {
  const queryFn =
    (sdk: AccelByteSDK, input: Parameters<typeof usePlaytesthubServiceAdminApi_GetSurveyResponses_ByPlaytestId>[1]) => async () => {
      const response = await PlaytesthubServiceAdminApi(sdk, {
        coreConfig: input.coreConfig,
        axiosConfig: input.axiosConfig
      }).getSurveyResponses_ByPlaytestId(input.playtestId, input.queryParams)
      callback?.(response)
      return response.data
    }

  return useQuery<V1ListSurveyResponsesResponse, AxiosError<ApiError>>({
    queryKey: [Key_PlaytesthubServiceAdmin.SurveyResponses_ByPlaytestId, input],
    queryFn: queryFn(sdk, input),
    ...options
  })
}

/**
 * Fetch-only recovery for the case where AGS holds codes our DB never persisted. STEAM_KEYS playtests reject with FailedPrecondition.
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.CodesSyncFromAg_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateCodesSyncFromAg_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1SyncFromAgsResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceSyncFromAgsBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1SyncFromAgsResponse) => void
): UseMutationResult<
  V1SyncFromAgsResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceSyncFromAgsBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceSyncFromAgsBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createCodesSyncFromAg_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.CodesSyncFromAg_ByPlaytestId],
    mutationFn,
    ...options
  })
}

/**
 * Walks every applicant with last_dm_status=FAILED for the playtest and enqueues each through the same DM-queue path as approve, respecting the 10k cap and configured drain rate. Overflowed rows stay FAILED with last_dm_error='dm_queue_overflow' (PRD §5.5).
 *
 * #### Default Query Options
 * The default options include:
 * ```
 * {
 *    queryKey: [Key_PlaytesthubServiceAdmin.ApplicantsRetryFailedDm_ByPlaytestId, input]
 * }
 * ```
 */
export const usePlaytesthubServiceAdminApi_CreateApplicantsRetryFailedDm_ByPlaytestIdMutation = (
  sdk: AccelByteSDK,
  options?: Omit<
    UseMutationOptions<
      V1RetryFailedDmsResponse,
      AxiosError<ApiError>,
      SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceRetryFailedDmsBody }
    >,
    'mutationKey'
  >,
  callback?: (data: V1RetryFailedDmsResponse) => void
): UseMutationResult<
  V1RetryFailedDmsResponse,
  AxiosError<ApiError>,
  SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceRetryFailedDmsBody }
> => {
  const mutationFn = async (input: SdkSetConfigParam & { playtestId: string; data: PlaytesthubServiceRetryFailedDmsBody }) => {
    const response = await PlaytesthubServiceAdminApi(sdk, {
      coreConfig: input.coreConfig,
      axiosConfig: input.axiosConfig
    }).createApplicantsRetryFailedDm_ByPlaytestId(input.playtestId, input.data)
    callback?.(response.data)
    return response.data
  }

  return useMutation({
    mutationKey: [Key_PlaytesthubServiceAdmin.ApplicantsRetryFailedDm_ByPlaytestId],
    mutationFn,
    ...options
  })
}
