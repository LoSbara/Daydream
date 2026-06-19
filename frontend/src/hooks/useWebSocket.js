import { useEffect, useRef } from 'react'
import { useAuthStore } from '../store/authStore.js'
import { useGameStore } from '../store/gameStore.js'

const WS_RECONNECT_MS = 5000

export function useWebSocket() {
  const { token } = useAuthStore()
  const addAnnouncement = useGameStore((s) => s.addAnnouncement)
  const wsRef = useRef(null)
  const reconnectTimer = useRef(null)

  useEffect(() => {
    if (!token) return

    let cancelled = false

    function connect() {
      if (cancelled) return
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const url = `${proto}//${window.location.host}/ws?token=${token}`
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (cancelled) { ws.close(1000, 'cancelled'); return }
        console.log('[ws] connesso')
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'announcement') {
            addAnnouncement({ id: Date.now(), text: msg.text, level: msg.level || 'info' })
          }
        } catch {
          // ignora messaggi non-JSON (es. ping)
        }
      }

      ws.onclose = (e) => {
        if (e.code !== 1000) {
          // Riconnessione automatica solo se chiusura inattesa
          reconnectTimer.current = setTimeout(connect, WS_RECONNECT_MS)
        }
      }

      ws.onerror = () => {
        ws.close()
      }
    }

    connect()

    return () => {
      cancelled = true
      clearTimeout(reconnectTimer.current)
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close(1000, 'unmount')
      }
      wsRef.current = null
    }
  }, [token]) // eslint-disable-line
}
