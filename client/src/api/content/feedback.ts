import { requestApi } from '@/api/client'

import { ensureClientSession } from '@/api/auth'



export interface FeedbackInput {

  subject: string

  content: string

}



export interface FeedbackResult {

  id: number

  subject: string

  createdAt: string

}



export async function submitFeedback(input: FeedbackInput): Promise<FeedbackResult> {

  await ensureClientSession()

  return requestApi<FeedbackResult>('/client/content/feedback', {

    method: 'POST',

    body: input,

  })

}

