// Единый клиент админ-API. Все запросы — same-origin (/admin/v1), сессия в
// httpOnly-cookie, поэтому credentials обязательны.

const BASE = '/admin/v1';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

// Сообщения для пользователя по кодам §4.7 — бэкенд отвечает по-английски,
// панель показывает русский текст.
const MESSAGES: Record<string, string> = {
  unauthorized: 'Неверные данные или сессия истекла.',
  forbidden: 'Действие запрещено.',
  validation_failed: 'Проверьте правильность заполнения полей.',
  not_found: 'Не найдено.',
  rate_limited: 'Слишком много попыток. Подождите немного и повторите.',
  payload_too_large: 'Файл или запрос слишком большой.',
  unsupported_media_type: 'Неподдерживаемый формат файла.',
  service_unavailable: 'Сервис временно недоступен. Попробуйте позже.',
  internal_error: 'Внутренняя ошибка сервера. Мы уже видим её в журнале.',
};

export function humanError(err: unknown): string {
  if (err instanceof ApiError) {
    return MESSAGES[err.code] ?? err.message;
  }
  return 'Нет связи с сервером. Проверьте интернет-соединение.';
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: 'same-origin',
    headers: init?.body instanceof FormData ? undefined : { 'Content-Type': 'application/json' },
    ...init,
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    // Prefer the backend's §4.7 envelope; if the body is not JSON (e.g. a proxy
    // 502/504 HTML page or a bare 429/413 before reaching the app), derive the
    // code from the HTTP status so the operator sees a meaningful message.
    const code = body?.error?.code ?? codeFromStatus(res.status);
    const message = body?.error?.message ?? res.statusText;
    throw new ApiError(res.status, code, message);
  }
  return body as T;
}

function codeFromStatus(status: number): string {
  switch (status) {
    case 401:
      return 'unauthorized';
    case 403:
      return 'forbidden';
    case 404:
      return 'not_found';
    case 413:
      return 'payload_too_large';
    case 415:
      return 'unsupported_media_type';
    case 429:
      return 'rate_limited';
    case 502:
    case 503:
    case 504:
      return 'service_unavailable';
    default:
      return 'internal_error';
  }
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, payload?: unknown) =>
    request<T>(path, { method: 'POST', body: payload === undefined ? undefined : JSON.stringify(payload) }),
  put: <T>(path: string, payload: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(payload) }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  upload: <T>(path: string, form: FormData) => request<T>(path, { method: 'POST', body: form }),
};
