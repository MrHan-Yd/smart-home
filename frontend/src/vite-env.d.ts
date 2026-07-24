/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_AUTH_PORTAL_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
