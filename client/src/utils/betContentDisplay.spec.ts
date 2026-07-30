import { describe, it, expect } from 'vitest'
import { betContentLines } from './betContentDisplay'

// 这个模块是「我的投注位名错位」那个缺陷的修复落点。
// 原先前端按行序硬编码位名（万/千/百/十/个），中三码、后三码这类
// 非全五位的玩法就会被标错位——投注内容看着合法，位置全是错的。
// 修法是位名改由后端按玩法位段算好随详情下发，前端只在后端没给时才原样分行。
// 所以这里最要紧的一条是：后端给了 lines 就必须原样用，一个字都不能自己加。

describe('betContentLines', () => {
  describe('后端已下发位名时原样透传', () => {
    it('直接返回后端的行，不做任何加工', () => {
      const lines = ['百 十 个', '1 2 3']
      expect(betContentLines(lines, '1,2,3')).toEqual(lines)
    })

    it('后端的行优先于 raw，二者不一致时以后端为准', () => {
      // raw 是原始投注内容，lines 是后端算好位名的展示行。
      // 若这里回退去解析 raw，位名就丢了——正是修复前的错误行为。
      expect(betContentLines(['万 千', '3 7'], '9,9,9')).toEqual(['万 千', '3 7'])
    })

    it('后端只给一行也算给了', () => {
      expect(betContentLines(['大'], '1,2,3')).toEqual(['大'])
    })

    it('后端给空数组视为没给，回退解析 raw', () => {
      expect(betContentLines([], '1,2')).toEqual(['1 2'])
    })

    it('后端给 undefined 视为没给', () => {
      expect(betContentLines(undefined, '1,2')).toEqual(['1 2'])
    })
  })

  describe('后端未下发时按分隔符原样分行', () => {
    it.each([
      { name: '半角逗号', raw: '1,2,3', want: ['1 2 3'] },
      { name: '全角逗号', raw: '1，2，3', want: ['1 2 3'] },
      { name: '空格', raw: '1 2 3', want: ['1 2 3'] },
      { name: '竖线', raw: '1|2|3', want: ['1 2 3'] },
      { name: '混合分隔符', raw: '1, 2|3', want: ['1 2 3'] },
      { name: '连续分隔符不产生空项', raw: '1,,2', want: ['1 2'] },
      { name: '首尾分隔符', raw: ',1,2,', want: ['1 2'] },
    ])('$name', ({ raw, want }) => {
      expect(betContentLines(undefined, raw)).toEqual(want)
    })

    it('多行内容逐行处理', () => {
      expect(betContentLines(undefined, '1,2\n3,4')).toEqual(['1 2', '3 4'])
    })

    it('CRLF 换行与 LF 等价', () => {
      expect(betContentLines(undefined, '1,2\r\n3,4')).toEqual(['1 2', '3 4'])
    })

    it('空行被剔除', () => {
      expect(betContentLines(undefined, '1,2\n\n3,4')).toEqual(['1 2', '3 4'])
    })

    it('中文属性内容原样保留', () => {
      expect(betContentLines(undefined, '大,单')).toEqual(['大 单'])
      expect(betContentLines(undefined, '豹子,对子,顺子')).toEqual(['豹子 对子 顺子'])
    })
  })

  describe('空值一律显示破折号', () => {
    it.each([
      { name: '空串', raw: '' },
      { name: '只有分隔符', raw: ',,,' },
      { name: '只有换行', raw: '\n\n' },
      { name: '只有空格', raw: '   ' },
    ])('$name', ({ raw }) => {
      expect(betContentLines(undefined, raw)).toEqual(['—'])
    })

    it('null 与 undefined 也显示破折号', () => {
      expect(betContentLines(undefined, null as unknown as string)).toEqual(['—'])
      expect(betContentLines(undefined, undefined as unknown as string)).toEqual(['—'])
    })
  })
})
