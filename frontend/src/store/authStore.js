import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { STORAGE_PREFIX } from '../config.js'

export const useAuthStore = create(
  persist(
    (set) => ({
      token: null,
      refreshToken: null,
      user: null,

      setAuth: (token, refreshToken, user) =>
        set({ token, refreshToken, user }),

      clearAuth: () =>
        set({ token: null, refreshToken: null, user: null }),
    }),
    {
      name: `${STORAGE_PREFIX}-auth`,
      // Salva solo i dati essenziali in localStorage
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        user: state.user,
      }),
    }
  )
)
