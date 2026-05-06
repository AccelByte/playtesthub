/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1DistributionModel } from './V1DistributionModel.js'
import { V1Platform } from './V1Platform.js'
import { V1PlaytestStatus } from './V1PlaytestStatus.js'

export const V1PlayerPlaytest = z.object({
  slug: z.string().nullish(),
  title: z.string().nullish(),
  description: z.string().nullish(),
  bannerImageUrl: z.string().nullish(),
  platforms: z.array(V1Platform).nullish(),
  startsAt: z.string().nullish(),
  endsAt: z.string().nullish(),
  status: V1PlaytestStatus.nullish(),
  ndaRequired: z.boolean().nullish(),
  ndaText: z.string().nullish(),
  currentNdaVersionHash: z.string().nullish(),
  surveyId: z.string().nullish(),
  distributionModel: V1DistributionModel.nullish()
})

export interface V1PlayerPlaytest extends z.TypeOf<typeof V1PlayerPlaytest> {}
