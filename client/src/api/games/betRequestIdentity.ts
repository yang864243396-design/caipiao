export interface BetRequestIdentity {
  requestId: string
  storageKey: string
}

const storagePrefix = 'caipiao:pending-bet:'
const memoryIdentities = new Map<string, string>()

function stableSerialize(value: unknown): string {
  if (value === null || typeof value !== 'object') return JSON.stringify(value)
  if (Array.isArray(value)) return `[${value.map(stableSerialize).join(',')}]`
  const record = value as Record<string, unknown>
  return `{${Object.keys(record)
    .sort()
    .map(key => `${JSON.stringify(key)}:${stableSerialize(record[key])}`)
    .join(',')}}`
}

function fingerprint(value: string): string {
  let first = 0x811c9dc5
  let second = 0x9e3779b9
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index)
    first = Math.imul(first ^ code, 0x01000193)
    second = Math.imul(second ^ code, 0x85ebca6b)
  }
  return `${(first >>> 0).toString(36)}-${(second >>> 0).toString(36)}-${value.length.toString(36)}`
}

function createRequestId(): string {
  const uuid = globalThis.crypto?.randomUUID?.()
  if (uuid) return `web_${uuid}`
  const random = Math.random().toString(36).slice(2, 14)
  return `web_${Date.now().toString(36)}_${random}`
}

function readIdentity(storageKey: string): string | undefined {
  try {
    return sessionStorage.getItem(storageKey) || memoryIdentities.get(storageKey)
  } catch {
    return memoryIdentities.get(storageKey)
  }
}

function writeIdentity(storageKey: string, requestId: string): void {
  memoryIdentities.set(storageKey, requestId)
  try {
    sessionStorage.setItem(storageKey, requestId)
  } catch {
    // The in-memory value still preserves retries in this page session.
  }
}

export function acquireBetRequestIdentity(lotteryCode: string, input: unknown): BetRequestIdentity {
  const canonical = stableSerialize({ lotteryCode: lotteryCode.trim(), input })
  const storageKey = `${storagePrefix}${fingerprint(canonical)}`
  const existing = readIdentity(storageKey)
  if (existing) return { requestId: existing, storageKey }

  const requestId = createRequestId()
  writeIdentity(storageKey, requestId)
  return { requestId, storageKey }
}

export function releaseBetRequestIdentity(identity: BetRequestIdentity): void {
  if (readIdentity(identity.storageKey) !== identity.requestId) return
  memoryIdentities.delete(identity.storageKey)
  try {
    sessionStorage.removeItem(identity.storageKey)
  } catch {
    // The in-memory identity was already released.
  }
}
