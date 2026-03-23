import { useState, useRef, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { api, ApiError } from '@/api/client'
import ThemeSwitcher from '@/components/ThemeSwitcher'

export default function ResetPassword() {
  const [searchParams]                  = useSearchParams()
  const [done,        setDone]          = useState(false)
  const [loading,     setLoading]       = useState(false)
  const [showPwd,     setShowPwd]       = useState(false)
  const [passwordErr, setPasswordErr]   = useState('')
  const [confirmErr,  setConfirmErr]    = useState('')
  const [globalErr,   setGlobalErr]     = useState('')
  const [tokenErr,    setTokenErr]      = useState('')
  const passwordRef = useRef<HTMLInputElement>(null)
  const confirmRef  = useRef<HTMLInputElement>(null)
  const navigate    = useNavigate()

  const token = searchParams.get('token') ?? ''

  useEffect(() => {
    if (!token) setTokenErr('重置链接无效，请重新申请')
  }, [token])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const password = passwordRef.current?.value ?? ''
    const confirm  = confirmRef.current?.value  ?? ''
    let valid = true

    if (password.length < 8) { setPasswordErr('密码长度不能少于 8 位'); valid = false }
    if (confirm !== password) { setConfirmErr('两次密码不一致'); valid = false }
    if (!valid) return

    setGlobalErr(''); setLoading(true)
    try {
      await api.post('/api/v1/reset-password', { token, password })
      setDone(true)
    } catch (err) {
      setGlobalErr(err instanceof ApiError ? err.message : '重置失败，链接可能已失效')
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
  const labelStyle: React.CSSProperties = {
    display:'block', fontFamily:"'Syne',sans-serif", fontWeight:600,
    fontSize:'15px', letterSpacing:'0.04em',
    color:'var(--color-text-muted)', marginBottom:'6px',
  }
  const errStyle: React.CSSProperties = {
    fontFamily:"'IBM Plex Mono',monospace", fontSize:'12px',
    color:'var(--color-danger)', marginTop:'6px', display:'block',
  }

  return (
    <>
      <style>{`@keyframes gridPulse{0%,100%{opacity:0.04}50%{opacity:0.09}}`}</style>

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

        <div style={{width:'min(400px,90vw)',position:'relative',zIndex:1,padding:'clamp(32px,5vw,48px)'}}>

          {/* Logo */}
          <div style={{display:'flex',alignItems:'center',gap:'12px',marginBottom:'40px'}}>
            <svg width="36" height="36" viewBox="0 0 40 40" fill="none">
              <rect width="40" height="40" rx="9" fill="var(--color-accent)"/>
              <path d="M23 7L11 21h11l-5 12L30 19H19l4-12z" fill="var(--color-base)"/>
            </svg>
            <span style={{fontFamily:"'Syne',sans-serif",fontWeight:700,fontSize:'18px',letterSpacing:'0.22em',color:'var(--color-text-primary)'}}>BLITZ</span>
          </div>

          {/* token 无效 */}
          {tokenErr ? (
            <div style={{display:'flex',flexDirection:'column',gap:'20px'}}>
              <h2 style={{fontFamily:"'Syne',sans-serif",fontWeight:800,fontSize:'22px',color:'var(--color-danger)',margin:0}}>链接无效</h2>
              <p style={{fontFamily:"'DM Sans',sans-serif",fontSize:'14px',color:'var(--color-text-muted)',margin:0}}>{tokenErr}</p>
              <button type="button" onClick={() => navigate('/forgot-password')} style={{
                padding:'16px 24px', background:'var(--color-accent)', border:'none', borderRadius:'4px',
                fontFamily:"'Syne',sans-serif", fontWeight:700, fontSize:'14px',
                color:'var(--color-base)', cursor:'pointer',
              }}>重新申请重置</button>
            </div>
          ) : !done ? (
            <form onSubmit={handleSubmit} noValidate style={{display:'flex',flexDirection:'column',gap:'24px'}}>
              <div>
                <h2 style={{fontFamily:"'Syne',sans-serif",fontWeight:800,fontSize:'24px',letterSpacing:'-0.02em',color:'var(--color-text-primary)',margin:'0 0 8px 0'}}>
                  设置新密码
                </h2>
                <p style={{fontFamily:"'DM Sans',sans-serif",fontSize:'14px',color:'var(--color-text-muted)',margin:0}}>
                  请输入您的新密码，长度至少 8 位。
                </p>
              </div>

              {/* 新密码 */}
              <div style={{position:'relative'}}>
                <label style={labelStyle}>新密码</label>
                <input
                  ref={passwordRef}
                  type={showPwd ? 'text' : 'password'}
                  autoComplete="new-password" autoFocus
                  placeholder="••••••••••"
                  onChange={() => setPasswordErr('')}
                  style={{...inputStyle, borderBottomColor: passwordErr ? 'var(--color-danger)' : undefined}}
                  onFocus={e => { if (!passwordErr) e.currentTarget.style.borderBottomColor='var(--color-accent)' }}
                  onBlur={e  => { if (!passwordErr) e.currentTarget.style.borderBottomColor='var(--color-border-strong)' }}
                />
                <button type="button" onClick={() => setShowPwd(p => !p)} style={{
                  position:'absolute', right:0, bottom: passwordErr ? '28px' : '14px',
                  background:'none', border:'none', padding:'4px',
                  cursor:'pointer', color:'var(--color-text-faint)', lineHeight:1,
                }}>
                  {showPwd ? (
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                      <path d="M2 2l12 12M6.5 6.5A2 2 0 0110 10M4 4.5C2.5 5.8 1.5 7 1.5 8c0 1.5 3 5 6.5 5 1.2 0 2.3-.3 3.2-.8M7 3.1C7.3 3 7.7 3 8 3c3.5 0 6.5 3.5 6.5 5 0 .7-.4 1.5-1 2.3"
                        stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                    </svg>
                  ) : (
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                      <path d="M1.5 8C1.5 8 4.5 3 8 3s6.5 5 6.5 5-3 5-6.5 5S1.5 8 1.5 8z" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                      <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.2"/>
                    </svg>
                  )}
                </button>
                {passwordErr && <span style={errStyle}>{passwordErr}</span>}
              </div>

              {/* 确认密码 */}
              <div>
                <label style={labelStyle}>确认新密码</label>
                <input
                  ref={confirmRef}
                  type={showPwd ? 'text' : 'password'}
                  autoComplete="new-password"
                  placeholder="••••••••••"
                  onChange={() => setConfirmErr('')}
                  style={{...inputStyle, borderBottomColor: confirmErr ? 'var(--color-danger)' : undefined}}
                  onFocus={e => { if (!confirmErr) e.currentTarget.style.borderBottomColor='var(--color-accent)' }}
                  onBlur={e  => { if (!confirmErr) e.currentTarget.style.borderBottomColor='var(--color-border-strong)' }}
                />
                {confirmErr && <span style={errStyle}>{confirmErr}</span>}
              </div>

              {globalErr && (
                <span style={{fontFamily:"'DM Sans',sans-serif",fontSize:'13px',color:'var(--color-danger)'}}>{globalErr}</span>
              )}

              <button type="submit" disabled={loading} style={{
                display:'flex', alignItems:'center', justifyContent:'space-between',
                width:'100%', padding:'18px 24px',
                background:'var(--color-accent)', border:'none', borderRadius:'4px',
                fontFamily:"'Syne',sans-serif", fontWeight:700,
                fontSize:'14px', letterSpacing:'0.06em',
                color:'var(--color-base)', cursor: loading ? 'not-allowed' : 'pointer',
                opacity: loading ? 0.6 : 1, transition:'all 0.2s',
              }}>
                <span>{loading ? '重置中…' : '确认重置'}</span>
                {!loading && (
                  <svg width="16" height="16" viewBox="0 0 18 18" fill="none">
                    <path d="M3.5 9h11M10 5l4 4-4 4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                  </svg>
                )}
              </button>
            </form>
          ) : (
            /* 重置成功 */
            <div style={{display:'flex',flexDirection:'column',gap:'24px'}}>
              <div style={{
                width:'48px', height:'48px', borderRadius:'50%',
                background:'color-mix(in srgb,var(--color-accent) 15%,transparent)',
                display:'flex', alignItems:'center', justifyContent:'center',
              }}>
                <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
                  <path d="M20 12a8 8 0 11-16 0 8 8 0 0116 0z" stroke="var(--color-accent)" strokeWidth="1.5"/>
                  <path d="M8 12l3 3 5-5" stroke="var(--color-accent)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <div>
                <h2 style={{fontFamily:"'Syne',sans-serif",fontWeight:800,fontSize:'22px',color:'var(--color-text-primary)',margin:'0 0 8px 0'}}>
                  密码已重置
                </h2>
                <p style={{fontFamily:"'DM Sans',sans-serif",fontSize:'14px',color:'var(--color-text-muted)',margin:0}}>
                  您的密码已成功更新，请使用新密码登录。
                </p>
              </div>
              <button type="button" onClick={() => navigate('/login')} style={{
                padding:'16px 24px', background:'var(--color-accent)',
                border:'none', borderRadius:'4px',
                fontFamily:"'Syne',sans-serif", fontWeight:700,
                fontSize:'14px', letterSpacing:'0.06em',
                color:'var(--color-base)', cursor:'pointer', transition:'all 0.2s',
              }}
                onMouseEnter={e=>(e.currentTarget.style.filter='brightness(1.12)')}
                onMouseLeave={e=>(e.currentTarget.style.filter='')}
              >
                前往登录
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
