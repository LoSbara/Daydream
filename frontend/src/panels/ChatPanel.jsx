import { useEffect, useRef, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import { useChat } from '../hooks/useChat'

// Renderizza la narrativa del GM con markdown basilare
function NarrativeText({ text }) {
  // Evidenzia **grassetto**, *corsivo*, e va a capo su ---
  const lines = text.split('\n').map((line, i) => {
    if (line.trim() === '---') {
      return <hr key={i} className="border-surface-600 my-2" />
    }

    // Sostituisce **...** e *...*
    const parts = line.split(/(\*\*[^*]+\*\*|\*[^*]+\*)/g).map((part, j) => {
      if (part.startsWith('**') && part.endsWith('**')) {
        return <strong key={j} className="text-surface-100">{part.slice(2, -2)}</strong>
      }
      if (part.startsWith('*') && part.endsWith('*')) {
        return <em key={j} className="text-surface-400">{part.slice(1, -1)}</em>
      }
      return part
    })
    return <p key={i} className="mb-1 last:mb-0">{parts}</p>
  })

  return <div className="text-surface-200 leading-relaxed text-sm">{lines}</div>
}

function MessageBubble({ message }) {
  const isPlayer = message.role === 'player'
  const isSystem = message.role === 'system'

  if (isSystem) {
    return (
      <div className="text-center text-surface-500 text-xs italic py-1">
        {message.text}
      </div>
    )
  }

  return (
    <div className={`flex ${isPlayer ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[85%] rounded-lg px-3 py-2 ${
          isPlayer
            ? 'bg-accent/20 border border-accent/30 text-surface-200 text-sm'
            : 'bg-surface-700 border border-surface-600'
        }`}
      >
        {isPlayer ? (
          <p className="text-sm text-surface-200">{message.text}</p>
        ) : (
          <NarrativeText text={message.text} />
        )}
      </div>
    </div>
  )
}

function StreamingBubble({ text }) {
  if (!text) return null
  return (
    <div className="flex justify-start">
      <div className="max-w-[85%] rounded-lg px-3 py-2 bg-surface-700 border border-surface-600">
        <NarrativeText text={text} />
        <span className="inline-block w-1.5 h-3.5 bg-accent ml-0.5 animate-pulse align-text-bottom" />
      </div>
    </div>
  )
}

export default function ChatPanel() {
  const { messages, isStreaming, pendingText } = useGameStore()
  const { sendMessage } = useChat()
  const [input, setInput] = useState('')
  const bottomRef = useRef(null)
  const textareaRef = useRef(null)

  // Auto-scroll to bottom on new content
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, pendingText])

  function handleSubmit(e) {
    e.preventDefault()
    const msg = input.trim()
    if (!msg || isStreaming) return
    setInput('')
    sendMessage(msg)
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  return (
    <div className="h-full flex flex-col min-h-0">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-surface-600">
        <h2 className="text-surface-200 text-sm font-semibold">Sessione di gioco</h2>
        {isStreaming && (
          <span className="text-xs text-accent animate-pulse">GM sta scrivendo…</span>
        )}
      </div>

      {/* Messages area */}
      <div className="flex-1 overflow-y-auto min-h-0 p-3 flex flex-col">
        {messages.length === 0 && !isStreaming ? (
          <div className="flex-1 flex items-center justify-center text-surface-500 text-sm text-center">
            <div>
              <p className="text-2xl mb-2">⚔</p>
              <p>Scrivi un'azione per iniziare la sessione</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map((msg) => (
              <MessageBubble key={msg.id} message={msg} />
            ))}
            {isStreaming && <StreamingBubble text={pendingText} />}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Input */}
      <form onSubmit={handleSubmit} className="p-3 border-t border-surface-600">
        <div className="flex gap-2">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={isStreaming ? 'Attendi la risposta del GM…' : 'Cosa fai? (Invio per inviare, Shift+Invio per nuova riga)'}
            disabled={isStreaming}
            rows={2}
            className="input flex-1 resize-none text-sm leading-relaxed"
            maxLength={2000}
          />
          <button
            type="submit"
            disabled={isStreaming || !input.trim()}
            className="btn self-end px-4 py-2 text-sm"
          >
            Invia
          </button>
        </div>
      </form>
    </div>
  )
}
