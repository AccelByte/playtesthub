/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1Platform } from './V1Platform.js'

export const PlaytesthubServiceSignupBody = z.object({ platforms: z.array(V1Platform).nullish() })

export interface PlaytesthubServiceSignupBody extends z.TypeOf<typeof PlaytesthubServiceSignupBody> {}
