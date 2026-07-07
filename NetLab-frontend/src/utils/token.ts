const TOKEN_KEY = 'netlab_access_token'
const REFRESH_KEY = 'netlab_refresh_token'

export const tokenStorage = {
  getAccessToken(): string | null {
    return localStorage.getItem(TOKEN_KEY)
  },

  setAccessToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token)
  },

  getRefreshToken(): string | null {
    return localStorage.getItem(REFRESH_KEY)
  },

  setRefreshToken(token: string): void {
    localStorage.setItem(REFRESH_KEY, token)
  },

  clear(): void {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}
