import { describe, expect, it } from 'vitest'
import {
  resolveSchemeMinBetPlayConfig,
  schemeMinBetOpenError,
  schemeMinBetUnits,
  schemeMinSingleBetAmount,
} from './schemeMinBet'

describe('scheme minimum amount for qian-zhong-hou three span', () => {
  const config = {
    typeId: 'g007',
    subId: '104',
    playTypeId: 'g007',
    subPlayId: '104',
    playTemplate: 'ssc_std',
    betMode: 'kuadu',
    betUnit: '0.001',
    schemeCurrency: 'USDT',
    schemeGroups: ['0,1,2,3,4,5'],
    rounds: [{ mult: 1, afterHit: 0, afterMiss: 1 }],
  }

  it('counts 1740 bets and permits a 1.74 USDT first bet', () => {
    expect(schemeMinBetUnits(config)).toBe(1740)
    expect(schemeMinSingleBetAmount(config, 1)).toBe(1.74)
    expect(schemeMinBetOpenError(config, 1, 'USDT')).toBeNull()
  })

  it('keeps PK10 g007 as a five-position play', () => {
    const pk10Config = {
      playTemplate: 'pk10_std',
      typeId: 'g007',
      subId: '200',
      betMode: 'fushi',
      betUnit: '0.01',
      schemeGroups: ['01,02\n03,04\n05,06\n07,08\n09,10'],
      rounds: [{ mult: 1 }],
    }

    expect(resolveSchemeMinBetPlayConfig(pk10Config)).toMatchObject({
      playTemplate: 'pk10_std',
      playTypeId: 'g007',
      segmentLen: 5,
      betMode: 'fushi',
    })
  })
})
