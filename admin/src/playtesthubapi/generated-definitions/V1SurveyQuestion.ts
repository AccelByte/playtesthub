/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1MultiChoiceOption } from './V1MultiChoiceOption.js'
import { V1SurveyQuestionType } from './V1SurveyQuestionType.js'

export const V1SurveyQuestion = z.object({
  id: z.string().nullish(),
  type: V1SurveyQuestionType.nullish(),
  prompt: z.string().nullish(),
  required: z.boolean().nullish(),
  options: z.array(V1MultiChoiceOption).nullish(),
  allowMultiple: z.boolean().nullish()
})

export interface V1SurveyQuestion extends z.TypeOf<typeof V1SurveyQuestion> {}
