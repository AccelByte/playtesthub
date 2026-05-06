/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyQuestion } from './V1SurveyQuestion.js'

export const PlaytesthubServiceCreateSurveyBody = z.object({ questions: z.array(V1SurveyQuestion).nullish() })

export interface PlaytesthubServiceCreateSurveyBody extends z.TypeOf<typeof PlaytesthubServiceCreateSurveyBody> {}
