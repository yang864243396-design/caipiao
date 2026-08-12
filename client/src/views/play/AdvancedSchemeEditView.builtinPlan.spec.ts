import { flushPromises, shallowMount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdvancedSchemeEditView from './AdvancedSchemeEditView.vue'

vi.mock('@/api/schemes/addToCloud', () => ({
  addSchemeToCloud: vi.fn(),
}))

vi.mock('@/api/schemes/definitions', () => ({
  checkSchemeNameAvailable: vi.fn(),
  createScheme: vi.fn(),
  fetchSchemeDefinitions: vi.fn(async () => ({ items: [] })),
  updateSchemeDefinition: vi.fn(),
  deleteSchemeDefinition: vi.fn(),
  fetchHotColdWarmTiers: vi.fn(),
}))

vi.mock('@/api/schemes/favorites', () => ({
  fetchSchemeFavorites: vi.fn(async () => [
    {
      snapshotId: 'fav-lhc-1',
      schemeName: '六合彩收藏方案',
      lotteryCode: 'lhc',
      lotteryLabel: '六合彩',
      playMethod: '二全中复式',
      favoredAt: '2026-08-11T00:00:00Z',
    },
  ]),
}))

vi.mock('@/api/schemes/schemeOptions', () => ({
  fetchLotterySchemeOptions: vi.fn(async () => ({
    lotteryCode: 'lhc',
    runTypes: [{ label: '内置计划', value: 'builtin_plan' }],
    playTypes: [],
    subPlays: [],
  })),
}))

vi.mock('@/api/games/detail', () => ({
  fetchGameDraws: vi.fn(),
}))

vi.mock('@/api/games/lotteries', () => ({
  fetchPlayTree: vi.fn(async () => ({
    playTypes: [
      {
        typeId: 'lianma',
        label: '连码',
        subPlays: [{ subId: 'erquanzhong', label: '二全中复式' }],
      },
    ],
  })),
}))

vi.mock('@/composables/usePublicLotteries', () => ({
  usePublicLotteries: () => ({
    lotteries: ref([{ code: 'lhc', label: '六合彩' }]),
    load: vi.fn(async () => {}),
    codeToLabel: (code: string) => code,
  }),
}))

vi.mock('@/composables/usePlayTreeConfig', () => ({
  usePlayTreeConfig: () => ({
    playConfig: ref({
      playTemplate: 'lhc_std',
      playTypeId: 'lianma',
      subPlayId: 'erquanzhong',
      betMode: 'fushi',
      segmentLen: 1,
      segmentLabels: [],
      inputMode: 'lhc_numbers',
    }),
    load: vi.fn(async () => {}),
  }),
}))

function createBuiltinPlanRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/play/bet-multiplier/advanced-scheme/:schemeId',
        component: { template: '<div />' },
      },
    ],
  })
}

async function mountBuiltinPlanEditor() {
  const router = createBuiltinPlanRouter()
  await router.push('/play/bet-multiplier/advanced-scheme/new?draft=1&lottery=lhc&runType=builtin_plan')
  await router.isReady()
  const wrapper = shallowMount(AdvancedSchemeEditView, {
    global: { plugins: [router, ElementPlus] },
  })
  await flushPromises()
  return wrapper
}

describe('AdvancedSchemeEditView 内置计划选择', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  afterEach(() => {
    sessionStorage.clear()
  })

  it('点击收藏方案后立即应用并保持在列表中选中', async () => {
    const wrapper = await mountBuiltinPlanEditor()

    await wrapper.find('.scf-bp-item').trigger('click')
    await flushPromises()

    expect(wrapper.find('.scf-bp-list').exists()).toBe(true)
    expect(wrapper.find('.scf-bp-item.is-sel').text()).toContain('六合彩收藏方案')
  })

  it('选择后不进入已跟随摘要页', async () => {
    const wrapper = await mountBuiltinPlanEditor()
    await wrapper.find('.scf-bp-item').trigger('click')
    await flushPromises()

    expect(wrapper.find('.scf-bp-summary').exists()).toBe(false)
  })
})
