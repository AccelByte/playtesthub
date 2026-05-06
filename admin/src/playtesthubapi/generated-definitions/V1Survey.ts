/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyQuestion } from './V1SurveyQuestion.js'

export const V1Survey = z.object({
  id: z.string().nullish(),
  playtestId: z.string().nullish(),
  version: z.number().int().nullish(),
  questions: z.array(V1SurveyQuestion).nullish(),
  createdAt: z.string().nullish()
})

export interface V1Survey extends z.TypeOf<typeof V1Survey> {}
