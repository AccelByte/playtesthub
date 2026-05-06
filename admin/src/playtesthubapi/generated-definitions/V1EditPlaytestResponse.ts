/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1Playtest } from './V1Playtest.js'

export const V1EditPlaytestResponse = z.object({ playtest: V1Playtest.nullish() })

export interface V1EditPlaytestResponse extends z.TypeOf<typeof V1EditPlaytestResponse> {}
