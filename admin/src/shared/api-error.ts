import { message } from 'antd'

// ApiError matches the shape AccelByte codegen surfaces for failed
// gateway requests. The errorMessage from the gRPC status (when
// present) is preferred over the fallback verb.
export type ApiError = { response?: { data?: { errorMessage?: string } } }

export function toastError(verb: string) {
  return (err: ApiError) => message.error(err?.response?.data?.errorMessage ?? `Failed to ${verb}`)
}
