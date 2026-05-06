/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1Survey } from './V1Survey.js'

export const V1GetSurveyResponse = z.object({ survey: V1Survey.nullish() })

export interface V1GetSurveyResponse extends z.TypeOf<typeof V1GetSurveyResponse> {}
