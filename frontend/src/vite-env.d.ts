/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Full API root shown in the Developer Quickstart, e.g. https://api.transcriptflow.io/api.
   *  Falls back to the current origin + /api when unset (accurate in dev and prod). */
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
