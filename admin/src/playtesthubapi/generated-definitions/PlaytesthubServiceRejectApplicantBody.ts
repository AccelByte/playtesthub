/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const PlaytesthubServiceRejectApplicantBody = z.object({ rejectionReason: z.string().nullish() })

export interface PlaytesthubServiceRejectApplicantBody extends z.TypeOf<typeof PlaytesthubServiceRejectApplicantBody> {}
