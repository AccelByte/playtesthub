/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1UploadCodesRejection = z.object({
  lineNumber: z.number().int().nullish(),
  reason: z.string().nullish(),
  value: z.string().nullish()
})

export interface V1UploadCodesRejection extends z.TypeOf<typeof V1UploadCodesRejection> {}
