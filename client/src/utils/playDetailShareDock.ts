import type { BetMultiplierPayload } from '@/api/schemes/betMultiplier'

const PREFIX = 'client:play-detail-share-dock:v1:'

export interface PlayDetailShareDockState {
  entryMode?: 'manual' | 'cloud'
  betMultiplierKind: '' | '0' | '1' | '2' | '3'
  betMultiplier?: BetMultiplierPayload
}

function storageKey(snapshotId: string): string {
  return `${PREFIX}${snapshotId.trim() || '__no_snapshot__'}`
}

export function loadPlayDetailShareDock(snapshotId: string): PlayDetailShareDockState | null {
  try {
    const raw = sessionStorage.getItem(storageKey(snapshotId))
    if (!raw) return null
    return JSON.parse(raw) as PlayDetailShareDockState
  } catch {
    return null
  }
}

export function savePlayDetailShareDock(snapshotId: string, state: PlayDetailShareDockState): void {
  try {
    sessionStorage.setItem(storageKey(snapshotId), JSON.stringify(state))
  } catch {
    /* ignore quota */
  }
}

export function clearPlayDetailShareDock(snapshotId: string): void {
  try {
    sessionStorage.removeItem(storageKey(snapshotId))
  } catch {
    /* ignore */
  }
}
