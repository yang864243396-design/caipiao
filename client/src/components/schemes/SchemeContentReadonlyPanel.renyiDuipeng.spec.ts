import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SchemeContentReadonlyPanel from './SchemeContentReadonlyPanel.vue'
import SchemeLhcRenyiDuipengPanel from './SchemeLhcRenyiDuipengPanel.vue'

const elementStubs = {
  'el-button': true,
  'el-empty': true,
  'el-input': true,
  'el-input-number': true,
  'el-popover': true,
  'el-radio': true,
  'el-radio-group': true,
  'el-switch': true,
}

describe('SchemeContentReadonlyPanel 任意对碰展示', () => {
  it('固定号码内容以 A/B 双区面板只读展示', () => {
    const wrapper = shallowMount(SchemeContentReadonlyPanel, {
      props: {
        runTypeId: 'fixed_rotate',
        runTypeLabel: '定码轮换',
        playConfig: {
          playTemplate: 'lhc_std',
          playTypeId: 'g003',
          subPlayId: '284',
          betMode: 'renyi_dp',
          segmentLen: 1,
          segmentLabels: [],
          inputMode: 'lhc_num',
        },
        schemeGroups: ['01,02|03,04'],
        jushuList: [],
        triggerBet: null,
        hotColdWarm: null,
        randomDraw: null,
      },
      global: { stubs: elementStubs },
    })

    const panel = wrapper.findComponent(SchemeLhcRenyiDuipengPanel)
    expect(panel.exists()).toBe(true)
    expect(panel.props('modelValue')).toBe('01,02|03,04')
    expect(panel.props('disabled')).toBe(true)
  })
})

describe('SchemeContentReadonlyPanel 任意对碰随机出号展示', () => {
  it('renders separate A and B counters and estimates their product', () => {
    const wrapper = shallowMount(SchemeContentReadonlyPanel, {
      props: {
        runTypeId: 'random_draw',
        runTypeLabel: '随机出号',
        playConfig: {
          playTemplate: 'lhc_std',
          playTypeId: 'g003',
          subPlayId: '284',
          betMode: 'renyi_dp',
          segmentLen: 1,
          segmentLabels: [],
          inputMode: 'lhc_num',
        },
        schemeGroups: [],
        jushuList: [],
        triggerBet: null,
        hotColdWarm: null,
        randomDraw: { counts: [2, 3], strategy: 'every' },
      },
      global: { stubs: elementStubs },
    })

    expect(wrapper.text()).toContain('A区随机')
    expect(wrapper.text()).toContain('B区随机')
    expect(wrapper.text()).toContain('预估 6 注')
    const counters = wrapper.findAll('el-input-number-stub')
    expect(counters).toHaveLength(2)
    expect(counters[0]?.attributes('model-value')).toBe('2')
    expect(counters[1]?.attributes('model-value')).toBe('3')
  })
})

describe('SchemeContentReadonlyPanel 内置计划展示', () => {
  it('显示被跟随收藏方案的名称，而不是当前方案实例名称', () => {
    const wrapper = shallowMount(SchemeContentReadonlyPanel, {
      props: {
        runTypeId: 'builtin_plan',
        runTypeLabel: '内置计划',
        playConfig: {
          playTemplate: 'ssc_std',
          playTypeId: 'dingwei',
          subPlayId: 'dingwei_ge',
          betMode: 'dingwei',
          segmentLen: 1,
          segmentLabels: [],
          inputMode: 'dingwei',
        },
        schemeGroups: [],
        jushuList: [],
        triggerBet: null,
        hotColdWarm: null,
        randomDraw: null,
        schemeName: '我的内置计划',
        builtinPlanSchemeName: '原收藏方案',
      },
      global: { stubs: elementStubs },
    })

    expect(wrapper.text()).toContain('已跟随：原收藏方案')
    expect(wrapper.text()).not.toContain('已跟随：我的内置计划')
  })
})
