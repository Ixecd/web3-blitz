import { useState } from 'react'
import { useAuth } from '@/contexts/AuthContext'
import { api, ApiError } from '@/api/client'
import { QRCodeSVG } from 'qrcode.react'
import { Copy, Check, ArrowDownToLine } from 'lucide-react'

type Chain = 'btc' | 'eth'

interface AddressResp {
  address: string
  chain:   string
  user_id: string
}

export default function Deposit() {
  const { userID }    = useAuth()
  const [chain,   setChain]   = useState<Chain>('eth')
  const [address, setAddress] = useState('')
  const [error,   setError]   = useState('')
  const [loading, setLoading] = useState(false)
  const [copied,  setCopied]  = useState(false)

  const generate = async () => {
    if (!userID) return
    setError('')
    setLoading(true)
    try {
      const res = await api.post<AddressResp>('/api/v1/address', {
        user_id: String(userID),
        chain,
      })
      setAddress(res.address)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '获取地址失败')
    } finally {
      setLoading(false)
    }
  }

  const copy = async () => {
    if (!address) return
    await navigator.clipboard.writeText(address)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <ArrowDownToLine size={16} className="text-success" />
          <h2 className="font-display font-bold text-2xl">充值</h2>
        </div>
        <p className="text-text-muted text-sm">生成专属充值地址，向该地址转账即可完成充值</p>
      </div>

      <div className="card p-6 max-w-md space-y-6">
        {/* Chain selector */}
        <div className="space-y-2">
          <label className="text-xs font-mono text-text-muted uppercase tracking-widest">
            选择链
          </label>
          <div className="flex gap-2">
            {(['eth', 'btc'] as Chain[]).map(c => (
              <button
                key={c}
                onClick={() => { setChain(c); setAddress('') }}
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

        <button
          onClick={generate}
          disabled={loading}
          className="btn-primary w-full"
        >
          {loading ? '生成中…' : '获取充值地址'}
        </button>

        {error && (
          <p className="text-danger text-xs font-mono bg-danger/5 border border-danger/20 rounded-lg px-3 py-2">
            {error}
          </p>
        )}

        {/* Address result */}
        {address && (
          <div className="space-y-4 animate-fade-up">
            {/* QR Code */}
            <div className="flex justify-center">
              <div className="p-3 bg-white rounded-xl">
                <QRCodeSVG
                  value={address}
                  size={160}
                  bgColor="#ffffff"
                  fgColor="#070711"
                  level="M"
                />
              </div>
            </div>

            {/* Address */}
            <div className="space-y-1.5">
              <label className="text-xs font-mono text-text-muted uppercase tracking-widest">
                {chain.toUpperCase()} 充值地址
              </label>
              <div className="flex items-center gap-2">
                <div className="flex-1 bg-raised border border-border rounded-lg px-3 py-2.5 font-mono text-xs text-text-primary break-all leading-relaxed">
                  {address}
                </div>
                <button
                  onClick={copy}
                  className={
                    'shrink-0 w-9 h-9 rounded-lg border flex items-center justify-center transition-all duration-150 ' +
                    (copied
                      ? 'border-success/40 bg-success/10 text-success'
                      : 'border-border bg-raised text-text-muted hover:border-gold hover:text-gold')
                  }
                >
                  {copied ? <Check size={14} /> : <Copy size={14} />}
                </button>
              </div>
            </div>

            <p className="text-xs text-text-faint font-mono text-center">
              {chain === 'btc'
                ? '等待 6 个区块确认'
                : '等待 12 个区块确认'}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
