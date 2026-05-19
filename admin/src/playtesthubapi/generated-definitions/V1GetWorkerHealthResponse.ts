/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1WorkerHealthEntry } from './V1WorkerHealthEntry.js'

export const V1GetWorkerHealthResponse = z.object({ workers: z.array(V1WorkerHealthEntry).nullish() })

export interface V1GetWorkerHealthResponse extends z.TypeOf<typeof V1GetWorkerHealthResponse> {}
