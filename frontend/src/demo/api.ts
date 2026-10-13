import QRCode from "qrcode";
import type { User, UserRole } from "../lib/userAccess";
import { demoShortUrl } from "./config";

type Link = {
  url_id: number;
  original_url: string;
  short_code: string;
  user_id: number | null;
  click_count: number;
  created_at: string;
  expires_at: string | null;
  is_active: boolean;
  visits: Record<string, number>;
};
type Report = {
  report_id: number;
  short_code: string;
  original_url: string;
  reporter_email: string;
  reason: string;
  details: string;
  status: string;
  created_at: string;
};
type Key = {
  key_id: number;
  user_id: number;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at: null;
};
type Domain = {
  domain_id: number;
  domain: string;
  reason: string;
  created_at: string;
};
type State = {
  version: 1;
  session: number | null;
  users: User[];
  links: Link[];
  reports: Report[];
  keys: Key[];
  domains: Domain[];
};
const roles: UserRole[] = ["owner", "admin", "support", "user"];
const storageKey = `shortli:demo:v1:${new URL(import.meta.env.BASE_URL, window.location.href).pathname}`;
const iso = () => new Date().toISOString();
const day = (offset = 0) =>
  new Date(Date.now() - offset * 86_400_000).toISOString().slice(0, 10);
const nextId = (values: number[]) => Math.max(0, ...values) + 1;
const reply = (body: unknown, status = 200) => Response.json(body, { status });
const fail = (en: string, ru: string, status = 400) =>
  reply({ error: document.documentElement.lang === "ru" ? ru : en }, status);

function seed(): State {
  const users = roles.map((role, i) => ({
    user_id: i + 1,
    email: `${role}@shortli.example`,
    role,
    is_admin: role === "owner" || role === "admin",
    created_at: iso(),
  }));
  const links: Link[] = Array.from({ length: 16 }, (_, i) => {
    const visits = Object.fromEntries(
      Array.from({ length: 14 }, (_, d) => [day(d), (i + d * 3) % 11]),
    );
    return {
      url_id: i + 1,
      original_url: `https://example.com/${["portfolio", "product", "newsletter", "event"][i % 4]}${i > 3 ? `-${i + 1}` : ""}`,
      short_code:
        ["portfolio", "product", "newsletter", "event"][i] || `demo-${i + 1}`,
      user_id: i < 12 ? 4 : i === 15 ? null : 2,
      created_at: new Date(Date.now() - 14 * 86_400_000).toISOString(),
      expires_at:
        i === 2 ? new Date(Date.now() - 86_400_000).toISOString() : null,
      is_active: i !== 1,
      click_count: Object.values(visits).reduce((sum, count) => sum + count, 0),
      visits,
    };
  });
  return {
    version: 1,
    session: null,
    users,
    links,
    keys: [],
    domains: [],
    reports: [
      {
        report_id: 1,
        short_code: "event",
        original_url: "https://example.com/event",
        reporter_email: "visitor@shortli.example",
        reason: "spam",
        details:
          "Demo report / Учебная жалоба: проверьте ссылку и выберите действие.",
        status: "pending",
        created_at: iso(),
      },
    ],
  };
}

function read(): State {
  try {
    const raw = localStorage.getItem(storageKey);
    if (raw) {
      const state = JSON.parse(raw) as State;
      if (
        state.version === 1 &&
        Array.isArray(state.users) &&
        Array.isArray(state.links) &&
        Array.isArray(state.keys) &&
        Array.isArray(state.reports) &&
        Array.isArray(state.domains) &&
        state.users.every((u) => roles.includes(u.role)) &&
        state.links.every(
          (l) =>
            typeof l.short_code === "string" &&
            l.visits &&
            typeof l.original_url === "string",
        )
      )
        return state;
    }
  } catch {
    return seed();
  }
  return seed();
}

function save(state: State) {
  localStorage.setItem(storageKey, JSON.stringify(state));
}

