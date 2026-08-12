import { describe, expect, it } from 'vitest'
import { toDownloadRow, type SchemeShareSnapshot } from './shareCatalog'

function snapshot(config: Record<string, unknown>): SchemeShareSnapshot {
  return {
    id: 'SD1',
    kind: 'custom',
    schemeName: '币种方案',
    lotteryCode: 'fc3d',
    fundYuan: 100,
    config,
    createdAt: '',
    updatedAt: '',
  }
}

describe('toDownloadRow', () => {
  it('使用方案配置中的币种，缺失时兼容为 USDT', () => {
    expect(toDownloadRow(snapshot({ schemeCurrency: 'trx' })).schemeCurrency).toBe('TRX')
    expect(toDownloadRow(snapshot({ schemeCurrency: 'CNY' })).schemeCurrency).toBe('CNY')
    expect(toDownloadRow(snapshot({})).schemeCurrency).toBe('USDT')
  })
})
