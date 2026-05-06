/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyMultiChoiceAnswer } from './V1SurveyMultiChoiceAnswer.js'

export const V1SurveyAnswer = z.object({
  questionId: z.string().nullish(),
  text: z.string().nullish(),
  rating: z.number().int().nullish(),
  multiChoice: V1SurveyMultiChoiceAnswer.nullish()
})

export interface V1SurveyAnswer extends z.TypeOf<typeof V1SurveyAnswer> {}
