import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore.js'

const BASE = '/api'

export default function Login() {
  const [mode, setMode] = useState('login') // 'login' | 'register'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const setAuth = useAuthStore((s) => s.setAuth)
  const navigate = useNavigate()

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const endpoint = mode === 'login' ? '/auth/login' : '/auth/register'
      const res = await fetch(`${BASE}${endpoint}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const json = await res.json()

      if (!res.ok) {
        setError(json.error || 'Errore sconosciuto')
        return
      }

      const { access_token, refresh_token, user } = json.data
      setAuth(access_token, refresh_token, user)
      navigate('/game')
    } catch (err) {
      setError('Impossibile contattare il server')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-surface px-4">
      <div className="w-full max-w-sm">

        {/* Logo / titolo */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-mono font-bold tracking-tight text-text">
            <span className="text-accent">Day</span>dream
          </h1>
          <p className="mt-2 text-text-muted text-sm">
            {mode === 'login' ? 'Bentornato.' : 'Crea il tuo account.'}
          </p>
        </div>

        {/* Card form */}
        <div className="card">
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div>
              <label className="block text-xs text-text-muted mb-1 uppercase tracking-wider">
                Username
              </label>
              <input
                className="input"
                type="text"
                placeholder="nome_utente"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                required
                minLength={3}
                maxLength={32}
              />
            </div>

            <div>
              <label className="block text-xs text-text-muted mb-1 uppercase tracking-wider">
                Password
              </label>
              <input
                className="input"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                required
                minLength={8}
              />
            </div>

            {error && (
              <p className="text-sm text-danger bg-danger/10 border border-danger/20 rounded px-3 py-2">
                {error}
              </p>
            )}

            <button
              type="submit"
              className="btn-primary w-full mt-1"
              disabled={loading}
            >
              {loading
                ? 'Caricamento...'
                : mode === 'login'
                ? 'Accedi'
                : 'Registrati'}
            </button>
          </form>

          {/* Toggle modalità */}
          <div className="mt-4 text-center text-sm text-text-muted">
            {mode === 'login' ? (
              <>
                Nessun account?{' '}
                <button
                  className="text-accent hover:underline"
                  onClick={() => { setMode('register'); setError('') }}
                >
                  Registrati
                </button>
              </>
            ) : (
              <>
                Hai già un account?{' '}
                <button
                  className="text-accent hover:underline"
                  onClick={() => { setMode('login'); setError('') }}
                >
                  Accedi
                </button>
              </>
            )}
          </div>
        </div>

      </div>
    </div>
  )
}
