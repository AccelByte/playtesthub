/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1ExchangeDiscordCodeResponse = z.object({
  accessToken: z.string().nullish(),
  refreshToken: z.string().nullish(),
  expiresIn: z.number().int().nullish(),
  tokenType: z.string().nullish()
})

export interface V1ExchangeDiscordCodeResponse extends z.TypeOf<typeof V1ExchangeDiscordCodeResponse> {}
