import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SchemeDownloadView from './SchemeDownloadView.vue'
import { fetchShareCatalog, fetchShareCatalogRows, toDownloadRow } from '@/api/schemes/shareCatalog'

vi.mock('vue-router', () => ({
  useRouter: () => ({ back: vi.fn(), push: vi.fn() }),
}))

vi.mock('@/api/schemes/shareCatalog', () => ({
  fetchShareCatalog: vi.fn(),
  fetchShareCatalogRows: vi.fn(),
  toDownloadRow: vi.fn((item) => ({
    schemeId: item.id,
    schemeName: item.schemeName,
    lotteryLabel: item.lotteryLabel ?? item.lotteryCode,
    playMethod: item.playMethod ?? '测试玩法',
    fundYuan: item.fundYuan ?? 0,
    schemeCurrency: 'USDT',
  })),
}))

vi.mock('@/api/schemes/shareAddToCloud', () => ({ shareAddToCloud: vi.fn() }))

const inputStub = {
  name: 'ElInput',
  inheritAttrs: false,
  props: ['modelValue'],
  emits: ['update:modelValue', 'clear'],
  template: '<div />',
}
const buttonStub = {
  name: 'ElButton',
  emits: ['click'],
  template: '<button @click="$emit(\'click\')"><slot /></button>',
}
const stubs = {
  'el-input': inputStub,
  'el-button': buttonStub,
  'el-table': true,
  'el-table-column': true,
  ConfirmDialog: true,
}
const global = { stubs, directives: { loading: {} } }

function snapshot(id: string) {
  return {
    id,
    kind: 'custom' as const,
    schemeName: `方案 ${id}`,
    lotteryCode: 'fc3d',
    config: {},
    createdAt: '',
    updatedAt: '',
  }
}

describe('SchemeDownloadView 分页', () => {
  beforeEach(() => {
    vi.mocked(fetchShareCatalog).mockReset()
    vi.mocked(fetchShareCatalogRows).mockReset().mockResolvedValue([])
    vi.mocked(toDownloadRow).mockClear()
  })

  it('默认每页 20 条，点击加载更多后按游标追加下一页', async () => {
    vi.mocked(fetchShareCatalog)
      .mockResolvedValueOnce({ items: [snapshot('SD20')], page: { hasMore: true, nextCursor: 'SD20' } })
      .mockResolvedValueOnce({ items: [snapshot('SD21')], page: { hasMore: false } })

    const wrapper = shallowMount(SchemeDownloadView, { global })
    await flushPromises()

    expect(fetchShareCatalog).toHaveBeenCalledWith({ limit: 20 })
    expect(wrapper.find('[data-testid="scheme-download-load-more"]').exists()).toBe(true)

    await wrapper.find('[data-testid="scheme-download-load-more"]').trigger('click')
    await flushPromises()

    expect(fetchShareCatalog).toHaveBeenLastCalledWith({ cursor: 'SD20', limit: 20 })
    expect(toDownloadRow).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-testid="scheme-download-load-more"]').exists()).toBe(false)
  })

  it('搜索和重置均从第一页重新加载', async () => {
    vi.mocked(fetchShareCatalog)
      .mockResolvedValueOnce({ items: [snapshot('SD1')], page: { hasMore: true, nextCursor: 'SD1' } })
      .mockResolvedValueOnce({ items: [snapshot('SD-search')], page: { hasMore: false } })
      .mockResolvedValueOnce({ items: [snapshot('SD-reset')], page: { hasMore: false } })

    const wrapper = shallowMount(SchemeDownloadView, { global })
    await flushPromises()

    const input = wrapper.findComponent(inputStub)
    input.vm.$emit('update:modelValue', 'SD-search')
    await nextTick()
    await wrapper.find('.sdw-search-btn').trigger('click')
    await flushPromises()
    expect(fetchShareCatalog).toHaveBeenLastCalledWith({ keyword: 'SD-search', limit: 20 })

    input.vm.$emit('clear')
    await flushPromises()
    expect(fetchShareCatalog).toHaveBeenLastCalledWith({ limit: 20 })
  })
})
