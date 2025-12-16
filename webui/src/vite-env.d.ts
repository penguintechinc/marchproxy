/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_WS_URL: string;
  readonly VITE_LICENSE_SERVER_URL: string;
  readonly VITE_PRODUCT_NAME: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
