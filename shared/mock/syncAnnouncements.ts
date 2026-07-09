/**
 * 与 admin `contentSeed.seedAnnouncements` 同结构的公告 Mock（维护弹窗 id 解析用）
 */
export interface SyncAnnouncement {
  id: string
  title: string
  status: '草稿' | '已发布'
  publishedAt: string | null
  bodyHtml: string
}

export function seedSyncAnnouncements(count = 16): SyncAnnouncement[] {
  const base: SyncAnnouncement[] = []
  for (let i = 1; i <= count; i++) {
    base.push({
      id: `ANN${String(i).padStart(4, '0')}`,
      title: `平台公告（Mock）第 ${i} 条`,
      status: i % 3 === 0 ? '草稿' : '已发布',
      publishedAt: i % 3 === 0 ? null : new Date(Date.now() - i * 86400000).toISOString(),
      bodyHtml: `<p>这是 <strong>公告 ${i}</strong> 的富文本 Mock（§8.10 Q71 不 sanitize）。</p><p>可含 <a href="#">外链占位</a>。</p>`,
    })
  }
  return base
}

const CACHE = seedSyncAnnouncements()

export function findSyncAnnouncement(id: string): SyncAnnouncement | undefined {
  return CACHE.find((a) => a.id === id)
}

export function listPublishedSyncAnnouncements(): SyncAnnouncement[] {
  return CACHE.filter((a) => a.status === '已发布' && a.publishedAt)
}
