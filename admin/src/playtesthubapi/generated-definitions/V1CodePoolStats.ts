/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1CodePoolStats = z.object({
  total: z.number().int().nullish(),
  unused: z.number().int().nullish(),
  reserved: z.number().int().nullish(),
  granted: z.number().int().nullish()
})

export interface V1CodePoolStats extends z.TypeOf<typeof V1CodePoolStats> {}
