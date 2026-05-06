/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1PlayerPlaytest } from './V1PlayerPlaytest.js'

export const V1GetPlaytestForPlayerResponse = z.object({ playtest: V1PlayerPlaytest.nullish() })

export interface V1GetPlaytestForPlayerResponse extends z.TypeOf<typeof V1GetPlaytestForPlayerResponse> {}
