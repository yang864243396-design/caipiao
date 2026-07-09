/** 角色 menuPaths 树形配置，与 AdminLayout 侧栏菜单一致 */
export interface AdminMenuRouteNode {
  id: string
  label: string
  /** 写入 menuPaths 的路由前缀；「/」表示全部菜单 */
  path?: string
  children?: AdminMenuRouteNode[]
}

export const ADMIN_MENU_ALL_NODE_ID = '__all__'

export const ADMIN_MENU_ROUTE_TREE: AdminMenuRouteNode[] = [
  { id: ADMIN_MENU_ALL_NODE_ID, label: '全部菜单', path: '/' },
  { id: '/dashboard', label: '仪表盘', path: '/dashboard' },
  {
    id: 'group-operations',
    label: '运维',
    children: [{ id: '/operations/maintenance', label: '系统维护', path: '/operations/maintenance' }],
  },
  {
    id: 'group-members',
    label: '会员与用户',
    children: [
      { id: '/members', label: '会员查询', path: '/members' },
      { id: '/schemes/monitor', label: '全站方案监控', path: '/schemes/monitor' },
    ],
  },
  {
    id: 'group-games',
    label: '游戏与玩法',
    children: [
      { id: '/games/lottery-catalog', label: '彩种目录', path: '/games/lottery-catalog' },
      { id: '/games/copy-hall', label: '跟单大厅运营', path: '/games/copy-hall' },
      { id: '/games/scheme-defaults', label: '方案模板库', path: '/games/scheme-defaults' },
    ],
  },
  {
    id: 'group-orders',
    label: '订单与帐变',
    children: [
      { id: '/orders/bets', label: '投注与追号', path: '/orders/bets' },
      { id: '/orders/ledger', label: '帐变流水', path: '/orders/ledger' },
    ],
  },
  { id: '/reports/lottery-stat', label: '经营报表', path: '/reports/lottery-stat' },
  {
    id: 'group-content',
    label: '站点与内容',
    children: [
      { id: '/content/banners', label: 'Banner 管理', path: '/content/banners' },
      { id: '/content/announcements', label: '公告管理', path: '/content/announcements' },
      { id: '/content/faq', label: '常见问题', path: '/content/faq' },
    ],
  },
  {
    id: 'group-service',
    label: '客服',
    children: [{ id: '/service/customer-service', label: '客服设置', path: '/service/customer-service' }],
  },
  {
    id: 'group-system',
    label: '系统',
    children: [
      { id: '/system/roles', label: '角色管理', path: '/system/roles' },
      { id: '/system/admin-users', label: 'Admin 账号', path: '/system/admin-users' },
      { id: '/system/audit', label: '操作审计', path: '/system/audit' },
    ],
  },
]
