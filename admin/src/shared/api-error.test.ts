import { describe, expect, it } from 'vitest'
import { apiErrorCode, apiErrorMessage, apiErrorStatus, isSlugConflict, type ApiError } from './api-error'

describe('api-error helpers', () => {
  describe('apiErrorMessage', () => {
    it('prefers the grpc-gateway data.message field', () => {
      const err: ApiError = { response: { status: 500, data: { code: 13, message: 'boom from gateway' } } }
      expect(apiErrorMessage(err, 'fallback')).toBe('boom from gateway')
    })

    it('falls back to legacy errorMessage when message is absent', () => {
      const err: ApiError = { response: { data: { errorMessage: 'legacy text' } } }
      expect(apiErrorMessage(err, 'fallback')).toBe('legacy text')
    })

    it('returns the fallback when neither field is present', () => {
      expect(apiErrorMessage({}, 'Create failed')).toBe('Create failed')
    })
  })

  describe('apiErrorCode / apiErrorStatus', () => {
    it('reads the numeric gRPC code from response.data.code', () => {
      expect(apiErrorCode({ response: { data: { code: 6 } } })).toBe(6)
    })

    it('reads the HTTP status from response.status', () => {
      expect(apiErrorStatus({ response: { status: 409 } })).toBe(409)
    })
  })

  describe('isSlugConflict', () => {
    it('is true when the gRPC code is 6 (AlreadyExists)', () => {
      const err: ApiError = { response: { status: 409, data: { code: 6, message: 'slug "x" already exists in namespace "y"' } } }
      expect(isSlugConflict(err)).toBe(true)
    })

    it('is true when the HTTP status is 409 even without a numeric code', () => {
      expect(isSlugConflict({ response: { status: 409, data: {} } })).toBe(true)
    })

    it('is false for an unrelated error (code 13 / status 500)', () => {
      expect(isSlugConflict({ response: { status: 500, data: { code: 13, message: 'internal' } } })).toBe(false)
    })

    it('is false for an empty error', () => {
      expect(isSlugConflict({})).toBe(false)
    })
  })
})
