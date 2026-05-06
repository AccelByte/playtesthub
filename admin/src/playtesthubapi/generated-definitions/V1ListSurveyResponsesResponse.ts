/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyResponse } from './V1SurveyResponse.js'

export const V1ListSurveyResponsesResponse = z.object({
  responses: z.array(V1SurveyResponse).nullish(),
  nextPageToken: z.string().nullish()
})

export interface V1ListSurveyResponsesResponse extends z.TypeOf<typeof V1ListSurveyResponsesResponse> {}
