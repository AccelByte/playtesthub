// Proto enum string constants mirrored here because @accelbyte/codegen
// emits z.any() for every enum, so no generated consts are importable.
// Keep these in lockstep with proto/playtesthub/v1/playtesthub.proto.

export const PlaytestStatus = {
  UNSPECIFIED: 'PLAYTEST_STATUS_UNSPECIFIED',
  DRAFT: 'PLAYTEST_STATUS_DRAFT',
  OPEN: 'PLAYTEST_STATUS_OPEN',
  CLOSED: 'PLAYTEST_STATUS_CLOSED'
} as const
export type PlaytestStatusValue = (typeof PlaytestStatus)[keyof typeof PlaytestStatus]

export const ApplicantStatus = {
  UNSPECIFIED: 'APPLICANT_STATUS_UNSPECIFIED',
  PENDING: 'APPLICANT_STATUS_PENDING',
  APPROVED: 'APPLICANT_STATUS_APPROVED',
  REJECTED: 'APPLICANT_STATUS_REJECTED'
} as const
export type ApplicantStatusValue = (typeof ApplicantStatus)[keyof typeof ApplicantStatus]

export const DistributionModel = {
  UNSPECIFIED: 'DISTRIBUTION_MODEL_UNSPECIFIED',
  STEAM_KEYS: 'DISTRIBUTION_MODEL_STEAM_KEYS',
  AGS_CAMPAIGN: 'DISTRIBUTION_MODEL_AGS_CAMPAIGN',
  ADT: 'DISTRIBUTION_MODEL_ADT'
} as const

export const AnnouncementSendToFilter = {
  ALL: 'ANNOUNCEMENT_SEND_TO_FILTER_ALL',
  APPROVED_ONLY: 'ANNOUNCEMENT_SEND_TO_FILTER_APPROVED_ONLY',
  PENDING_ONLY: 'ANNOUNCEMENT_SEND_TO_FILTER_PENDING_ONLY'
} as const

export const DmStatus = {
  UNSPECIFIED: 'DM_STATUS_UNSPECIFIED',
  SENT: 'DM_STATUS_SENT',
  FAILED: 'DM_STATUS_FAILED'
} as const
