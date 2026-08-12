/* eslint-disable react-refresh/only-export-components */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import {
  detectLocale,
  interpolate,
  translations,
  type Locale,
  type TranslationKey,
} from "../i18n/translations";

interface LocaleContextValue {
  locale: Locale;
  toggleLocale: () => void;
  t: (key: TranslationKey, values?: Record<string, string | number>) => string;
  formatDate: (
    value: string | Date,
    options?: Intl.DateTimeFormatOptions,
  ) => string;
  formatNumber: (value: number) => string;
  apiError: (message: string | undefined, fallback: TranslationKey) => string;
}

const LocaleContext = createContext<LocaleContextValue | undefined>(undefined);
const storageKey = "shortli:locale";

const ruApiErrors: Record<string, string> = {
  "Invalid email or password": "Неверная почта или пароль.",
  "User with this email already exists":
    "Пользователь с такой почтой уже существует.",
  "Password must be at least 6 characters":
    "Пароль должен содержать не менее 6 символов.",
  "Email and password are required": "Укажите почту и пароль.",
  "Required original_url": "Введите исходную ссылку.",
  "Authentication required": "Необходима авторизация.",
  "Admin access required": "Необходимы права администратора.",
  "Custom alias is already in use": "Такое окончание ссылки уже занято.",
  "custom alias must be between 3 and 32 characters":
    "Окончание должно содержать от 3 до 32 символов.",
  "custom alias may contain only latin letters, numbers, hyphens, and underscores":
    "Используйте только латинские буквы, цифры, дефисы и подчёркивания.",
  "custom alias is reserved": "Это окончание зарезервировано сервисом.",
};

export function LocaleProvider({ children }: { children: ReactNode }) {
  const storedLocale = localStorage.getItem(storageKey);
  const [isManual, setIsManual] = useState(
    storedLocale === "en" || storedLocale === "ru",
  );
  const [locale, setLocale] = useState<Locale>(() =>
    storedLocale === "en" || storedLocale === "ru"
      ? storedLocale
      : detectLocale(),
  );

  const t = useCallback(
    (key: TranslationKey, values?: Record<string, string | number>) =>
      interpolate(translations[locale][key], values),
    [locale],
  );

  useEffect(() => {
    document.documentElement.lang = locale;
    document.title = translations[locale]["meta.title"];
    document
      .querySelector('meta[name="description"]')
      ?.setAttribute("content", translations[locale]["meta.description"]);
  }, [locale]);

  useEffect(() => {
    if (isManual) return;
    const handleLanguageChange = () => setLocale(detectLocale());
    window.addEventListener("languagechange", handleLanguageChange);
    return () =>
      window.removeEventListener("languagechange", handleLanguageChange);
  }, [isManual]);

  const toggleLocale = useCallback(() => {
    setLocale((current) => {
      const next = current === "ru" ? "en" : "ru";
      localStorage.setItem(storageKey, next);
      return next;
    });
    setIsManual(true);
  }, []);

  const formatDate = useCallback(
    (value: string | Date, options?: Intl.DateTimeFormatOptions) =>
      new Intl.DateTimeFormat(
        locale === "ru" ? "ru-RU" : "en-US",
        options,
      ).format(new Date(value)),
    [locale],
  );

  const formatNumber = useCallback(
    (value: number) =>
      new Intl.NumberFormat(locale === "ru" ? "ru-RU" : "en-US").format(value),
    [locale],
  );

  const apiError = useCallback(
    (message: string | undefined, fallback: TranslationKey) => {
      if (!message) return t(fallback);
      if (locale === "en") return message;
      return ruApiErrors[message] || t(fallback);
    },
    [locale, t],
  );

  const value = useMemo(
    () => ({ locale, toggleLocale, t, formatDate, formatNumber, apiError }),
    [apiError, formatDate, formatNumber, locale, t, toggleLocale],
  );

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  );
}

export const useLocale = () => {
  const context = useContext(LocaleContext);
  if (!context)
    throw new Error("useLocale must be used within a LocaleProvider");
  return context;
};
