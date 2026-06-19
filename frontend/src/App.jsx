import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/authStore.js'
import Login from './pages/Login.jsx'
import Game from './pages/Game.jsx'

function ProtectedRoute({ children }) {
  const token = useAuthStore((s) => s.token)
  return token ? children : <Navigate to="/login" replace />
}

function PublicRoute({ children }) {
  const token = useAuthStore((s) => s.token)
  return token ? <Navigate to="/game" replace /> : children
}

export default function App() {
  return (
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route
          path="/login"
          element={
            <PublicRoute>
              <Login />
            </PublicRoute>
          }
        />
        <Route
          path="/game"
          element={
            <ProtectedRoute>
              <Game />
            </ProtectedRoute>
          }
        />
        <Route path="*" element={<Navigate to="/game" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
