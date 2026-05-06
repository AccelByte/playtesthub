/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyAnswer } from './V1SurveyAnswer.js'

export const PlaytesthubServiceSubmitSurveyResponseBody = z.object({
  surveyId: z.string().nullish(),
  answers: z.array(V1SurveyAnswer).nullish()
})

export interface PlaytesthubServiceSubmitSurveyResponseBody extends z.TypeOf<typeof PlaytesthubServiceSubmitSurveyResponseBody> {}
