/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'

export const V1AuditLogEntry = z.object({
  id: z.string().nullish(),
  namespace: z.string().nullish(),
  playtestId: z.string().nullish(),
  actorUserId: z.string().nullish(),
  action: z.string().nullish(),
  beforeJson: z.string().nullish(),
  afterJson: z.string().nullish(),
  createdAt: z.string().nullish()
})

export interface V1AuditLogEntry extends z.TypeOf<typeof V1AuditLogEntry> {}
