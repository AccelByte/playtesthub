/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1PlaytestStatus } from './V1PlaytestStatus.js'

export const PlaytesthubServiceTransitionPlaytestStatusBody = z.object({ targetStatus: V1PlaytestStatus.nullish() })

export interface PlaytesthubServiceTransitionPlaytestStatusBody extends z.TypeOf<typeof PlaytesthubServiceTransitionPlaytestStatusBody> {}
