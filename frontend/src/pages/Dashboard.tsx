import { useEffect, useState } from 'react'
import { api, ApiError } from '@/api/client'
import { useAuth } from '@/contexts/AuthContext'
import { TrendingDown, TrendingUp, Clock } from 'lucide-react'

interface Deposit {
  id:         number
  chain:      string
  amount:     number
  confirmed:  number
  created_at: string
}

interface Withdrawal {
  id:         number
  chain:      string
  amount:     number
  fee:        number
  status:     string
  created_at: string
}

const statusBadge = (status: string) => {
  switch (status) {
    case 'completed': return <span className="badge badge-success">completed</span>
    case 'failed':    return <span className="badge badge-danger">failed</span>
    default:          return <span className="badge badge-warning">pending</span>
  }
}

export default function Dashboard() {
  const { userID } = useAuth()
  const [deposits,    setDeposits]    = useState<Deposit[]>([])
  const [withdrawals, setWithdrawals] = useState<Withdrawal[]>([])
  const [error,       setError]       = useState('')
  const [loading,     setLoading]     = useState(true)

  useEffect(() => {
    if (!userID) return
    Promise.all([
      api.get<Deposit[]>('/api/v1/deposits?user_id=' + userID),
      api.get<Withdrawal[]>('/api/v1/withdrawals?user_id=' + userID),
    ])
      .then(([d, w]) => {
        setDeposits(d ?? [])
        setWithdrawals(w ?? [])
      })
      .catch(err => setError(err instanceof ApiError ? err.message : '加载失败'))
      .finally(() => setLoading(false))
  }, [userID])

  if (loading) return (
    <div className="flex items-center gap-2 text-text-muted text-sm font-mono pt-16">
      <span className="animate-pulse-gold">▋</span> 加载中…
    </div>
  )

  return (
    <div className="space-y-8">
      <div>
        <h2 className="font-display font-bold text-2xl mb-1">总览</h2>
        <p className="text-text-muted text-sm">充值与提币历史记录</p>
      </div>

      {error && (
        <p className="text-danger text-xs font-mono bg-danger/5 border border-danger/20 rounded-lg px-3 py-2">
          {error}
        </p>
      )}

      {/* ── Recent Deposits ── */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <TrendingDown size={14} className="text-success" />
          <h3 className="text-sm font-mono text-text-muted uppercase tracking-widest">
            最近充值
          </h3>
        </div>
        <div className="card overflow-hidden">
          {deposits.length === 0 ? (
            <EmptyRow message="暂无充值记录" />
          ) : (
            <table className="data-table">
              <thead>
                <tr>
                  <th>ID</th><th>链</th><th>金额</th><th>状态</th><th>时间</th>
                </tr>
              </thead>
              <tbody>
                {deposits.slice(0, 10).map(d => (
                  <tr key={d.id}>
                    <td className="text-text-faint">#{d.id}</td>
                    <td>
                      <span className="badge badge-muted">{d.chain.toUpperCase()}</span>
                    </td>
                    <td className="text-text-primary">{Number(d.amount).toFixed(8)}</td>
                    <td>
                      {d.confirmed
                        ? <span className="badge badge-success">confirmed</span>
                        : <span className="badge badge-warning">pending</span>}
                    </td>
                    <td className="text-text-faint">{d.created_at}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>

      {/* ── Recent Withdrawals ── */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <TrendingUp size={14} className="text-warning" />
          <h3 className="text-sm font-mono text-text-muted uppercase tracking-widest">
            最近提币
          </h3>
        </div>
        <div className="card overflow-hidden">
          {withdrawals.length === 0 ? (
            <EmptyRow message="暂无提币记录" />
          ) : (
            <table className="data-table">
              <thead>
                <tr>
                  <th>ID</th><th>链</th><th>金额</th><th>手续费</th><th>状态</th><th>时间</th>
                </tr>
              </thead>
              <tbody>
                {withdrawals.slice(0, 10).map(w => (
                  <tr key={w.id}>
                    <td className="text-text-faint">#{w.id}</td>
                    <td>
                      <span className="badge badge-muted">{w.chain.toUpperCase()}</span>
                    </td>
                    <td className="text-text-primary">{Number(w.amount).toFixed(8)}</td>
                    <td className="text-text-faint">{Number(w.fee).toFixed(8)}</td>
                    <td>{statusBadge(w.status)}</td>
                    <td className="text-text-faint">{w.created_at}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>
    </div>
  )
}

function EmptyRow({ message }: { message: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-10 text-text-faint text-xs font-mono">
      <Clock size={12} />
      {message}
    </div>
  )
}
