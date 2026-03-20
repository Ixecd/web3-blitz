import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { api, ApiError } from '@/api/client'
import { ArrowUpFromLine, CheckCircle2 } from 'lucide-react'

type Chain = 'btc' | 'eth'

interface WithdrawResp {
  id:         number
  tx_id:      string
  amount:     number
  fee:        number
  status:     string
}

export default function Withdraw() {
  const { userID }  = useAuth()
  const [chain,     setChain]     = useState<Chain>('eth')
  const [toAddress, setToAddress] = useState('')
  const [amount,    setAmount]    = useState('')
  const [result,    setResult]    = useState<WithdrawResp | null>(null)
  const [error,     setError]     = useState('')
  const [loading,   setLoading]   = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!userID) return
    setError('')
    setResult(null)
    setLoading(true)
    try {
      const res = await api.post<WithdrawResp>('/api/v1/withdraw', {
        user_id:    String(userID),
        to_address: toAddress,
        amount:     parseFloat(amount),
        chain,
      })
      setResult(res)
      setToAddress('')
      setAmount('')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '提币失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <ArrowUpFromLine size={16} className="text-warning" />
          <h2 className="font-display font-bold text-2xl">提币</h2>
        </div>
        <p className="text-text-muted text-sm">向外部地址发起提币，受余额和每日限额约束</p>
      </div>

      <form onSubmit={submit} className="card p-6 max-w-md space-y-5">
        {/* Chain */}
        <div className="space-y-2">
          <label className="text-xs font-mono text-text-muted uppercase tracking-widest">链</label>
          <div className="flex gap-2">
            {(['eth', 'btc'] as Chain[]).map(c => (
              <button
                type="button"
                key={c}
                onClick={() => setChain(c)}
                className={
                  'chain-pill ' +
                  (chain === c ? 'chain-pill-active' : 'chain-pill-inactive')
                }
              >
                {c.toUpperCase()}
              </button>
            ))}
          </div>
        </div>

        {/* To address */}
        <div className="space-y-2">
          <label className="text-xs font-mono text-text-muted uppercase tracking-widest">
            收款地址
          </label>
          <input
            value={toAddress}
            onChange={e => setToAddress(e.target.value)}
            required
            placeholder={chain === 'eth' ? '0x...' : 'bc1q...'}
            className="input"
          />
        </div>

        {/* Amount */}
        <div className="space-y-2">
          <label className="text-xs font-mono text-text-muted uppercase tracking-widest">
            金额
          </label>
          <input
            type="number"
            step="0.00000001"
            min="0"
            value={amount}
            onChange={e => setAmount(e.target.value)}
            required
            placeholder="0.00000000"
            className="input"
          />
        </div>

        {error && (
          <p className="text-danger text-xs font-mono bg-danger/5 border border-danger/20 rounded-lg px-3 py-2">
            {error}
          </p>
        )}

        {/* Success */}
        {result && (
          <div className="bg-success/5 border border-success/20 rounded-lg px-4 py-3 space-y-1.5 animate-fade-up">
            <div className="flex items-center gap-2 text-success text-sm font-medium">
              <CheckCircle2 size={14} />
              提币成功
            </div>
            <p className="font-mono text-xs text-text-muted break-all">
              TxID: {result.tx_id || '—'}
            </p>
          </div>
        )}

        <button
          type="submit"
          disabled={loading}
          className="btn-primary w-full"
        >
          {loading ? '广播中…' : '确认提币'}
        </button>
      </form>
    </div>
  )
}
