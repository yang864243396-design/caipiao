/** 路径别名：新旧菜单 path 互通，避免 RBAC / 书签失效 */
const PATH_ALIASES: Record<string, string[]> = {
  '/content/banners': ['/content/lobby'],
  '/content/lobby': ['/content/banners'],
}

function expandPaths(path: string): string[] {
  const normalized = path.startsWith('/') ? path : `/${path}`
  const aliases = PATH_ALIASES[normalized] ?? []
  return [normalized, ...aliases]
}

/** 判断 path 是否在角色 menuPaths 白名单内（前缀匹配；「/」= 全部） */
export function canAccessPath(path: string, menuPaths: string[]): boolean {
  if (!menuPaths.length) return false
  if (menuPaths.some((p) => p === '/' || p === '/*')) return true
  return expandPaths(path).some((normalized) =>
    menuPaths.some((prefix) => {
      if (!prefix) return false
      const p = prefix.startsWith('/') ? prefix : `/${prefix}`
      return normalized === p || normalized.startsWith(`${p}/`)
    }),
  )
}

export function canAccessAny(path: string, prefixes: string[]): boolean {
  return prefixes.some((p) => canAccessPath(path, [p]))
}
