export const env = {
  // an empty URL = relative /api/v1 through the Vite proxy (dev) or a shared reverse proxy (prod),
  // so the backend doesn't need CORS
  apiBaseUrl: (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '',
};
