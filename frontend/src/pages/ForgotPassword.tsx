import { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, ApiError } from '@/api/client'
import ThemeSwitcher from '@/components/ThemeSwitcher'

export default function ForgotPassword() {
  const [sent,     setSent]     = useState(false)
  const [loading,  setLoading]  = useState(false)
  const [emailErr, setEmailErr] = useState('')
  const [globalErr,setGlobalErr]= useState('')
  const emailRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const email = emailRef.current?.value ?? ''
    const emailReg = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!email.trim())          { setEmailErr('请输入邮件地址'); return }
    if (!emailReg.test(email))  { setEmailErr('邮件地址格式不正确'); return }

    setEmailErr(''); setGlobalErr(''); setLoading(true)
    try {
      await api.post('/api/v1/forgot-password', { email })
      setSent(true)
    } catch (err) {
      setGlobalErr(err instanceof ApiError ? err.message : '发送失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }

  const inputStyle: React.CSSProperties = {
    width:'100%', background:'transparent', border:'none',
    borderBottom:'1px solid var(--color-border-strong)',
    borderRadius:0, padding:'14px 0',
    fontFamily:"'IBM Plex Mono',monospace", fontSize:'17px',
    color:'var(--color-text-primary)', outline:'none',
    transition:'border-color 0.2s', boxSizing:'border-box',
  }

  return (
    <>
      <style>{`
        @keyframes gridPulse { 0%,100%{opacity:0.04} 50%{opacity:0.09} }
      `}</style>

      <div style={{
        minHeight:'100vh', backgroundColor:'var(--color-base)',
        display:'flex', alignItems:'center', justifyContent:'center',
        position:'relative', overflow:'hidden', fontFamily:"'DM Sans',sans-serif",
      }}>
        <div style={{
          position:'absolute', inset:0, pointerEvents:'none',
          backgroundImage:`linear-gradient(var(--color-grid) 1px,transparent 1px),linear-gradient(90deg,var(--color-grid) 1px,transparent 1px)`,
          backgroundSize:'60px 60px', animation:'gridPulse 6s ease-in-out infinite',
        }}/>

        <div style={{position:'fixed',top:'24px',right:'28px',zIndex:50}}>
          <ThemeSwitcher/>
        </div>

        <div style={{
          width:'min(400px,90vw)', position:'relative', zIndex:1,
          padding:'clamp(32px,5vw,48px)',
        }}>

          {/* Logo */}
          <div style={{display:'flex',alignItems:'center',gap:'12px',marginBottom:'40px'}}>
            <svg width="36" height="36" viewBox="0 0 40 40" fill="none">
              <rect width="40" height="40" rx="9" fill="var(--color-accent)"/>
              <path d="M23 7L11 21h11l-5 12L30 19H19l4-12z" fill="var(--color-base)"/>
            </svg>
            <span style={{fontFamily:"'Syne',sans-serif",fontWeight:700,fontSize:'18px',letterSpacing:'0.22em',color:'var(--color-text-primary)'}}>BLITZ</span>
          </div>

          {!sent ? (
            <form onSubmit={handleSubmit} noValidate style={{display:'flex',flexDirection:'column',gap:'24px'}}>
              <div>
                <h2 style={{fontFamily:"'Syne',sans-serif",fontWeight:800,fontSize:'24px',letterSpacing:'-0.02em',color:'var(--color-text-primary)',margin:'0 0 8px 0'}}>
                  重置密码
                </h2>
                <p style={{fontFamily:"'DM Sans',sans-serif",fontSize:'14px',color:'var(--color-text-muted)',margin:0}}>
                  输入注册时的邮件地址，我们将发送重置链接。
                </p>
              </div>

              <div>
                <label style={{display:'block',fontFamily:"'Syne',sans-serif",fontWeight:600,fontSize:'15px',letterSpacing:'0.04em',color:'var(--color-text-muted)',marginBottom:'6px'}}>
                  邮件地址
                </label>
                <input
                  ref={emailRef}
                  type="email" autoFocus autoComplete="email"
                  placeholder="your@email.com"
                  onChange={() => setEmailErr('')}
                  style={{
                    ...inputStyle,
                    borderBottomColor: emailErr ? 'var(--color-danger)' : undefined,
                  }}
                  onFocus={e => { if (!emailErr) e.currentTarget.style.borderBottomColor='var(--color-accent)' }}
                  onBlur={e => { if (!emailErr) e.currentTarget.style.borderBottomColor='var(--color-border-strong)' }}
                />
                {emailErr && (
                  <span style={{fontFamily:"'IBM Plex Mono',monospace",fontSize:'12px',color:'var(--color-danger)',marginTop:'6px',display:'block'}}>
                    {emailErr}
                  </span>
                )}
              </div>

              {globalErr && (
                <span style={{fontFamily:"'DM Sans',sans-serif",fontSize:'13px',color:'var(--color-danger)'}}>
                  {globalErr}
                </span>
              )}

              <button
                type="submit" disabled={loading}
                style={{
                  display:'flex', alignItems:'center', justifyContent:'space-between',
                  width:'100%', padding:'18px 24px',
                  background:'var(--color-accent)',
                  border:'none', borderRadius:'4px',
                  fontFamily:"'Syne',sans-serif", fontWeight:700,
                  fontSize:'14px', letterSpacing:'0.06em',
                  color:'var(--color-base)', cursor: loading ? 'not-allowed' : 'pointer',
                  opacity: loading ? 0.6 : 1, transition:'all 0.2s',
                }}
              >
                <span>{loading ? '发送中…' : '发送重置链接'}</span>
                {!loading && (
                  <svg width="16" height="16" viewBox="0 0 18 18" fill="none">
                    <path d="M3.5 9h11M10 5l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                )}
              </button>

              <button type="button" onClick={() => navigate('/login')} style={{
                background:'none', border:'none', padding:0,
                fontFamily:"'DM Sans',sans-serif", fontSize:'13px',
                color:'var(--color-text-muted)', cursor:'pointer',
                textAlign:'center', transition:'color 0.15s',
              }}
                onMouseEnter={e=>(e.currentTarget.style.color='var(--color-text-primary)')}
                onMouseLeave={e=>(e.currentTarget.style.color='var(--color-text-muted)')}
              >
                ← 返回登录
              </button>
            </form>
          ) : (
            <div style={{display:'flex',flexDirection:'column',gap:'24px'}}>
              <div style={{
                width:'48px', height:'48px', borderRadius:'50%',
                background:'color-mix(in srgb,var(--color-accent) 15%,transparent)',
                display:'flex', alignItems:'center', justifyContent:'center',
              }}>
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
                  <path d="M20 4H4a2 2 0 00-2 2v12a2 2 0 002 2h16a2 2 0 002-2V6a2 2 0 00-2-2z" stroke="var(--color-accent)" strokeWidth="1.5"/>
                  <path d="M22 6l-10 7L2 6" stroke="var(--color-accent)" strokeWidth="1.5" strokeLinecap="round"/>
                </svg>
              </div>
              <div>
                <h2 style={{fontFamily:"'Syne',sans-serif",fontWeight:800,fontSize:'22px',color:'var(--color-text-primary)',margin:'0 0 8px 0'}}>
                  邮件已发送
                </h2>
                <p style={{fontFamily:"'DM Sans',sans-serif",fontSize:'14px',color:'var(--color-text-muted)',margin:0,lineHeight:1.6}}>
                  如果该邮箱已注册，重置链接将在几分钟内发送。链接有效期 30 分钟。
                </p>
              </div>
              <button type="button" onClick={() => navigate('/login')} style={{
                display:'flex', alignItems:'center', justifyContent:'center', gap:'8px',
                width:'100%', padding:'16px 24px',
                background:'transparent',
                border:'1px solid var(--color-border-strong)', borderRadius:'4px',
                fontFamily:"'DM Sans',sans-serif", fontWeight:500,
                fontSize:'14px', color:'var(--color-text-muted)', cursor:'pointer',
                transition:'all 0.15s',
              }}
                onMouseEnter={e=>{ e.currentTarget.style.color='var(--color-text-primary)'; e.currentTarget.style.borderColor='var(--color-accent)' }}
                onMouseLeave={e=>{ e.currentTarget.style.color='var(--color-text-muted)'; e.currentTarget.style.borderColor='var(--color-border-strong)' }}
              >
                返回登录
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
