/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1PublicPlaytest } from './V1PublicPlaytest.js'

export const V1GetPublicPlaytestResponse = z.object({ playtest: V1PublicPlaytest.nullish() })

export interface V1GetPublicPlaytestResponse extends z.TypeOf<typeof V1GetPublicPlaytestResponse> {}
