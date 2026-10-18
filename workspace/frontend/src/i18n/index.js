// UI translations (EC-7): Vietnamese (default) and English, with i18next.
//
// All UI text lives in src/locales/<lang>.json (same keys in every file, a
// test checks this). Components call `const { t } = useTranslation()` and
// re-render when the language changes.
//
// Which language: the signed-in user's saved `locale` (from /auth/me, see
// AuthContext) wins; before login it is the last language picked in this
// browser (localStorage), then the browser language, then Vietnamese. Only the
// language code is stored in the browser, never anything about the user.
//
// To add a language: add src/locales/<code>.json, list it in LANGUAGES, and add
// the same code on the backend (i18n.Supported, migrations ck_user_config_locale).

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from '../locales/en.json';
import vi from '../locales/vi.json';

export const DEFAULT_LANGUAGE = 'vi';

/** Supported languages, in switcher order. `intl` is the locale for Intl formatting. */
export const LANGUAGES = [
  { code: 'vi', short: 'VI', label: 'Tiếng Việt', intl: 'vi-VN' },
  { code: 'en', short: 'EN', label: 'English', intl: 'en-US' },
];

export const LANGUAGE_STORAGE_KEY = 'mq-admin-lang';

const supported = (code) => LANGUAGES.some((l) => l.code === code);

/** "en-US" -> "en"; anything unsupported -> null. */
export function normalizeLanguage(tag) {
  const code = String(tag ?? '').toLowerCase().split(/[-_]/)[0];
  return supported(code) ? code : null;
}

function storedLanguage() {
  try {
    return normalizeLanguage(window.localStorage.getItem(LANGUAGE_STORAGE_KEY));
  } catch {
    return null; // storage blocked
  }
}

/** Language to start with before we know who the user is. */
export function detectLanguage() {
  const stored = storedLanguage();
  if (stored) return stored;
  const browser = typeof navigator === 'undefined' ? [] : (navigator.languages ?? [navigator.language]);
  for (const tag of browser) {
    const code = normalizeLanguage(tag);
    if (code) return code;
  }
  return DEFAULT_LANGUAGE;
}

/** Remember the choice for the next visit to the login page. */
export function rememberLanguage(code) {
  try {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, code);
  } catch {
    // storage blocked: the choice still applies until the page is reloaded
  }
}

/** Switches the UI language (no-op for unsupported codes). */
export async function applyLanguage(code) {
  const lang = normalizeLanguage(code);
  if (!lang) return;
  if (i18n.language !== lang) await i18n.changeLanguage(lang);
  rememberLanguage(lang);
}

/** Current language code ('vi' | 'en'). */
export function currentLanguage() {
  return normalizeLanguage(i18n.language) ?? DEFAULT_LANGUAGE;
}

/** Intl locale of the current language, e.g. 'vi-VN'. */
export function currentIntlLocale() {
  const lang = currentLanguage();
  return LANGUAGES.find((l) => l.code === lang).intl;
}

function syncDocument(lang) {
  if (typeof document !== 'undefined') document.documentElement.lang = lang;
}

i18n.use(initReactI18next).init({
  resources: { vi: { translation: vi }, en: { translation: en } },
  lng: detectLanguage(),
  fallbackLng: DEFAULT_LANGUAGE,
  supportedLngs: LANGUAGES.map((l) => l.code),
  // React escapes rendered text already; never render translations as HTML.
  interpolation: { escapeValue: false },
  returnNull: false,
  initImmediate: false, // resources are bundled: initialise synchronously
});
syncDocument(currentLanguage());
i18n.on('languageChanged', (lng) => syncDocument(normalizeLanguage(lng) ?? DEFAULT_LANGUAGE));

export default i18n;