function normalizeDestination(value: unknown): string | null {
  if (typeof value !== "string" || value.length > 2048) return null;
  try {
    const input = value.trim();
    const url = new URL(
      /^[a-z][a-z\d+.-]*:/i.test(input) ? input : `https://${input}`,
    );
    if (
      !["https:", "http:"].includes(url.protocol) ||
      url.username ||
      url.password
    )
      return null;
    if (
      !url.hostname.includes(".") ||
      /^(localhost|127\.|10\.|192\.168\.|169\.254\.|0\.|\[)/i.test(
        url.hostname,
      ) ||
      /^172\.(1[6-9]|2\d|3[01])\./.test(url.hostname) ||
      /\.(local|localhost|internal)$/.test(url.hostname)
    )
      return null;
    return url.href;
  } catch {
    return null;
  }
}

function blocked(state: State, original: string) {
  const host = new URL(original).hostname;
  return state.domains.some(
    ({ domain }) => host === domain || host.endsWith(`.${domain}`),
  );
}

function expiry(value: unknown) {
  if (value === undefined || value === null || value === "") return null;
  const date = new Date(String(value));
  if (
    !Number.isFinite(date.getTime()) ||
    date.getTime() < Date.now() + 300_000 ||
    date.getTime() > Date.now() + 366 * 86_400_000
  )
    throw new Error("expiry");
  return date.toISOString();
}

async function present(link: Link, qr = false) {
  const { visits: _, ...record } = link;
  void _;
  const short_url = demoShortUrl(link.short_code);
  return {
    ...record,
    short_url,
    ...(qr
      ? {
          qr_code_base64: await QRCode.toDataURL(short_url, {
            errorCorrectionLevel: "H",
            width: 512,
            margin: 4,
          }),
        }
      : {}),
  };
}

function paginate<T>(data: T[], params: URLSearchParams) {
  const number = (name: string, fallback: number, max: number) =>
    Math.min(
      max,
      Math.max(1, Number.parseInt(params.get(name) || "", 10) || fallback),
    );
  const limit = number("limit", 10, 50);
  const totalPages = Math.max(1, Math.ceil(data.length / limit));
  const page = number("page", 1, totalPages);
  return {
    data: data.slice((page - 1) * limit, page * limit),
    meta: { total: data.length, totalPages, page, limit },
  };
}

async function handle(url: URL, method: string, body: Record<string, unknown>) {
  const state = read();
  const path = url.pathname;
  const user = state.users.find((u) => u.user_id === state.session);
  const staff = user && user.role !== "user";
  const admin = user && ["owner", "admin"].includes(user.role);
  const denied = () =>
    fail(
      "Authentication required",
      "Необходима авторизация.",
      user ? 403 : 401,
    );
  const done = (value: unknown = { ok: true }, status = 200) => {
    save(state);
    return reply(value, status);
  };

  if (path === "/api/demo/reset" && method === "POST") {
    save(seed());
    return reply({ ok: true });
  }
  if (path === "/api/demo/role" && method === "POST") {
    if (body.role === "guest") {
      state.session = null;
      return done();
    }
    if (!roles.includes(body.role as UserRole))
      return fail("Unknown role", "Неизвестная роль.");
    let profile = state.users.find((u) => u.role === body.role);
    if (!profile) {
      profile = {
        user_id: nextId(state.users.map((u) => u.user_id)),
        email: `${body.role}-${crypto.randomUUID().slice(0, 6)}@shortli.example`,
        role: body.role as UserRole,
        is_admin: ["owner", "admin"].includes(String(body.role)),
        created_at: iso(),
      };
      state.users.push(profile);
    }
    state.session = profile.user_id;
    return done(profile);
  }
  if (path === "/api/me" && method === "GET")
    return user ? reply(user) : denied();
  if (path === "/api/logout" && method === "POST") {
    state.session = null;
    return done();
  }
  if (["/api/register", "/api/login"].includes(path) && method === "POST") {
    const email = String(body.email || "")
      .trim()
      .toLowerCase();
    if (
      !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) ||
      String(body.password || "").length < 6
    )
      return fail(
        "Use a demo email and a password of at least 6 characters",
        "Укажите учебную почту и пароль от 6 символов.",
      );
    const existing = state.users.find((u) => u.email === email);
    if (path === "/api/register") {
      if (existing)
        return fail(
          "User with this email already exists",
          "Такая почта уже есть в песочнице.",
          409,
        );
      if (state.users.length >= 50)
        return fail(
          "Reset the demo to add more users",
          "Сбросьте демо для добавления пользователей.",
          429,
        );
      const profile: User = {
        user_id: nextId(state.users.map((u) => u.user_id)),
        email,
        role: "user",
        is_admin: false,
        created_at: iso(),
      };
      state.users.push(profile);
      return done(profile, 201);
    }
    if (!existing)
      return fail(
        "Register a demo profile or select a role above",
        "Зарегистрируйте учебный профиль или выберите роль сверху.",
        401,
      );
    state.session = existing.user_id;
    return done(existing);
  }
  if (path === "/api/health" && method === "GET")
    return reply({
      status: "operational",
      version: "demo",
      checked_at: iso(),
      services: { api: "operational", database: "operational" },
    });
  if (path === "/api/shorten" && method === "POST") {
    const original = normalizeDestination(body.original_url);
    if (!original)
      return fail(
        "Use a public HTTP or HTTPS URL",
        "Укажите публичный адрес HTTP или HTTPS.",
      );
    if (blocked(state, original))
      return fail(
        "This destination domain is blocked",
        "Домен заблокирован в песочнице.",
      );
    const expires_at = expiry(body.expires_at);
    const duplicate = state.links.find(
      (l) =>
        l.original_url === original && l.user_id === (user?.user_id ?? null),
    );
    if (duplicate) return reply(await present(duplicate, true));
    const alias = String(body.custom_alias || "")
      .trim()
      .toLowerCase();
    if (alias && !/^[a-z0-9_-]{3,32}$/.test(alias))
      return fail(
        "Invalid custom alias",
        "Окончание: 3–32 латинских символа, цифры, дефис или подчёркивание.",
      );
    if (
      [
        "api",
        "admin",
        "register",
        "login",
        "logout",
        "stats",
        "health",
        "privacy",
        "terms",
        "status",
        "developers",
        "report",
        "assets",
      ].includes(alias)
    )
      return fail(
        "Custom alias is reserved",
        "Это окончание зарезервировано сервисом.",
      );
    if (alias && state.links.some((l) => l.short_code === alias))
      return fail(
        "Custom alias is already in use",
        "Такое окончание ссылки уже занято.",
        409,
      );
    if (state.links.length >= 100)
      return fail(
        "The demo holds up to 100 links. Reset it to continue.",
        "В демо можно сохранить до 100 ссылок. Сбросьте песочницу.",
        429,
      );
    const link: Link = {
      url_id: nextId(state.links.map((l) => l.url_id)),
      short_code: alias || crypto.randomUUID().slice(0, 8),
      original_url: original,
      user_id: user?.user_id ?? null,
      created_at: iso(),
      expires_at,
      is_active: true,
      click_count: 0,
      visits: {},
    };
    const result = await present(link, true);
    state.links.unshift(link);
    return done(result, 201);
  }
  if (path === "/api/history" && method === "GET") {
    if (!user) return denied();
    const page = paginate(
      state.links.filter((l) => l.user_id === user.user_id),
      url.searchParams,
    );
    return reply({
      ...page,
      data: await Promise.all(page.data.map((l) => present(l, true))),
    });
  }
  const preview = path.match(/^\/api\/demo\/links\/([a-z0-9_-]+)(\/visit)?$/);
  if (
    preview &&
    ((!preview[2] && method === "GET") || (preview[2] && method === "POST"))
  ) {
    const link = state.links.find((l) => l.short_code === preview[1]);
    if (!link)
      return fail(
        "This link only exists in the browser where it was created, or has been deleted",
        "Ссылка доступна только в браузере, где создана, либо уже удалена.",
        404,
      );
    if (
      !link.is_active ||
      (link.expires_at && new Date(link.expires_at).getTime() <= Date.now()) ||
      blocked(state, link.original_url)
    )
      return fail(
        "This demo link is paused, expired or blocked",
        "Демо-ссылка приостановлена, просрочена или заблокирована.",
        410,
      );
    if (preview[2]) {
      link.click_count++;
      link.visits[day()] = (link.visits[day()] || 0) + 1;
      return done(await present(link));
    }
    return reply(await present(link));
  }
  const managed = path.match(
    /^\/api\/(admin\/)?urls\/([a-z0-9_-]+)(\/analytics)?$/,
  );
  if (managed) {
    if (!user || (managed[1] && !staff)) return denied();
    const link = state.links.find((l) => l.short_code === managed[2]);
    if (!link) return fail("URL not found", "Ссылка не найдена.", 404);
    if (!staff && link.user_id !== user.user_id) return denied();
    if (managed[3] && method === "GET") {
      const days = Math.min(
        90,
        Math.max(1, Number(url.searchParams.get("days")) || 30),
      );
      const daily = Array.from({ length: days }, (_, i) => ({
        label: day(days - i - 1),
        count: link.visits[day(days - i - 1)] || 0,
      }));
      const count = daily.reduce((sum, value) => sum + value.count, 0);
      const bucket = [{ label: "Demo", count }];
      return reply({
        short_code: link.short_code,
        period_days: days,
        lifetime_clicks: link.click_count,
        total_clicks: count,
        unique_clicks: Math.min(count, Math.ceil(count * 0.7)),
        daily,
        devices: bucket,
        browsers: bucket,
        operating_systems: bucket,
        referrers: bucket,
        countries: bucket,
      });
    }
    if (!managed[3] && method === "PATCH") {
      if (body.original_url !== undefined)
        return fail(
          "Destination cannot be changed",
          "Адрес назначения изменить нельзя.",
        );
      if (typeof body.is_active === "boolean") link.is_active = body.is_active;
      if (body.clear_expiration === true) link.expires_at = null;
      else if (body.expires_at !== undefined)
        link.expires_at = expiry(body.expires_at);
      return done(await present(link));
    }
    if (!managed[3] && method === "DELETE") {
      state.links = state.links.filter((l) => l !== link);
      state.reports = state.reports.filter(
        (r) => r.short_code !== link.short_code,
      );
      return done();
    }
  }
  if (path.startsWith("/api/admin/")) {
    if (!staff) return denied();
    if (
      ["/api/admin/urls", "/api/admin/search"].includes(path) &&
      method === "GET"
    ) {
      const query = (url.searchParams.get("q") || "").toLowerCase();
      const page = paginate(
        state.links.filter((l) =>
          `${l.original_url} ${l.short_code}`.toLowerCase().includes(query),
        ),
        url.searchParams,
      );
      return reply({
        ...page,
        data: await Promise.all(page.data.map((l) => present(l))),
      });
    }
    if (path === "/api/admin/users" && method === "GET") {
      if (!admin) return denied();
      return reply({
        ...paginate(state.users, url.searchParams),
        meta: {
          ...paginate(state.users, url.searchParams).meta,
          roleCounts: Object.fromEntries(
            roles.map((role) => [
              role,
              state.users.filter((u) => u.role === role).length,
            ]),
          ),
        },
      });
    }
    const member = path.match(/^\/api\/admin\/users\/(\d+)$/);
    if (member && method === "PUT") {
      if (!admin) return denied();
      const target = state.users.find((u) => u.user_id === Number(member[1]));
      if (!target)
        return fail("User not found", "Пользователь не найден.", 404);
      if (
        target.user_id === user.user_id ||
        (user.role !== "owner" &&
          (target.role === "owner" || body.role === "owner"))
      )
        return denied();
      if (!roles.includes(body.role as UserRole))
        return fail("Unknown role", "Неизвестная роль.");
      if (
        target.role === "owner" &&
        body.role !== "owner" &&
        state.users.filter((u) => u.role === "owner").length === 1
      )
        return fail(
          "The last owner cannot be removed",
          "Нельзя убрать последнего владельца.",
          409,
        );
      target.role = body.role as UserRole;
      target.is_admin = ["owner", "admin"].includes(target.role);
      return done(target);
    }
    if (path === "/api/admin/abuse-reports" && method === "GET")
      return reply({
        data: state.reports.filter((r) => r.status === "pending"),
      });
    if (path === "/api/admin/blocked-domains" && method === "GET")
      return reply({ data: state.domains });
    const report = path.match(/^\/api\/admin\/abuse-reports\/(\d+)$/);
    if (report && method === "PATCH") {
      const item = state.reports.find((r) => r.report_id === Number(report[1]));
      if (!item) return fail("Report not found", "Жалоба не найдена.", 404);
      item.status = String(body.status);
      const link = state.links.find((l) => l.short_code === item.short_code);
      if (body.pause_link && link) link.is_active = false;
      if (body.block_domain && link) {
        const domain = new URL(link.original_url).hostname;
        if (!state.domains.some((d) => d.domain === domain))
          state.domains.push({
            domain_id: nextId(state.domains.map((d) => d.domain_id)),
            domain,
            reason: item.reason,
            created_at: iso(),
          });
      }
      return done();
    }
    const domain = path.match(/^\/api\/admin\/blocked-domains\/(\d+)$/);
    if (domain && method === "DELETE") {
      if (!admin) return denied();
      state.domains = state.domains.filter(
        (d) => d.domain_id !== Number(domain[1]),
      );
      return done();
    }
  }
  if (path === "/api/abuse-reports" && method === "POST") {
    const code = String(body.short_link || "")
      .split(/[/?#]/)
      .filter(Boolean)
      .at(-1);
    const link = state.links.find((l) => l.short_code === code);
    if (!link)
      return fail("URL not found", "Ссылка не найдена в этой песочнице.", 404);
    if (state.reports.length >= 100)
      return fail(
        "Reset the demo to add more reports",
        "Сбросьте демо для добавления жалоб.",
        429,
      );
    state.reports.unshift({
      report_id: nextId(state.reports.map((r) => r.report_id)),
      short_code: link.short_code,
      original_url: link.original_url,
      reporter_email: String(body.reporter_email || "").slice(0, 254),
      reason: String(body.reason),
      details: String(body.details || "").slice(0, 2000),
      status: "pending",
      created_at: iso(),
    });
    return done({ ok: true }, 201);
  }
  if (path.startsWith("/api/developer/keys")) {
    if (!user) return denied();
    if (path === "/api/developer/keys" && method === "GET")
      return reply({
        data: state.keys.filter((k) => k.user_id === user.user_id),
      });
    if (path === "/api/developer/keys" && method === "POST") {
      const name = String(body.name || "")
        .trim()
        .slice(0, 80);
      if (name.length < 2)
        return fail("Enter a key name", "Укажите имя ключа.");
      if (state.keys.length >= 20)
        return fail(
          "Reset the demo to create more keys",
          "Сбросьте демо для создания ключей.",
          429,
        );
      const key = `demo_not_a_real_key_${crypto.randomUUID()}`;
      state.keys.push({
        key_id: nextId(state.keys.map((k) => k.key_id)),
        user_id: user.user_id,
        name,
        prefix: key.slice(0, 25),
        created_at: iso(),
        last_used_at: null,
      });
      return done({ key }, 201);
    }
    const key = path.match(/^\/api\/developer\/keys\/(\d+)$/);
    if (key && method === "DELETE") {
      state.keys = state.keys.filter(
        (k) => k.key_id !== Number(key[1]) || k.user_id !== user.user_id,
      );
      return done();
    }
  }
  return fail(
    "This operation is not available in the static demo",
    "Эта операция недоступна в статическом демо.",
    404,
  );
}

let pending: Promise<unknown> = Promise.resolve();

export function sendRequest(
  input: RequestInfo | URL,
  init: RequestInit = {},
): Promise<Response> {
  const operation = pending.then(async () => {
    init.signal?.throwIfAborted();
    try {
      const url = new URL(
        input instanceof Request ? input.url : String(input),
        window.location.origin,
      );
      const body =
        typeof init.body === "string"
          ? (JSON.parse(init.body) as Record<string, unknown>)
          : {};
      if (!body || typeof body !== "object" || Array.isArray(body))
        return fail("Invalid request", "Некорректный запрос.");
      const result = await handle(
        url,
        (init.method || "GET").toUpperCase(),
        body,
      );
      init.signal?.throwIfAborted();
      return result;
    } catch (error) {
      if (init.signal?.aborted) throw error;
      if (error instanceof Error && error.message === "expiry")
        return fail(
          "Expiration must be between 5 minutes and one year from now",
          "Срок должен быть от 5 минут до одного года.",
        );
      return fail(
        "Demo storage is unavailable or full. Reset the sandbox or allow browser storage.",
        "Хранилище демо недоступно или заполнено. Сбросьте песочницу или разрешите хранение данных.",
        503,
      );
    }
  });
  pending = operation.catch(() => undefined);
  return operation;
}
