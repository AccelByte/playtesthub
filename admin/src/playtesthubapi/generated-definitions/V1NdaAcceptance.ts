/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1NdaAcceptance = z.object({
  userId: z.string().nullish(),
  playtestId: z.string().nullish(),
  ndaVersionHash: z.string().nullish(),
  acceptedAt: z.string().nullish()
})

export interface V1NdaAcceptance extends z.TypeOf<typeof V1NdaAcceptance> {}
