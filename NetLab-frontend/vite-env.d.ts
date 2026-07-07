/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_AUTH_SIGNATURE_KEY?: string
  readonly VITE_AUTH_SIGNATURE_SALT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
