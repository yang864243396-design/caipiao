// 登录 hash.iyes.dev 测试账号，对指定 rule 批量探测 bet_content�?
import https from 'https'

const HTTP = 'https://www.v6hs1.com'
const AUTH = process.env.GUAJI_AUTH_BASE || 'https://www.v6hs1.com'
const ORIGIN = 'https://www.v6hs1.com'
const USER = process.env.GUAJI_TEST_USERNAME || 'vs8888'
const PASS = process.env.GUAJI_TEST_PASSWORD || 'vs8888'
const GAME = Number(process.env.GUAJI_PROBE_GAME || 41)
const RULE = process.env.GUAJI_PROBE_RULE || '75'

function req(method, url, token, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(url)
    const payload = body ? JSON.stringify(body) : null
    const opts = {
      method,
      hostname: u.hostname,
      path: u.pathname + u.search,
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        Origin: ORIGIN,
        Referer: ORIGIN + '/',
        ...(token ? { Authorization: 'Bearer ' + token } : {}),
        ...(payload ? { 'Content-Length': Buffer.byteLength(payload) } : {}),
      },
    }
    const r = https.request(opts, (res) => {
      let d = ''
      res.on('data', (c) => (d += c))
      res.on('end', () => resolve({ status: res.statusCode, body: d }))
    })
    r.on('error', reject)
    if (payload) r.write(payload)
    r.end()
  })
}

function parseErr(raw) {
  try {
    const j = JSON.parse(raw)
    return j.message || j.msg || raw.slice(0, 120)
  } catch {
    return raw.slice(0, 120)
  }
}

async function login() {
  const { body } = await req('POST', AUTH + '/auth/login', '', {
    username: USER,
    password: PASS,
    is_ai: true,
  })
  const j = JSON.parse(body)
  const token = j.token || j.data?.token
  if (!token) throw new Error('login failed: ' + body.slice(0, 300))
  return token
}

async function place(token, rule, content, bets, solo) {
  const unit = 2
  const amount = unit * bets
  const { body } = await req('POST', HTTP + '/api/web_bets/lott', token, {
    auto_type: 'platform',
    bet_contents: [
      {
        rule_id: rule,
        bet_content: content,
        amount_unit: unit,
        bets_nums: bets,
        multiple: 1,
        bet_amount: amount,
        solo,
      },
    ],
    game_id: GAME,
    currency: 3,
    bet_multiple: [],
  })
  return body
}

const probes = {
  75: [
    ['12', 1, false],
    ['12', 1, true],
    ['1,2', 1, false],
    ['1|2', 1, false],
    ['1 2', 1, false],
    ['01,02', 1, false],
    ['12,34', 2, false],
    ['12,34,56,78,90', 10, false],
    ['1,2,3,4,5', 10, false],
    ['1,2,,,', 1, false],
    ['1,,,,2', 1, false],
    ['1,2,3,4,5,6,7,8,9,0', 10, false],
    ['(0)(1)12', 1, false],
    ['0,1,12', 1, false],
    ['12,,,,', 1, false],
    ['1;2', 1, false],
    ['1-2', 1, false],
    ['万千12', 1, false],
    ['12万千', 1, false],
  ],
  76: [
    ['6', 1, true],
    ['6', 1, false],
    ['6', 3, false],
    ['3', 1, true],
    ['3,4,5', 3, false],
  ],
  77: [
    ['1,2', 1, true],
    ['1,2', 1, false],
    ['1,2,3', 3, false],
    ['12', 1, true],
    ['12', 1, false],
  ],
  79: [
    ['6', 1, false],
    ['6', 3, false],
    ['3', 1, false],
  ],
  261: [
    [',,,�?�?, 1, false],
    [',,,�?�?, 1, false],
    [',,,�?�?, 1, false],
    ['3,3,,,', 1, false],
    ['2,3,,,', 1, false],
    ['0,0,,,', 1, false],
    ['1,1,,,', 1, false],
    ['�?�?,,', 1, false],
    ['�?�?,,', 1, false],
  ],
}

const token = await login()
console.log('logged in as', USER, 'game', GAME, 'rule', RULE)
const list = probes[RULE] || probes[75]
let ok = 0
for (const [content, bets, solo] of list) {
  const raw = await place(token, RULE, content, bets, solo)
  let success = false
  try {
    const j = JSON.parse(raw)
    success = j.code === 201 || j.code === '201' || j.code === 0 || j.code === '0' || j.success === true
    if (success && j.id) success = true
    if (j.periods) success = true
  } catch {}
  const err = parseErr(raw)
  if (success || raw.includes('"periods"')) {
    ok++
    console.log('OK  ', JSON.stringify({ content, bets, solo }), raw.slice(0, 120))
  } else {
    console.log('FAIL', JSON.stringify({ content, bets, solo }), err)
  }
  await new Promise((r) => setTimeout(r, 800))
}
console.log('summary ok=' + ok + ' fail=' + (list.length - ok))
