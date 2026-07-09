import { requestApi } from './client'

export interface CustomerServiceAgent {
  id: string
  name: string
  tgLink: string
  workHours: string
  sort: number
  enabled: boolean
}

export async function fetchCustomerServiceAgents(): Promise<CustomerServiceAgent[]> {
  const res = await requestApi<{ items: CustomerServiceAgent[] }>('/admin/service/customer-service/agents')
  return res.items ?? []
}

export async function saveCustomerServiceAgent(body: CustomerServiceAgent): Promise<CustomerServiceAgent> {
  return requestApi<CustomerServiceAgent>('/admin/service/customer-service/agents', { method: 'PUT', body })
}

export async function deleteCustomerServiceAgent(id: string): Promise<void> {
  await requestApi(`/admin/service/customer-service/agents/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
