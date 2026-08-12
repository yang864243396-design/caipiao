import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SchemeLhcGuoguanPanel from './SchemeLhcGuoguanPanel.vue'

const config = {
  playTemplate: 'lhc_std',
  playTypeId: 'g004',
  subPlayId: 'guoguan',
  betMode: 'guoguan',
  segmentLen: 1,
  segmentLabels: ['过关'],
  inputMode: 'lhc_attr' as const,
}

describe('SchemeLhcGuoguanPanel', () => {
  it('renders six fixed zhengma positions as two-column select controls and preserves omitted positions', async () => {
    const wrapper = mount(SchemeLhcGuoguanPanel, {
      props: { config, modelValue: '大,单,,大,,双' },
    })

    const rows = wrapper.findAll('.sgg-position')
    expect(rows).toHaveLength(6)
    expect(wrapper.text()).toContain('正码1')
    expect(wrapper.text()).toContain('正码6')
    expect(wrapper.find('.sgg-hint').exists()).toBe(false)
    expect(wrapper.find('.sgg-wire').exists()).toBe(false)
    expect(wrapper.findAll('.sgg-positions > .sgg-position').length).toBe(6)
    expect(rows[2]!.find('select').element.value).toBe('')

    await rows[2]!.find('select').setValue('红波')
    const emissions = wrapper.emitted('update:modelValue') ?? []
    expect(emissions[emissions.length - 1]).toEqual(['大,单,红波,大,,双'])
  })
})
