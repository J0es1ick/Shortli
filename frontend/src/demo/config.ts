export const isDemo = import.meta.env.VITE_DEMO_MODE === "true";

export const demoShortUrl = (code: string) =>
  `${new URL(import.meta.env.BASE_URL, window.location.href).href}#/s/${encodeURIComponent(code)}`;
