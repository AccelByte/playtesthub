/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1RetryFailedDmsResponse = z.object({ enqueued: z.number().int().nullish(), overflow: z.number().int().nullish() })

export interface V1RetryFailedDmsResponse extends z.TypeOf<typeof V1RetryFailedDmsResponse> {}
