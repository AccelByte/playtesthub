/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1WorkerHealthEntry = z.object({
  name: z.string().nullish(),
  leaseHolder: z.string().nullish(),
  lastTickAt: z.string().nullish(),
  expiresAt: z.string().nullish(),
  stale: z.boolean().nullish()
})

export interface V1WorkerHealthEntry extends z.TypeOf<typeof V1WorkerHealthEntry> {}
