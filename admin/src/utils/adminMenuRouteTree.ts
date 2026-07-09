import {
  ADMIN_MENU_ALL_NODE_ID,
  ADMIN_MENU_ROUTE_TREE,
  type AdminMenuRouteNode,
} from '@/constants/adminMenuRouteTree'
import { canAccessPath } from '@/utils/menuRbac'

function walkNodes(nodes: AdminMenuRouteNode[], visit: (node: AdminMenuRouteNode) => void) {
  for (const node of nodes) {
    visit(node)
    if (node.children?.length) walkNodes(node.children, visit)
  }
}

/** menuPaths → 树节点 id（用于回显勾选） */
export function checkedKeysFromMenuPaths(menuPaths: string[]): string[] {
  if (!menuPaths.length) return []
  if (menuPaths.some((p) => p === '/' || p === '/*')) return [ADMIN_MENU_ALL_NODE_ID]

  const keys: string[] = []
  walkNodes(ADMIN_MENU_ROUTE_TREE, (node) => {
    if (!node.path || node.path === '/') return
    if (canAccessPath(node.path, menuPaths)) keys.push(node.id)
  })
  return keys
}

/** 已勾选树节点 → menuPaths */
export function menuPathsFromCheckedNodes(nodes: AdminMenuRouteNode[]): string[] {
  if (nodes.some((node) => node.path === '/')) return ['/']

  const paths = nodes
    .map((node) => node.path)
    .filter((path): path is string => !!path && path !== '/')

  return [...new Set(paths)].sort()
}

/** menuPaths → 与编辑树一致的中文菜单名 */
export function menuLabelsFromMenuPaths(menuPaths: string[]): string[] {
  if (!menuPaths.length) return []
  if (menuPaths.some((p) => p === '/' || p === '/*')) {
    const allNode = ADMIN_MENU_ROUTE_TREE.find((node) => node.id === ADMIN_MENU_ALL_NODE_ID)
    return [allNode?.label ?? '全部菜单']
  }

  const labels: string[] = []
  walkNodes(ADMIN_MENU_ROUTE_TREE, (node) => {
    if (!node.path || node.path === '/') return
    if (canAccessPath(node.path, menuPaths)) labels.push(node.label)
  })
  return labels
}

export function formatMenuLabelsSummary(menuPaths: string[]): string {
  const labels = menuLabelsFromMenuPaths(menuPaths)
  if (!labels.length) return '未配置'
  const preview = labels.slice(0, 3).join(' · ')
  return labels.length > 3 ? `${preview} …（共 ${labels.length} 项）` : preview
}
