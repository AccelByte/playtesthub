/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1UploadCodesRejection } from './V1UploadCodesRejection.js'

export const V1UploadCodesResponse = z.object({
  inserted: z.number().int().nullish(),
  rejections: z.array(V1UploadCodesRejection).nullish()
})

export interface V1UploadCodesResponse extends z.TypeOf<typeof V1UploadCodesResponse> {}
