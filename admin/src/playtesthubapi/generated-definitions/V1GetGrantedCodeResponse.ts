/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1DistributionModel } from './V1DistributionModel.js'

export const V1GetGrantedCodeResponse = z.object({ value: z.string().nullish(), distributionModel: V1DistributionModel.nullish() })

export interface V1GetGrantedCodeResponse extends z.TypeOf<typeof V1GetGrantedCodeResponse> {}
