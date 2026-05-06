/*
 * Copyright (c) 2022-2026 AccelByte Inc. All Rights Reserved
 * This is licensed software from AccelByte Inc, for limitations
 * and restrictions contact your company contract manager.
 */
import { z } from 'zod'
import { ProtobufAny } from './ProtobufAny.js'

export const RpcStatus = z.object({
  code: z.number().int().nullish(),
  message: z.string().nullish(),
  details: z.array(ProtobufAny).nullish()
})

export interface RpcStatus extends z.TypeOf<typeof RpcStatus> {}
