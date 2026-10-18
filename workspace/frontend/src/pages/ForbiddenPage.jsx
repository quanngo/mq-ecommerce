import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import ui from './ui.module.css';

export function ForbiddenPage() {
  const { t } = useTranslation();
  return (
    <main className={ui.main}>
      <section className={ui.card} aria-labelledby="forbidden-title">
        <h1 id="forbidden-title" className={ui.title}>
          {t('forbidden.title')}
        </h1>
        <p className={ui.muted}>{t('forbidden.text')}</p>
        <Link to="/profile" className={ui.secondaryButton}>
          {t('forbidden.back')}
        </Link>
      </section>
    </main>
  );
}
