/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1Platform } from './V1Platform.js'

export const V1PublicPlaytest = z.object({
  slug: z.string().nullish(),
  title: z.string().nullish(),
  description: z.string().nullish(),
  bannerImageUrl: z.string().nullish(),
  platforms: z.array(V1Platform).nullish(),
  startsAt: z.string().nullish(),
  endsAt: z.string().nullish()
})

export interface V1PublicPlaytest extends z.TypeOf<typeof V1PublicPlaytest> {}
