import { useAuthStore } from '../store/authStore.js'

const BASE = '/api'

async function request(path, options = {}) {
  const { token, refreshToken, setAuth, clearAuth } = useAuthStore.getState()

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  let res = await fetch(`${BASE}${path}`, { ...options, headers })

  // Refresh automatico se access token scaduto
  if (res.status === 401 && refreshToken) {
    const refreshed = await tryRefresh(refreshToken)
    if (refreshed) {
      setAuth(refreshed.access_token, refreshToken, useAuthStore.getState().user)
      headers['Authorization'] = `Bearer ${refreshed.access_token}`
      res = await fetch(`${BASE}${path}`, { ...options, headers })
    } else {
      clearAuth()
      window.location.href = '/login'
      return
    }
  }

  const json = await res.json()

  if (!res.ok) {
    const err = new Error(json.error || `HTTP ${res.status}`)
    err.status = res.status
    throw err
  }

  return json.data ?? json
}

async function tryRefresh(refreshToken) {
  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return null
    const json = await res.json()
    return json.data ?? json
  } catch {
    return null
  }
}

export const api = {
  get: (path) => request(path),
  post: (path, body) => request(path, { method: 'POST', body: JSON.stringify(body) }),
  put: (path, body) => request(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (path) => request(path, { method: 'DELETE' }),
}
