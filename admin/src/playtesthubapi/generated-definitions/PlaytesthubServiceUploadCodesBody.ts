/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const PlaytesthubServiceUploadCodesBody = z.object({ csvContent: z.string().nullish(), filename: z.string().nullish() })

export interface PlaytesthubServiceUploadCodesBody extends z.TypeOf<typeof PlaytesthubServiceUploadCodesBody> {}
