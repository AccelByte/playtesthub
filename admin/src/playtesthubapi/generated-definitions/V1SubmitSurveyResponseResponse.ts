/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1SurveyResponse } from './V1SurveyResponse.js'

export const V1SubmitSurveyResponseResponse = z.object({ response: V1SurveyResponse.nullish() })

export interface V1SubmitSurveyResponseResponse extends z.TypeOf<typeof V1SubmitSurveyResponseResponse> {}
