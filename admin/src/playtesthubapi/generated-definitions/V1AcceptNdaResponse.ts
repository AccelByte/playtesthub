/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1NdaAcceptance } from './V1NdaAcceptance.js'

export const V1AcceptNdaResponse = z.object({ acceptance: V1NdaAcceptance.nullish() })

export interface V1AcceptNdaResponse extends z.TypeOf<typeof V1AcceptNdaResponse> {}
