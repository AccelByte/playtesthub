/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1CodeState } from './V1CodeState.js'

export const V1Code = z.object({
  id: z.string().nullish(),
  playtestId: z.string().nullish(),
  value: z.string().nullish(),
  state: V1CodeState.nullish(),
  reservedBy: z.string().nullish(),
  reservedAt: z.string().nullish(),
  grantedAt: z.string().nullish(),
  createdAt: z.string().nullish()
})

export interface V1Code extends z.TypeOf<typeof V1Code> {}
