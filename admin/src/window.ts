import dayjs, { type Dayjs } from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import utc from 'dayjs/plugin/utc'

dayjs.extend(utc)
dayjs.extend(relativeTime)

export const DATE_RANGE_LABEL = 'Starts / Ends (UTC)'
export const DATE_RANGE_HELP =
  'Auto-opens at start, auto-closes at end. Leave either side empty to control that boundary manually.'

export const PLAYTEST_STATUS_DRAFT = 'PLAYTEST_STATUS_DRAFT'
export const PLAYTEST_STATUS_OPEN = 'PLAYTEST_STATUS_OPEN'

export function autoTransitionPreview(
  status: string | null | undefined,
  startsAt: string | null | undefined,
  endsAt: string | null | undefined
): string | null {
  if (status === PLAYTEST_STATUS_DRAFT && startsAt) return `Auto-opens ${dayjs.utc(startsAt).fromNow()}`
  if (status === PLAYTEST_STATUS_OPEN && endsAt) return `Auto-closes ${dayjs.utc(endsAt).fromNow()}`
  return null
}

export const dateRangeWindowRule = {
  validator: async (_: unknown, value: [Dayjs | null, Dayjs | null] | undefined) => {
    if (!value || !value[0] || !value[1]) return
    if (!value[1].isAfter(value[0])) throw new Error('ends_at must be after starts_at')
  }
}

export const dateRangeUtcFromEvent = (vals: [Dayjs | null, Dayjs | null] | null) =>
  vals ? (vals.map(v => (v ? v.utc(true) : v)) as [Dayjs | null, Dayjs | null]) : vals
