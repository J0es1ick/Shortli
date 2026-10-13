import { demoShortUrl, isDemo } from "../demo/config";

const configuredBase = (
  import.meta.env.VITE_API_BASE_URL as string | undefined
)?.replace(/\/$/, "");
const configuredPublicBase = (
  import.meta.env.VITE_PUBLIC_BASE_URL as string | undefined
)?.replace(/\/$/, "");

export const API_BASE = isDemo ? "" : (configuredBase ?? "");

export const apiUrl = (path: string) =>
  `${API_BASE}${path.startsWith("/") ? path : `/${path}`}`;

export const buildShortUrl = (shortCode: string, serverUrl?: string) =>
  isDemo
    ? demoShortUrl(shortCode)
    : serverUrl ||
      `${configuredPublicBase || window.location.origin}/${shortCode}`;
