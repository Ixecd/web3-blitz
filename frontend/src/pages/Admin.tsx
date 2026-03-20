import { useEffect, useState } from 'react'
import { api, ApiError } from '@/api/client'
import { ShieldCheck, RefreshCw } from 'lucide-react'

interface User {
  id:         number
  username:   string
  level:      number
  created_at: string
}

interface WithdrawalLimit {
  level:      number
  level_name: string
  btc_daily:  string
  eth_daily:  string
}

const LEVEL: Record<number, { label: string; cls: string }> = {
  0: { label: '普通', cls: 'badge-muted'    },
  1: { label: '白银', cls: 'badge-muted'    },
  2: { label: '黄金', cls: 'badge-warning'  },
  3: { label: '钻石', cls: 'badge-success'  },
}

export default function Admin() {
  const [users,   setUsers]   = useState<User[]>([])
  const [limits,  setLimits]  = useState<WithdrawalLimit[]>([])
  const [error,   setError]   = useState('')
  const [loading, setLoading] = useState(false)

  const fetchAll = () => {
    setLoading(true)
    Promise.all([
      api.get<User[]>('/api/v1/users'),
      api.get<WithdrawalLimit[]>('/api/v1/withdrawal-limits'),
    ])
      .then(([u, l]) => { setUsers(u ?? []); setLimits(l ?? []) })
      .catch(err => setError(err instanceof ApiError ? err.message : '加载失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchAll() }, [])

  const upgradeUser = async (userID: number, level: number) => {
    try {
      await api.post('/api/v1/users/upgrade', { user_id: userID, level })
      fetchAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '操作失败')
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <ShieldCheck size={16} className="text-gold" />
            <h2 className="font-display font-bold text-2xl">管理后台</h2>
          </div>
          <p className="text-text-muted text-sm">用户管理与提币限额配置</p>
        </div>
        <button
          onClick={fetchAll}
          disabled={loading}
          className="btn-ghost flex items-center gap-2"
        >
          <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
          刷新
        </button>
      </div>

      {error && (
        <p className="text-danger text-xs font-mono bg-danger/5 border border-danger/20 rounded-lg px-3 py-2">
          {error}
        </p>
      )}

      {/* ── Users ── */}
      <section>
        <h3 className="text-xs font-mono text-text-muted uppercase tracking-widest mb-3">
          用户列表
        </h3>
        <div className="card overflow-hidden">
          <table className="data-table">
            <thead>
              <tr>
                <th>ID</th>
                <th>用户名</th>
                <th>等级</th>
                <th>注册时间</th>
                <th>升级</th>
              </tr>
            </thead>
            <tbody>
              {users.map(u => (
                <tr key={u.id}>
                  <td className="text-text-faint">#{u.id}</td>
                  <td className="text-text-primary">{u.username}</td>
                  <td>
                    <span className={'badge ' + (LEVEL[u.level]?.cls ?? 'badge-muted')}>
                      {LEVEL[u.level]?.label ?? u.level}
                    </span>
                  </td>
                  <td className="text-text-faint">{u.created_at}</td>
                  <td>
                    <select
                      value={u.level}
                      onChange={e => upgradeUser(u.id, Number(e.target.value))}
                      className="bg-raised border border-border rounded px-2 py-1 text-xs font-mono text-text-primary outline-none focus:border-gold transition-colors"
                    >
                      {[0, 1, 2, 3].map(l => (
                        <option key={l} value={l}>{LEVEL[l]?.label}</option>
                      ))}
                    </select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* ── Limits ── */}
      <section>
        <h3 className="text-xs font-mono text-text-muted uppercase tracking-widest mb-3">
          提币限额配置
        </h3>
        <div className="card overflow-hidden">
          <table className="data-table">
            <thead>
              <tr>
                <th>等级</th>
                <th>BTC 日限额</th>
                <th>ETH 日限额</th>
              </tr>
            </thead>
            <tbody>
              {limits.map(l => (
                <tr key={l.level}>
                  <td>
                    <span className={'badge ' + (LEVEL[l.level]?.cls ?? 'badge-muted')}>
                      {l.level_name}
                    </span>
                  </td>
                  <td className="text-text-primary">{l.btc_daily} BTC</td>
                  <td className="text-text-primary">{l.eth_daily} ETH</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  )
}
