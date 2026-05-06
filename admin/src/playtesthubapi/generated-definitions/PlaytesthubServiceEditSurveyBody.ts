/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyQuestion } from './V1SurveyQuestion.js'

export const PlaytesthubServiceEditSurveyBody = z.object({ questions: z.array(V1SurveyQuestion).nullish() })

export interface PlaytesthubServiceEditSurveyBody extends z.TypeOf<typeof PlaytesthubServiceEditSurveyBody> {}
