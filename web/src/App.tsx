import React, { useEffect, useRef, useState, useCallback } from 'react'

type Point = { x: number; y: number }

type Stroke = {
  id?: number
  points: Point[]
  color: string
  width: number
  clientId: string
  startedAtUnixMs: number
}

type MsgStroke = { type: 'stroke'; stroke: Stroke }

type MsgDelete = { type: 'delete'; delete: number }

type Message = MsgStroke | MsgDelete

type User = { id: number; email: string }

type Candidate = { text: string; score: number }

const randomId = () => Math.random().toString(36).slice(2)

const apiBase = ''

async function apiFetch(path: string, init?: RequestInit) {
  const res = await fetch(`${apiBase}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json()
}

function useWebSocket(url: string, onMsg: (msg: Message) => void) {
  const wsRef = useRef<WebSocket | null>(null)
  const [ready, setReady] = useState(false)
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    let alive = true
    let reconnectTimer: number | undefined

    const connect = () => {
      if (!alive || !url || url.startsWith('ws://invalid')) return
      const ws = new WebSocket(url)
      wsRef.current = ws
      ws.onopen = () => setReady(true)
      ws.onclose = () => {
        setReady(false)
        if (!alive) return
        reconnectTimer = window.setTimeout(() => setAttempt((a) => a + 1), Math.min(1000 * (attempt + 1), 5000))
      }
      ws.onerror = () => { try { ws.close() } catch {} }
      ws.onmessage = (ev) => {
        try {
          const data = JSON.parse(ev.data)
          if (data && (data.type === 'stroke' || data.type === 'delete')) onMsg(data)
        } catch {}
      }
    }

    connect()
    return () => {
      alive = false
      if (reconnectTimer) window.clearTimeout(reconnectTimer)
      if (wsRef.current) { try { wsRef.current.close() } catch {} }
    }
  }, [url, attempt, onMsg])

  const send = useCallback((msg: Message) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  const close = useCallback(() => {
    if (wsRef.current) { try { wsRef.current.close() } catch {} }
    setReady(false)
  }, [])

  return { send, ready, close }
}

type Tool = 'pencil' | 'eraser'

export const App: React.FC = () => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [color, setColor] = useState('#1d4ed8')
  const [width, setWidth] = useState(4)
  const [tool, setTool] = useState<Tool>('pencil')
  const clientIdRef = useRef<string>(randomId())
  const [strokes, setStrokes] = useState<Stroke[]>([])
  const [user, setUser] = useState<User | null>(null)
  const [authErr, setAuthErr] = useState<string | null>(null)
  const [candidates, setCandidates] = useState<Candidate[] | null>(null)

  const isDev = location.port === '5173'
  const wsUrl = isDev
    ? `ws://${location.hostname}:5173/ws`
    : `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`

  const handleIncoming = useCallback((m: Message) => {
    if (m.type === 'stroke') {
      setStrokes((s) => {
        // If this is our own stroke (same clientId), update the existing one with the ID
        const existingIndex = s.findIndex(st => 
          st.clientId === m.stroke.clientId && 
          st.startedAtUnixMs === m.stroke.startedAtUnixMs &&
          !st.id // Only update if it doesn't already have an ID
        )
        
        if (existingIndex >= 0) {
          // Update existing stroke with the ID from server
          const updated = [...s]
          updated[existingIndex] = { ...updated[existingIndex], id: m.stroke.id }
          return updated
        } else {
          // Add new stroke (from another user)
          return [...s, m.stroke]
        }
      })
    } else if (m.type === 'delete') {
      const id = m.delete
      setStrokes((s) => s.filter((st) => st.id !== id))
    }
  }, [])
  const { send, ready, close } = useWebSocket(user ? wsUrl : 'ws://invalid', handleIncoming)

  useEffect(() => { apiFetch('/api/me').then((u) => setUser(u)).catch(() => setUser(null)) }, [])
  useEffect(() => { if (!user) return; apiFetch('/api/strokes').then((list: Stroke[]) => setStrokes(list)).catch(() => setStrokes([])) }, [user])

  useEffect(() => {
    const cvs = canvasRef.current
    if (!cvs) return
    const ctx = cvs.getContext('2d', { willReadFrequently: true })!

    const drawStroke = (s: Stroke) => {
      if (s.points.length < 2) return
      ctx.strokeStyle = s.color
      ctx.lineWidth = s.width
      ctx.lineCap = 'round'
      ctx.lineJoin = 'round'
      ctx.beginPath()
      ctx.moveTo(s.points[0].x, s.points[0].y)
      for (let i = 1; i < s.points.length; i++) ctx.lineTo(s.points[i].x, s.points[i].y)
      ctx.stroke()
    }

    ctx.clearRect(0, 0, cvs.width, cvs.height)
    for (const s of strokes) drawStroke(s)
  }, [strokes])

  useEffect(() => {
    const cvs = canvasRef.current
    if (!cvs) return
    const ctx = cvs.getContext('2d', { willReadFrequently: true })!

    const resize = () => {
      // Set fixed canvas size to match recognition canvas (300x300)
      const img = ctx.getImageData(0, 0, cvs.width, cvs.height)
      cvs.width = 300
      cvs.height = 300
      ctx.scale(1, 1) // No scaling needed for fixed size
      ctx.putImageData(img, 0, 0)
    }
    resize()
    window.addEventListener('resize', resize)
    return () => window.removeEventListener('resize', resize)
  }, [])

  const hitTest = (p: Point, s: Stroke): boolean => {
    // check distance from point to each segment less than threshold
    const threshold = Math.max(6, s.width + 4)
    for (let i = 0; i < s.points.length - 1; i++) {
      const a = s.points[i], b = s.points[i+1]
      const dx = b.x - a.x, dy = b.y - a.y
      const len2 = dx*dx + dy*dy
      if (len2 === 0) continue
      const t = Math.max(0, Math.min(1, ((p.x - a.x)*dx + (p.y - a.y)*dy) / len2))
      const cx = a.x + t*dx, cy = a.y + t*dy
      const dist = Math.hypot(p.x - cx, p.y - cy)
      if (dist <= threshold) return true
    }
    return false
  }

  useEffect(() => {
    const cvs = canvasRef.current
    if (!cvs || !user) return
    const ctx = cvs.getContext('2d', { willReadFrequently: true })!
    ctx.lineCap = 'round'
    ctx.lineJoin = 'round'

    const rect = () => cvs.getBoundingClientRect()

    let drawing = false
    let points: Point[] = []

    const toPoint = (e: PointerEvent): Point => ({ x: (e.clientX - rect().left), y: (e.clientY - rect().top) })

    const onDown = (e: PointerEvent) => {
      const p = toPoint(e)
      if (tool === 'eraser') {
        // find last stroke under pointer and delete it
        const target = [...strokes].reverse().find((s) => hitTest(p, s))
        if (target && typeof target.id === 'number') {
          setStrokes((s) => s.filter((st) => st.id !== target.id))
          send({ type: 'delete', delete: target.id })
          apiFetch(`/api/strokes/delete?id=${target.id}`, { method: 'POST' }).catch(() => {})
        }
        return
      }
      drawing = true
      points = [p]
      ctx.strokeStyle = color
      ctx.lineWidth = width
      ctx.beginPath()
      ctx.moveTo(p.x, p.y)
      cvs.setPointerCapture(e.pointerId)
    }

    const onMove = (e: PointerEvent) => {
      if (!drawing || tool !== 'pencil') return
      const p = toPoint(e)
      points.push(p)
      ctx.lineTo(p.x, p.y)
      ctx.stroke()
    }

    const onUp = () => {
      if (!drawing || tool !== 'pencil') { drawing = false; points = []; return }
      drawing = false
      if (points.length >= 2) {
        const stroke: Stroke = { points: [...points], color, width, clientId: clientIdRef.current, startedAtUnixMs: Date.now() }
        setStrokes((s) => [...s, stroke])
        send({ type: 'stroke', stroke })
      }
      points = []
      ctx.closePath()
    }

    cvs.addEventListener('pointerdown', onDown)
    cvs.addEventListener('pointermove', onMove)
    cvs.addEventListener('pointerup', onUp)
    cvs.addEventListener('pointercancel', onUp)
    return () => {
      cvs.removeEventListener('pointerdown', onDown)
      cvs.removeEventListener('pointermove', onMove)
      cvs.removeEventListener('pointerup', onUp)
      cvs.removeEventListener('pointercancel', onUp)
    }
  }, [color, width, send, user, tool, strokes])

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const doRegister = async () => {
    setAuthErr(null)
    try {
      await apiFetch('/api/register', { method: 'POST', body: JSON.stringify({ email, password }) })
      setPassword('')
      const u = await apiFetch('/api/me')
      setUser(u)
      setTimeout(() => { strokes.forEach((s) => send({ type: 'stroke', stroke: s })) }, 300)
    } catch (err) { setAuthErr('Registration failed') }
  }
  const doLogin = async () => {
    setAuthErr(null)
    try {
      await apiFetch('/api/login', { method: 'POST', body: JSON.stringify({ email, password }) })
      setPassword('')
      const u = await apiFetch('/api/me')
      setUser(u)
      setTimeout(() => { strokes.forEach((s) => send({ type: 'stroke', stroke: s })) }, 300)
    } catch (err) { setAuthErr('Login failed') }
  }
  const doLogout = async () => { await apiFetch('/api/logout', { method: 'POST' }); close(); setUser(null); setStrokes([]); setPassword('') }
  const doClear = async () => { 
    await apiFetch('/api/strokes/clear', { method: 'POST' }); 
    setStrokes([]); 
    const cvs = canvasRef.current; 
    if (cvs) { 
      const ctx = cvs.getContext('2d', { willReadFrequently: true })!; 
      ctx.clearRect(0, 0, cvs.width, cvs.height) 
    } 
  }

  const doRecognize = async () => {
    try {
      const cvs = canvasRef.current
      if (!cvs) return
      const res = await apiFetch('/api/recognize', { method: 'POST', body: JSON.stringify({ topN: 10, width: cvs.width, height: cvs.height }) })
      setCandidates(res.candidates || [])
    } catch (e) {
      setCandidates([])
    }
  }

  const doUndo = useCallback(() => {
    if (strokes.length === 0) return
    
    const lastStroke = strokes[strokes.length - 1]
    const newStrokes = strokes.slice(0, -1)
    
    console.log('All strokes:', strokes.map(s => ({ id: s.id, clientId: s.clientId, points: s.points.length })))
    console.log('Last stroke to undo:', { id: lastStroke.id, clientId: lastStroke.clientId, points: lastStroke.points.length })
    
    // Update strokes immediately for responsive UI
    setStrokes(newStrokes)
    
    // If the stroke has a valid ID, delete it from the server
    if (lastStroke.id && lastStroke.id > 0) {
      console.log('Undoing stroke with ID:', lastStroke.id, 'URL:', `/api/strokes/delete?id=${lastStroke.id}`)
      apiFetch(`/api/strokes/delete?id=${lastStroke.id}`, { 
        method: 'POST'
      }).catch(err => {
        console.error('Failed to delete stroke from server:', err)
        // If server deletion fails, we still removed it locally, so that's okay
      })
      
      // Send delete message via WebSocket for real-time updates
      send({ type: 'delete', delete: lastStroke.id })
    } else {
      console.log('Undoing local stroke without ID (not saved to server yet)')
      // For local strokes, we don't send any server requests or WebSocket messages
      // since they were never saved to the server
    }
  }, [strokes, send])

  // Keyboard event handling for Ctrl+Z undo
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
        e.preventDefault()
        doUndo()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [doUndo])

  return (
    <div style={{ height: '100vh', display: 'grid', gridTemplateRows: 'auto auto auto 1fr' }}>
      <header style={{ padding: 12, display: 'flex', gap: 12, alignItems: 'center' }}>
        <b>Drawing Board</b>
        <label>Color <input type="color" value={color} onChange={(e) => setColor(e.target.value)} disabled={tool !== 'pencil'} /></label>
        <label>Width <input type="range" min={1} max={20} value={width} onChange={(e) => setWidth(parseInt(e.target.value, 10))} /></label>
        <button onClick={() => setTool('pencil')} disabled={tool==='pencil'}>Pencil</button>
        <button onClick={() => setTool('eraser')} disabled={tool==='eraser'}>Eraser</button>
        {user && <button onClick={doUndo} disabled={strokes.length === 0} title="Undo last stroke (Ctrl+Z)">Undo</button>}
        <span style={{ marginLeft: 'auto', opacity: 0.7 }}>{user ? (ready ? 'Connected' : 'Connecting...') : 'Sign in to draw'}</span>
        {user && <button onClick={doClear}>Clear</button>}
        {user && <button onClick={doLogout}>Logout</button>}
      </header>

      {!user && (
        <div style={{ padding: 12, display: 'flex', gap: 8, alignItems: 'center' }}>
          <input placeholder="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <input placeholder="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
          <button onClick={doLogin}>Login</button>
          <button onClick={doRegister}>Register</button>
          {authErr && <span style={{ color: 'crimson' }}>{authErr}</span>}
        </div>
      )}

      {user && (
        <div style={{ padding: 12, display: 'flex', gap: 8, alignItems: 'center' }}>
          <button onClick={doRecognize}>Recognize</button>
          {candidates && candidates.length > 0 && (
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              {candidates.map((c, i) => (
                <span key={i} style={{ padding: '4px 8px', border: '1px solid #ddd', borderRadius: 4 }} title={`${c.score}`}>
                  {c.text}
                </span>
              ))}
            </div>
          )}
          {candidates && candidates.length === 0 && <span>No candidates</span>}
        </div>
      )}

      <div style={{ position: 'relative', display: 'flex', justifyContent: 'center', alignItems: 'center', padding: '20px' }}>
        <canvas 
          ref={canvasRef} 
          style={{ 
            width: '300px', 
            height: '300px', 
            touchAction: 'none', 
            background: '#fff',
            border: '2px solid #333',
            borderRadius: '8px',
            boxShadow: '0 2px 8px rgba(0,0,0,0.1)'
          }} 
        />
      </div>
    </div>
  )
}
