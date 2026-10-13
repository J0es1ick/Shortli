import { useLocale } from "../context/LocaleContext";
import { isDemo } from "./config";

const notices = {
  analytics: {
    ru: "Учебная аналитика: исходные числа сгенерированы, оценка уникальных посетителей условная. Переходы, добавленные в предпросмотре, учитываются только в этом браузере.",
    en: "Demo analytics: initial numbers are generated and unique visitors are illustrative. Visits added in the preview are counted only in this browser.",
  },
  report: {
    ru: "Учебная жалоба остаётся в вашем браузере и не отправляется в настоящую поддержку. Для её проверки выберите роль администратора или техподдержки в панели демо.",
    en: "Demo reports stay in your browser and are not sent to real support. Select Admin or Support in the demo panel to review them yourself.",
  },
  api: {
    ru: "В песочнице API-ключи учебные и никуда не отправляются. HTTP API на GitHub Pages не работает; адрес ниже — только пример для полной серверной версии.",
    en: "Sandbox API keys are placeholders and are never sent anywhere. GitHub Pages does not run an HTTP API; the address below is an example for the full server version.",
  },
  status: {
    ru: "Это демонстрация страницы состояния. API и база данных ниже имитируются в браузере: здесь нет проверки рабочего сервера или доступности GitHub Pages.",
    en: "This is a status page demonstration. The API and database below are simulated in your browser; this is not a live server or GitHub Pages availability check.",
  },
  legal: {
    ru: "Ниже — документы полной версии продукта. В этой песочнице введённые ссылки и учебные профили хранятся только в браузере до сброса или очистки данных сайта. Не вводите личные данные. Сам хостинг GitHub Pages обрабатывает запросы согласно политике GitHub.",
    en: "Below are documents for the full product. In this sandbox, links and demo profiles stay in your browser until reset or site data deletion. Do not enter personal information. GitHub Pages itself handles hosting requests under GitHub's privacy policy.",
  },
  auth: {
    ru: "Учебный аккаунт: используйте вымышленную почту и пароль, например demo-pass-123. Пароль не сохраняется и не проверяется при входе. Это не настоящая авторизация.",
    en: "Demo account: use a made-up email and password, such as demo-pass-123. Passwords are not stored or verified on login. This is not real authentication.",
  },
};

export default function DemoNotice({ kind }: { kind: keyof typeof notices }) {
  const { locale } = useLocale();
  if (!isDemo) return null;
  return (
    <p className="demo-notice" role="note">
      {notices[kind][locale]}
    </p>
  );
}
