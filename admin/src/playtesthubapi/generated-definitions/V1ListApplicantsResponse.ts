/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1Applicant } from './V1Applicant.js'

export const V1ListApplicantsResponse = z.object({ applicants: z.array(V1Applicant).nullish(), nextPageToken: z.string().nullish() })

export interface V1ListApplicantsResponse extends z.TypeOf<typeof V1ListApplicantsResponse> {}
