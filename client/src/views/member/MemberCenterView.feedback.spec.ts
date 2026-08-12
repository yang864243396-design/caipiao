import { mount, flushPromises } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { setAccessToken } from '@/api/client'
import { seedGuajiAuthCache } from '@/composables/useGuajiAuthGuard'
import { router } from '@/router'
import MemberCenterView from './MemberCenterView.vue'

vi.mock('@/api/member/profile', () => ({
  fetchMemberProfile: async () => ({
    memberId: 1,
    account: 'member-test',
    displayName: 'Member Test',
    currency: 'USDT',
  }),
}))

vi.mock('@/api/guaji/accounts', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/guaji/accounts')>()
  return {
    ...actual,
    fetchGuajiAuthStatus: async () => ({
      hasActiveGuajiAuth: true,
      bindingCount: 1,
      activeUsername: 'member-test',
    }),
    fetchGuajiBalance: async () => ({
      currency: 'USDT',
      amount: 10,
      username: 'member-test',
      usdt: 10,
      trx: 0,
      cny: 0,
    }),
  }
})

describe('MemberCenterView feedback visibility', () => {
  beforeEach(() => {
    vi.stubGlobal('scrollTo', vi.fn())
  })

  afterEach(() => {
    setAccessToken(null)
    vi.unstubAllGlobals()
  })

  it('does not render an 意见回馈 card in the member center', async () => {
    const viewRouter = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await viewRouter.push('/')
    await viewRouter.isReady()

    const wrapper = mount(MemberCenterView, {
      global: { plugins: [viewRouter, ElementPlus] },
    })
    await flushPromises()

    expect(wrapper.text()).not.toContain('意见回馈')
  })

  it('redirects the legacy feedback URL to the member center', async () => {
    setAccessToken('test-token')
    seedGuajiAuthCache({
      hasActiveGuajiAuth: true,
      bindingCount: 1,
      activeUsername: 'member-test',
    })

    await router.push('/member/feedback')

    expect(router.currentRoute.value.name).toBe('member')
  })
})
