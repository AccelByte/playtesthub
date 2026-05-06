/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1MultiChoiceOption = z.object({ id: z.string().nullish(), label: z.string().nullish() })

export interface V1MultiChoiceOption extends z.TypeOf<typeof V1MultiChoiceOption> {}
