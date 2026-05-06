/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1CodePoolStats } from './V1CodePoolStats.js'

export const V1SyncFromAgsResponse = z.object({ pool: V1CodePoolStats.nullish(), added: z.number().int().nullish() })

export interface V1SyncFromAgsResponse extends z.TypeOf<typeof V1SyncFromAgsResponse> {}
