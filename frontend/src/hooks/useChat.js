import { useGameStore } from '../store/gameStore'
import { useAuthStore } from '../store/authStore'

// Parsa i chunk SSE grezzi in eventi strutturati.
// Gin scrive: "event: message\ndata: {...}\n\n"
function parseSSEChunk(text) {
  const events = []
  // Divide per blocchi (separati da doppio newline)
  const blocks = text.split(/\n\n+/)
  for (const block of blocks) {
    let data = null
    for (const line of block.split('\n')) {
      if (line.startsWith('data: ')) {
        data = line.slice(6).trim()
      }
    }
    if (data) {
      try {
        events.push(JSON.parse(data))
      } catch {
        // chunk incompleto o heartbeat, ignora
      }
    }
  }
  return events
}

export function useChat() {
  const { startTurn, appendToken, finishTurn, failTurn } = useGameStore()
  const { token } = useAuthStore()

  async function sendMessage(message) {
    startTurn(message)

    let response
    try {
      response = await fetch('/api/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ message }),
      })
    } catch (err) {
      failTurn('Connessione al server fallita: ' + err.message)
      return
    }

    if (!response.ok) {
      const err = await response.json().catch(() => ({ error: 'Errore sconosciuto' }))
      failTurn(err.error || `HTTP ${response.status}`)
      return
    }

    // Legge lo stream SSE
    const reader = response.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })

        // Processa tutti gli eventi completi nel buffer
        const lastDouble = buffer.lastIndexOf('\n\n')
        if (lastDouble === -1) continue

        const complete = buffer.slice(0, lastDouble + 2)
        buffer = buffer.slice(lastDouble + 2)

        for (const event of parseSSEChunk(complete)) {
          if (event.type === 'token') {
            appendToken(event.text)
          } else if (event.type === 'done') {
            const p = event.payload
            finishTurn(p.narrative, p.character, p.inventory, p.session, p.ui_events, p.level_up, p.new_level, p.loot, p.overdrive)
          } else if (event.type === 'error') {
            failTurn(event.text)
            return
          }
        }
      }
    } catch (err) {
      failTurn('Connessione interrotta: ' + err.message)
    }
  }

  return { sendMessage }
}
