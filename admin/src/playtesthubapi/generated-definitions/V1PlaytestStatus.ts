/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1PlaytestStatus = z.any()

export interface V1PlaytestStatus extends z.TypeOf<typeof V1PlaytestStatus> {}
