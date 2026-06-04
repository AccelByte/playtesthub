import { message } from 'antd'

// ApiError matches what the codegen axios client surfaces on a failed
// gateway request: the raw grpc-gateway body lives on `response.data` as
// `{ code, message, details }` (numeric gRPC code + human message), and the
// HTTP status on `response.status`. Older code assumed a `data.errorMessage`
// field that the gateway never emits — both names are read here so the helpers
// stay correct regardless of which surface produced the error.
export type ApiError = {
  response?: {
    status?: number
    data?: {
      code?: number
      message?: string
      errorMessage?: string
    }
  }
}

export function apiErrorCode(err: ApiError): number | undefined {
  return err?.response?.data?.code
}

export function apiErrorStatus(err: ApiError): number | undefined {
  return err?.response?.status
}

export function apiErrorMessage(err: ApiError, fallback: string): string {
  return err?.response?.data?.message ?? err?.response?.data?.errorMessage ?? fallback
}

// isSlugConflict detects the CreatePlaytest duplicate-slug case. The backend
// returns gRPC AlreadyExists (numeric code 6), which grpc-gateway maps to HTTP
// 409. The error message text is implementation-defined, so we match the code
// or status — never the literal string.
export function isSlugConflict(err: ApiError): boolean {
  if (apiErrorCode(err) === 6) return true
  return apiErrorStatus(err) === 409
}

export function toastError(verb: string) {
  return (err: ApiError) => message.error(apiErrorMessage(err, `Failed to ${verb}`))
}
