export class ApiError extends Error {
  status: number;

  constructor(message: string, status = 0) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function apiFetch(
  input: RequestInfo | URL,
  {
    timeoutMs = 15_000,
    signal,
    ...init
  }: RequestInit & { timeoutMs?: number } = {},
): Promise<Response> {
  const controller = new AbortController();
  const abort = () => controller.abort(signal?.reason);
  signal?.addEventListener("abort", abort, { once: true });
  if (signal?.aborted) abort();
  let timedOut = false;
  const timer = window.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);
  try {
    const response = await fetch(input, {
      credentials: "include",
      ...init,
      signal: controller.signal,
    });
    const body = await response.arrayBuffer();
    return new Response(
      response.status === 204 ||
        response.status === 205 ||
        response.status === 304
        ? null
        : body,
      {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers,
      },
    );
  } catch (error) {
    if (signal?.aborted) throw error;
    const russian = document.documentElement.lang === "ru";
    throw new ApiError(
      timedOut
        ? russian
          ? "Сервер не ответил вовремя. Проверьте состояние операции перед повтором."
          : "The request timed out. Check the operation status before retrying."
        : russian
          ? "Не удалось связаться с сервером. Проверьте подключение и повторите попытку."
          : "Could not reach the server. Check your connection and try again.",
    );
  } finally {
    window.clearTimeout(timer);
    signal?.removeEventListener("abort", abort);
  }
}

export async function apiJSON<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<T> {
  const response = await apiFetch(input, init);
  const data = await response.json().catch(() => null);
  if (!response.ok) {
    throw new ApiError(
      typeof data?.error === "string" ? data.error : "Request failed",
      response.status,
    );
  }
  if (data === null)
    throw new ApiError("Invalid server response", response.status);
  return data as T;
}
