/**
 * 附录 A：`client` ↔ `admin` Mock 关键字段对照（双端同源）
 * @see docs/admin-frontend-plan.md 附录 A
 */
export const MOCK_APPENDIX = {
  memberAId: 'M00001',
  memberBId: 'M00002',
  memberCId: 'M00003',
  memberA: '会员甲',
  memberB: '会员乙',
  memberC: '会员丙',
  agentL1Code: 'AGT-L1-JIA',
  agentL2Code: 'AGT-L2-YI',
  withdrawalOrderId: 'WD00001',
  rechargeOrderId: 'RC-MOCK-JIA-20260519',
  schemeAdvancedId: 'scheme_demo_1001',
  schemeCloudId: 'cloud_demo_2001',
  schemeCopyId: 'copy_demo_3001',
  schemeCustomId: 'custom_demo_4001',
} as const

export type MockAppendix = typeof MOCK_APPENDIX
