/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyAnswer } from './V1SurveyAnswer.js'

export const V1SurveyResponse = z.object({
  id: z.string().nullish(),
  playtestId: z.string().nullish(),
  userId: z.string().nullish(),
  surveyId: z.string().nullish(),
  answers: z.array(V1SurveyAnswer).nullish(),
  submittedAt: z.string().nullish()
})

export interface V1SurveyResponse extends z.TypeOf<typeof V1SurveyResponse> {}
