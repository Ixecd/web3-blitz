import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/contexts/AuthContext'
import { ApiError } from '@/api/client'
import ThemeSwitcher from '@/components/ThemeSwitcher'

export default function Login() {
  const [emailErr,    setEmailErr]    = useState('')
  const [passwordErr, setPasswordErr] = useState('')
  const [globalErr,   setGlobalErr]   = useState('')
  const [loading,     setLoading]     = useState(false)
  const [showPwd,     setShowPwd]     = useState(false)
  const [mounted,     setMounted]     = useState(false)
  const emailRef = useRef<HTMLInputElement>(null)
  const passwordRef = useRef<HTMLInputElement>(null)
  const { login } = useAuth()
  const navigate  = useNavigate()

  useEffect(() => { setTimeout(() => setMounted(true), 50) }, [])

  const [canSubmit, setCanSubmit] = useState(false)

  const checkCanSubmit = () => {
    const e = emailRef.current?.value ?? ''
    const p = passwordRef.current?.value ?? ''
    setCanSubmit(e.trim().length > 0 && p.trim().length > 0)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const email = emailRef.current?.value ?? ''
    const password = passwordRef.current?.value ?? ''
    let valid = true
    const emailReg = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!email.trim()) { setEmailErr('请输入邮件地址'); valid = false }
    else if (!emailReg.test(email)) { setEmailErr('邮件地址格式不正确'); valid = false }
    if (!password.trim()) { setPasswordErr('请输入密码');   valid = false }
    if (!valid) return
    setGlobalErr('')
    setLoading(true)
    try {
      await login(emailRef.current?.value ?? '', passwordRef.current?.value ?? '')
      navigate('/')
    } catch (err) {
      setGlobalErr(err instanceof ApiError ? err.message : '用户名或密码错误')
    } finally {
      setLoading(false)
    }
  }

  const labelStyle: React.CSSProperties = {
    display: 'block',
    fontFamily: "'Syne', sans-serif",
    fontWeight: 600,
    fontSize: '15px',
    letterSpacing: '0.04em',
    color: 'var(--color-text-muted)',
    marginBottom: '6px',
  }

  const errStyle: React.CSSProperties = {
    fontFamily: "'IBM Plex Mono', monospace",
    fontSize: '12px',
    color: 'var(--color-danger)',
    marginTop: '8px',
    display: 'block',
  }

  return (
    <>
      <style>{`
        @keyframes float1 {
          0%   { transform: translate(0,0)          scale(1);    }
          25%  { transform: translate(-300px,150px) scale(1.3);  }
          50%  { transform: translate(200px,300px)  scale(0.8);  }
          75%  { transform: translate(-150px,-200px) scale(1.15); }
          100% { transform: translate(0,0)          scale(1);    }
        }
        @keyframes float2 {
          0%   { transform: translate(0,0)          scale(1);   }
          33%  { transform: translate(-400px,-150px) scale(1.35);}
          66%  { transform: translate(250px,200px)  scale(0.75);}
          100% { transform: translate(0,0)          scale(1);   }
        }
        @keyframes float3 {
          0%   { transform: translate(0,0)          scale(1);   }
          40%  { transform: translate(350px,-200px) scale(1.25);}
          80%  { transform: translate(-200px,250px) scale(0.85);}
          100% { transform: translate(0,0)          scale(1);   }
        }
        @keyframes gridPulse {
          0%,100% { opacity:0.04; }
          50%     { opacity:0.09; }
        }
        .l-input {
          width:100%; background:transparent; border:none;
          border-bottom:1px solid var(--color-border-strong);
          border-radius:0; padding:14px 0;
          font-family:'IBM Plex Mono',monospace; font-size:17px;
          color:var(--color-text-primary); outline:none;
          transition:border-color 0.2s; box-sizing:border-box;
        }
        .l-input::placeholder { color:var(--color-text-faint); }
        .l-input:-webkit-autofill,
        .l-input:-webkit-autofill:hover,
        .l-input:-webkit-autofill:focus {
          -webkit-box-shadow: 0 0 0px 1000px var(--color-raised) inset !important;
          -webkit-text-fill-color: var(--color-text-primary) !important;
          transition: background-color 5000s ease-in-out 0s;
          caret-color: var(--color-text-primary);
        }
        .l-input:focus  { border-bottom-color:var(--color-accent); }
        .l-input.filled { border-bottom-color:var(--color-accent); }
        .l-input.err    { border-bottom-color:var(--color-danger)!important; }
        .l-alt {
          width:100%; display:flex; align-items:center;
          justify-content:center; gap:10px; padding:15px 20px;
          background:transparent;
          border:1px solid var(--color-border-strong); border-radius:4px;
          font-family:'DM Sans',sans-serif; font-weight:500; font-size:14px;
          color:var(--color-text-muted); cursor:pointer;
          transition:border-color 0.15s,color 0.15s,background 0.15s;
          letter-spacing:0.02em;
        }
        .l-alt:hover {
          background:color-mix(in srgb, var(--color-raised) 50%, transparent);
          color:var(--color-text-primary);
        }
      `}</style>

      <div style={{
        minHeight:'100vh', backgroundColor:'var(--color-base)',
        display:'flex', position:'relative', overflow:'hidden',
        fontFamily:"'DM Sans',sans-serif",
      }}>

        {/* ── 动态光晕 ── */}
        <div style={{position:'absolute',inset:0,pointerEvents:'none',overflow:'hidden'}}>
          <div style={{
            position:'absolute', width:'800px', height:'700px',
            borderRadius:'50%', top:'-20%', right:'-8%',
            background:`radial-gradient(ellipse,
              color-mix(in srgb,var(--color-accent) 35%,transparent) 0%,
              transparent 65%)`,
            animation:'float1 4s ease-in-out infinite',
          }}/>
          <div style={{
            position:'absolute', width:'600px', height:'550px',
            borderRadius:'50%', bottom:'5%', right:'15%',
            background:`radial-gradient(ellipse,
              color-mix(in srgb,var(--color-accent) 28%,transparent) 0%,
              transparent 60%)`,
            animation:'float2 5s ease-in-out infinite',
          }}/>
          <div style={{
            position:'absolute', width:'450px', height:'400px',
            borderRadius:'50%', top:'20%', left:'-10%',
            background:`radial-gradient(ellipse,
              color-mix(in srgb,var(--color-accent) 20%,transparent) 0%,
              transparent 55%)`,
            animation:'float3 6s ease-in-out infinite',
          }}/>
        </div>

        {/* ── 网格 ── */}
        <div style={{
          position:'absolute', inset:0, pointerEvents:'none',
          backgroundImage:`
            linear-gradient(var(--color-grid) 1px,transparent 1px),
            linear-gradient(90deg,var(--color-grid) 1px,transparent 1px)`,
          backgroundSize:'60px 60px',
          animation:'gridPulse 6s ease-in-out infinite',
        }}/>

        {/* 主题切换 */}
        <div style={{position:'fixed',top:'24px',right:'28px',zIndex:50}}>
          <ThemeSwitcher/>
        </div>

        {/* ── 左侧 ── */}
        <div className="login-left" style={{
          flex:'1', display:'flex', flexDirection:'column',
          padding:'clamp(32px,5vh,56px) clamp(32px,5vw,64px)', position:'relative', zIndex:1,
        }}>
          {/* Logo */}
          <div style={{display:'flex',alignItems:'center',gap:'16px'}}>
            <svg width="48" height="48" viewBox="0 0 40 40" fill="none">
              <rect width="40" height="40" rx="9" fill="var(--color-accent)"/>
              <path d="M23 7L11 21h11l-5 12L30 19H19l4-12z" fill="var(--color-base)"/>
            </svg>
            <span style={{
              fontFamily:"'Syne',sans-serif", fontWeight:700,
              fontSize:'22px', letterSpacing:'0.22em',
              color:'var(--color-text-primary)',
            }}>BLITZ</span>
          </div>

          {/* 标题 — 右移 + 更低 */}
          <div style={{
            flex:1, display:'flex',
            alignItems:'flex-end', justifyContent:'flex-start',
          }}>
            <h1 style={{
              position:'absolute',
              left:'64px',
              bottom:'64px',
              lineHeight:1,
              fontFamily:"'Syne',sans-serif", fontWeight:800,
              fontSize:'clamp(64px,7vw,104px)',
              color:'var(--color-text-primary)', margin:0,
              letterSpacing:'-0.025em',
              opacity: mounted ? 1 : 0,
              transform: mounted ? 'translateY(0)' : 'translateY(24px)',
              transition:'opacity 0.9s cubic-bezier(0.16,1,0.3,1), transform 0.9s cubic-bezier(0.16,1,0.3,1)',
            }}>
              欢迎<br/>回来。
            </h1>
          </div>
        </div>

        {/* ── 右侧 — 完全透明 ── */}
        <div style={{
          flex:'0 0 auto', width:'min(440px, 36vw)', marginRight:'2vw',
          display:'flex', flexDirection:'column', justifyContent:'center', overflowY:'auto',
          padding:'clamp(48px,8vh,96px) clamp(32px,4vw,56px)', position:'relative', zIndex:1,
        }}>
          <form onSubmit={handleSubmit} noValidate style={{
            display:'flex', flexDirection:'column', gap:'20px',
          }}>

            {/* 标题 */}
            <h2 style={{
              fontFamily:"'Syne',sans-serif", fontWeight:800,
              fontSize:'26px', letterSpacing:'-0.02em',
              color:'var(--color-text-primary)', margin:'0 0 8px 0',
            }}>登录您的账户</h2>

            {/* 邮件地址 */}
            <div>
              <label style={labelStyle}>邮件地址</label>
              <input
                ref={emailRef}
                type="email" autoFocus autoComplete="email"
                placeholder="your@email.com"
                onChange={() => { setEmailErr(''); checkCanSubmit() }}
                onInput={() => { setEmailErr(''); checkCanSubmit() }}
                className={`l-input${emailErr?' err':''}`}
              />
            </div>

            {/* 密码 */}
            <div style={{position:'relative'}}>
              <div style={{display:'flex',alignItems:'center',justifyContent:'space-between',marginBottom:'6px'}}>
                <label style={{...labelStyle, marginBottom:0}}>密码</label>
                <button type="button" style={{
                  background:'none', border:'none', padding:0,
                  fontFamily:"'DM Sans',sans-serif", fontSize:'13px',
                  color:'var(--color-accent)', cursor:'pointer',
                  transition:'opacity 0.15s',
                }}
                onMouseEnter={e=>(e.currentTarget.style.opacity='0.6')}
                onMouseLeave={e=>(e.currentTarget.style.opacity='1')}
                >忘记密码？</button>
              </div>
              <input
                ref={passwordRef}
                type={showPwd ? "text" : "password"} autoComplete="current-password"
                placeholder="••••••••••"
                onChange={() => { setPasswordErr(''); checkCanSubmit() }}
                onInput={() => { setPasswordErr(''); checkCanSubmit() }}
                className={`l-input${passwordErr?' err':''}`}
              />
              {/* 查看密码按钮 */}
              <button
                type="button"
                onClick={() => setShowPwd(p => !p)}
                style={{
                  position:'absolute', right:0, bottom:'14px',
                  background:'none', border:'none', padding:'4px',
                  cursor:'pointer', color:'var(--color-text-faint)',
                  transition:'color 0.15s', lineHeight:1,
                }}
                onMouseEnter={e=>(e.currentTarget.style.color='var(--color-text-muted)')}
                onMouseLeave={e=>(e.currentTarget.style.color='var(--color-text-faint)')}
              >
                {showPwd ? (
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                    <path d="M2 2l12 12M6.5 6.5A2 2 0 0110 10M4 4.5C2.5 5.8 1.5 7 1.5 8c0 1.5 3 5 6.5 5 1.2 0 2.3-.3 3.2-.8M7 3.1C7.3 3 7.7 3 8 3c3.5 0 6.5 3.5 6.5 5 0 .7-.4 1.5-1 2.3"
                      stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                  </svg>
                ) : (
                  <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                    <path d="M1.5 8C1.5 8 4.5 3 8 3s6.5 5 6.5 5-3 5-6.5 5S1.5 8 1.5 8z"
                      stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                    <circle cx="8" cy="8" r="2" stroke="currentColor" strokeWidth="1.2"/>
                  </svg>
                )}
              </button>
              {passwordErr && <span style={errStyle}>{passwordErr}</span>}
            </div>

            {/* 记住我 */}
            <label style={{
              display:'flex', alignItems:'center', gap:'10px',
              cursor:'pointer', userSelect:'none',
            }}>
              <div style={{
                width:'18px', height:'18px', borderRadius:'4px',
                border:'1px solid var(--color-border-strong)',
                backgroundColor:'transparent',
                display:'flex', alignItems:'center', justifyContent:'center',
                flexShrink:0,
              }}>
                <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                  <path d="M1 4l3 3 5-6" stroke="var(--color-accent)" strokeWidth="1.5"
                    strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              </div>
              <span style={{
                fontFamily:"'DM Sans',sans-serif", fontSize:'13px',
                color:'var(--color-text-muted)',
              }}>在该设备上记住我</span>
            </label>

            {/* 错误提示 — 紧凑单行 */}
            <div style={{height:'16px', display:'flex', alignItems:'center', gap:'5px', marginTop:'-4px'}}>
              {(emailErr || globalErr) && (<>
                <svg width="11" height="11" viewBox="0 0 14 14" fill="none" style={{flexShrink:0}}>
                  <path d="M7 1L13 12H1L7 1z" stroke="var(--color-danger)" strokeWidth="1.4"
                    strokeLinejoin="round"/>
                  <path d="M7 5.5v3M7 10h.01" stroke="var(--color-danger)" strokeWidth="1.4"
                    strokeLinecap="round"/>
                </svg>
                <span style={{
                  fontFamily:"'DM Sans',monospace",
                  fontSize:'12px', color:'var(--color-danger)',
                  opacity:0.9,
                }}>{emailErr || globalErr}</span>
              </>)}
            </div>

            {/* 进入系统 */}
            <button
              type="submit"
              disabled={!canSubmit || loading}
              style={{
                display:'flex', alignItems:'center', justifyContent:'space-between',
                width:'100%', padding:'20px 24px',
                background: canSubmit && !loading
                  ? 'var(--color-accent)'
                  : 'color-mix(in srgb,var(--color-raised) 55%,transparent)',
                border:'1px solid ' + (canSubmit && !loading
                  ? 'transparent' : 'var(--color-border)'),
                borderRadius:'4px',
                fontFamily:"'Syne',sans-serif", fontWeight:700,
                fontSize:'15px', letterSpacing:'0.06em',
                color: canSubmit && !loading ? 'var(--color-base)' : 'var(--color-text-faint)',
                cursor: canSubmit && !loading ? 'pointer' : 'not-allowed',
                transition:'all 0.2s',
              }}
              onMouseEnter={e => {
                if (canSubmit && !loading)
                  (e.currentTarget as HTMLButtonElement).style.filter='brightness(1.12)'
              }}
              onMouseLeave={e => {
                (e.currentTarget as HTMLButtonElement).style.filter=''
              }}
            >
              <span>{loading ? '验证中…' : '进入系统'}</span>
              {!loading && (
                <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
                  <path d="M3.5 9h11M10 5l4 4-4 4"
                    stroke="currentColor" strokeWidth="1.5"
                    strokeLinecap="round" strokeLinejoin="round"/>
                </svg>
              )}
            </button>

            {/* ── 或 分割线 ── */}
            <div style={{display:'flex',alignItems:'center',gap:'16px',margin:'0'}}>
              <div style={{flex:1,height:'1px',backgroundColor:'var(--color-border)'}}/>
              <span style={{
                fontFamily:"'IBM Plex Mono',monospace",
                fontSize:'11px', color:'var(--color-text-faint)', letterSpacing:'0.12em',
              }}>或</span>
              <div style={{flex:1,height:'1px',backgroundColor:'var(--color-border)'}}/>
            </div>

            {/* ── 其他选项 ── */}
            <div style={{display:'flex',flexDirection:'column',gap:'8px'}}>
              <button type="button" className="l-alt">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <rect x="1" y="1" width="5" height="5" rx="1" stroke="currentColor" strokeWidth="1.2"/>
                  <rect x="3" y="3" width="1" height="1" fill="currentColor"/>
                  <rect x="10" y="1" width="5" height="5" rx="1" stroke="currentColor" strokeWidth="1.2"/>
                  <rect x="12" y="3" width="1" height="1" fill="currentColor"/>
                  <rect x="1" y="10" width="5" height="5" rx="1" stroke="currentColor" strokeWidth="1.2"/>
                  <rect x="3" y="12" width="1" height="1" fill="currentColor"/>
                  <rect x="10" y="10" width="2" height="2" fill="currentColor"/>
                  <rect x="13" y="10" width="2" height="2" fill="currentColor"/>
                  <rect x="10" y="13" width="2" height="2" fill="currentColor"/>
                  <rect x="13" y="13" width="2" height="2" fill="currentColor"/>
                </svg>
                扫码登录
              </button>

              <button type="button" className="l-alt">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                  <circle cx="8" cy="5" r="3" stroke="currentColor" strokeWidth="1.2"/>
                  <path d="M2 13c0-2.2 2.7-4 6-4s6 1.8 6 4"
                    stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                  <path d="M12 1.5v4M10 3.5h4"
                    stroke="currentColor" strokeWidth="1.2" strokeLinecap="round"/>
                </svg>
                注册新账号
              </button>
            </div>

          </form>

          <p style={{
            position:'absolute',
            bottom:'64px',
            left:0, right:0,
            fontFamily:"'IBM Plex Mono',monospace",
            fontSize:'11px', color:'var(--color-text-faint)',
            letterSpacing:'0.08em', margin:0, textAlign:'center',
          }}>
            web3-blitz wallet infrastructure
          </p>
        </div>

      </div>
    </>
  )
}
