/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { V1AuditLogEntry } from './V1AuditLogEntry.js'

export const V1ListAuditLogResponse = z.object({ entries: z.array(V1AuditLogEntry).nullish(), nextPageToken: z.string().nullish() })

export interface V1ListAuditLogResponse extends z.TypeOf<typeof V1ListAuditLogResponse> {}
