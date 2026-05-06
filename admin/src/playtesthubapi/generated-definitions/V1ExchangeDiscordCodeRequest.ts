/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1ExchangeDiscordCodeRequest = z.object({ code: z.string().nullish(), redirectUri: z.string().nullish() })

export interface V1ExchangeDiscordCodeRequest extends z.TypeOf<typeof V1ExchangeDiscordCodeRequest> {}
