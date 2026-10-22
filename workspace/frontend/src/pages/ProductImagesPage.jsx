import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { IMAGE_MARKET_PLACES, imageFileProblem, productImagesApi } from '../api/productImages';
import { imageErrorMessage } from '../auth/errorMessages';
import { formatBytes, formatDateTime, formatNumber } from '../i18n/format';
import { marketPlaceLabel } from './orderFormat';
import { Spinner } from './Spinner';
import styles from './ProductImagesPage.module.css';
import ui from './ui.module.css';

const asError = (err) => (err instanceof ApiError ? err : 'INTERNAL_ERROR');
const FILTER_KEYS = ['market_place', 'shop_id'];

/**
 * Product images (EC-10, orders.write): upload images straight to a TikTok
 * Shop or Lazada shop, then list what was uploaded with the URL each
 * marketplace returned. List filters and page live in the URL.
 */
export function ProductImagesPage() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const search = params.toString();

  const [shops, setShops] = useState(null); // { shops, tiktok_use_cases } | null while loading
  const [shopsError, setShopsError] = useState(null);
  const [result, setResult] = useState(null);
  const [listError, setListError] = useState(null);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    productImagesApi
      .shops()
      .then(setShops)
      .catch((err) => {
        setShopsError(asError(err));
        setShops({ shops: [], tiktok_use_cases: [] });
      });
  }, []);

  useEffect(() => {
    let cancelled = false;
    setResult(null);
    setListError(null);
    productImagesApi
      .list(Object.fromEntries(new URLSearchParams(search)))
      .then((res) => !cancelled && setResult(res))
      .catch((err) => {
        if (cancelled) return;
        setListError(asError(err));
        setResult({ images: [], total: 0, page: 1, page_size: 24 });
      });
    return () => {
      cancelled = true;
    };
  }, [search, reloadKey]);

  function setFilter(key, value) {
    const next = new URLSearchParams();
    for (const k of FILTER_KEYS) {
      const v = k === key ? value : params.get(k);
      if (v) next.set(k, v);
    }
    if (key === 'market_place') next.delete('shop_id'); // shops belong to one marketplace
    setParams(next);
  }

  function goToPage(page) {
    const next = new URLSearchParams(params);
    if (page > 1) next.set('page', String(page));
    else next.delete('page');
    setParams(next);
  }

  const onUploaded = useCallback(() => setReloadKey((k) => k + 1), []);

  const allShops = shops?.shops ?? [];
  const filterMarket = params.get('market_place') ?? '';
  const filterShops = allShops.filter((s) => !filterMarket || s.market_place === filterMarket);
  const pages = result ? Math.max(1, Math.ceil(result.total / result.page_size)) : 1;
  const page = result?.page ?? 1;

  return (
    <main className={`${ui.main} ${styles.wide}`}>
      <h1 className={ui.title}>{t('productImages.title')}</h1>
      <p className={`${ui.muted} ${styles.intro}`}>{t('productImages.intro')}</p>

      {shops === null ? (
        <Spinner label={t('common.loading')} />
      ) : (
        <UploadForm shops={allShops} useCases={shops.tiktok_use_cases} shopsError={shopsError} onUploaded={onUploaded} />
      )}

      <section className={ui.card} aria-labelledby="images-title">
        <div className={styles.listHeader}>
          <h2 id="images-title" className={ui.sectionTitle}>
            {t('productImages.listTitle')}
          </h2>
          <div className={styles.listFilters} role="search" aria-label={t('productImages.filter')}>
            <label className={ui.label}>
              {t('productImages.marketPlace')}
              <select className={ui.select} value={filterMarket} onChange={(e) => setFilter('market_place', e.target.value)}>
                <option value="">{t('common.all')}</option>
                {IMAGE_MARKET_PLACES.map((m) => (
                  <option key={m} value={m}>
                    {marketPlaceLabel(m)}
                  </option>
                ))}
              </select>
            </label>
            <label className={ui.label}>
              {t('productImages.shop')}
              <select className={ui.select} value={params.get('shop_id') ?? ''} onChange={(e) => setFilter('shop_id', e.target.value)}>
                <option value="">{t('common.all')}</option>
                {filterShops.map((s) => (
                  <option key={`${s.market_place}:${s.shop_id}`} value={s.shop_id}>
                    {filterMarket ? s.shop_id : `${marketPlaceLabel(s.market_place)} · ${s.shop_id}`}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>

        {listError && (
          <div className={ui.error} role="alert">
            {imageErrorMessage(listError)}
          </div>
        )}

        {result === null ? (
          <Spinner label={t('productImages.loading')} />
        ) : result.images.length === 0 ? (
          <p className={ui.muted}>{t('productImages.empty')}</p>
        ) : (
          <>
            <p className={styles.summary}>{t('productImages.count', { count: result.total, total: formatNumber(result.total) })}</p>
            <ul className={styles.grid}>
              {result.images.map((img) => (
                <ImageCard key={img.id} image={img} />
              ))}
            </ul>
          </>
        )}

        {result && pages > 1 && (
          <nav className={styles.pager} aria-label={t('orders.pagination')}>
            <button className={ui.secondaryButton} type="button" disabled={page <= 1} onClick={() => goToPage(page - 1)}>
              {t('orders.prev')}
            </button>
            <span className={styles.pageInfo}>{t('orders.page', { page, pages })}</span>
            <button className={ui.secondaryButton} type="button" disabled={page >= pages} onClick={() => goToPage(page + 1)}>
              {t('orders.next')}
            </button>
          </nav>
        )}
      </section>
    </main>
  );
}

let nextItemId = 1;
const previewUrl = (file) => (typeof URL.createObjectURL === 'function' ? URL.createObjectURL(file) : '');
const revoke = (url) => url && typeof URL.revokeObjectURL === 'function' && URL.revokeObjectURL(url);

/**
 * Pick marketplace, shop, (TikTok) image type and files, then upload them one
 * by one. Each file shows its own status; bad files are flagged before upload.
 */
function UploadForm({ shops, useCases, shopsError, onUploaded }) {
  const { t } = useTranslation();
  const markets = IMAGE_MARKET_PLACES.filter((m) => shops.some((s) => s.market_place === m));
  const [market, setMarket] = useState(markets[0] ?? '');
  const [shopId, setShopId] = useState('');
  const [useCase, setUseCase] = useState('MAIN_IMAGE');
  const [items, setItems] = useState([]); // { id, file, preview, problem, status, error }
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState(null); // number of images uploaded in the last run
  const itemsRef = useRef(items);
  itemsRef.current = items;

  // Free the previews when the page goes away.
  useEffect(() => () => itemsRef.current.forEach((it) => revoke(it.preview)), []);

  const marketShops = shops.filter((s) => s.market_place === market);
  useEffect(() => {
    // Pick the only shop automatically; drop a shop that belongs to another marketplace.
    setShopId((current) => (marketShops.some((s) => s.shop_id === current) ? current : marketShops.length === 1 ? marketShops[0].shop_id : ''));
  }, [market, shops]); // marketShops is derived from these two

  if (shopsError || shops.length === 0) {
    return (
      <section className={ui.card} aria-labelledby="upload-title">
        <h2 id="upload-title" className={ui.sectionTitle}>
          {t('productImages.uploadTitle')}
        </h2>
        {shopsError ? (
          <div className={ui.error} role="alert">
            {t('productImages.loadShopsFailed')}
          </div>
        ) : (
          <p className={ui.muted}>{t('productImages.noShops')}</p>
        )}
      </section>
    );
  }

  function addFiles(fileList) {
    const added = Array.from(fileList).map((file) => {
      const problem = imageFileProblem(file);
      return { id: nextItemId++, file, preview: problem === 'type' ? '' : previewUrl(file), problem, status: 'waiting', error: null };
    });
    setItems((list) => [...list.filter((it) => it.status !== 'done'), ...added]);
    setNotice(null);
  }

  function removeItem(id) {
    setItems((list) => {
      const it = list.find((x) => x.id === id);
      if (it) revoke(it.preview);
      return list.filter((x) => x.id !== id);
    });
  }

  function clearAll() {
    items.forEach((it) => revoke(it.preview));
    setItems([]);
    setNotice(null);
  }

  const update = (id, patch) => setItems((list) => list.map((it) => (it.id === id ? { ...it, ...patch } : it)));
  const pending = items.filter((it) => !it.problem && it.status !== 'done');
  const canUpload = !busy && market && shopId && pending.length > 0;

  async function onSubmit(e) {
    e.preventDefault();
    if (!canUpload) return;
    setBusy(true);
    setNotice(null);
    let uploaded = 0;
    for (const it of pending) {
      update(it.id, { status: 'uploading', error: null });
      try {
        await productImagesApi.upload({ marketPlace: market, shopId, useCase, file: it.file });
        update(it.id, { status: 'done' });
        uploaded++;
      } catch (err) {
        update(it.id, { status: 'error', error: asError(err) });
      }
    }
    setBusy(false);
    if (uploaded > 0) {
      setNotice(uploaded);
      onUploaded();
    }
  }

  const doneCount = items.filter((it) => it.status === 'done').length;

  return (
    <section className={ui.card} aria-labelledby="upload-title">
      <h2 id="upload-title" className={ui.sectionTitle}>
        {t('productImages.uploadTitle')}
      </h2>
      {notice !== null && (
        <div className={ui.success} role="status">
          {t('productImages.uploadedCount', { count: notice })}
        </div>
      )}
      <form className={styles.uploadForm} onSubmit={onSubmit} noValidate>
        <div className={styles.uploadFields}>
          <label className={ui.label}>
            {t('productImages.marketPlace')}
            <select className={ui.select} value={market} disabled={busy} onChange={(e) => setMarket(e.target.value)}>
              {markets.map((m) => (
                <option key={m} value={m}>
                  {marketPlaceLabel(m)}
                </option>
              ))}
            </select>
          </label>
          <label className={ui.label}>
            {t('productImages.shop')}
            <select className={ui.select} value={shopId} disabled={busy} onChange={(e) => setShopId(e.target.value)}>
              <option value="">{t('productImages.chooseShop')}</option>
              {marketShops.map((s) => (
                <option key={s.shop_id} value={s.shop_id}>
                  {s.shop_id}
                </option>
              ))}
            </select>
          </label>
          {market === 'Tiktok' && (
            <label className={ui.label}>
              {t('productImages.useCase')}
              <select className={ui.select} value={useCase} disabled={busy} onChange={(e) => setUseCase(e.target.value)}>
                {useCases.map((u) => (
                  <option key={u} value={u}>
                    {t(`productImages.useCases.${u}`, { defaultValue: u })}
                  </option>
                ))}
              </select>
            </label>
          )}
        </div>

        <div className={ui.label}>
          {/* Hint outside the <label>, so the field's accessible name stays just "files". */}
          <label htmlFor="product-image-files">{t('productImages.files')}</label>
          <span id="product-image-files-hint" className={ui.hint}>
            {t('productImages.filesHint')}
          </span>
          <input
            id="product-image-files"
            aria-describedby="product-image-files-hint"
            className={styles.fileInput}
            type="file"
            accept="image/jpeg,image/png"
            multiple
            disabled={busy}
            onChange={(e) => {
              addFiles(e.target.files ?? []);
              e.target.value = ''; // allow picking the same file again
            }}
          />
        </div>

        {items.length > 0 && (
          <ul className={styles.queue}>
            {items.map((it) => (
              <li key={it.id} className={styles.queueItem}>
                {it.preview ? <img className={styles.queueThumb} src={it.preview} alt="" /> : <span className={styles.queueThumb} aria-hidden="true" />}
                <div className={styles.queueInfo}>
                  <div className={styles.fileName}>{it.file.name}</div>
                  <div className={styles.meta}>{formatBytes(it.file.size)}</div>
                  {it.problem ? (
                    <div className={styles.problem}>{t(`productImages.problem.${it.problem}`)}</div>
                  ) : it.status === 'error' ? (
                    <div className={styles.problem} role="alert">
                      {imageErrorMessage(it.error)}
                    </div>
                  ) : (
                    <span className={it.status === 'done' ? ui.badge : ui.badgeMuted}>{t(`productImages.status.${it.status}`)}</span>
                  )}
                </div>
                <button
                  type="button"
                  className={styles.removeButton}
                  aria-label={t('productImages.remove', { name: it.file.name })}
                  disabled={busy && it.status === 'uploading'}
                  onClick={() => removeItem(it.id)}
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
        )}

        <div className={ui.actions}>
          {items.length > 0 && (
            <button type="button" className={ui.secondaryButton} disabled={busy} onClick={clearAll}>
              {t('productImages.clear')}
            </button>
          )}
          <button type="submit" className={ui.primaryButton} disabled={!canUpload}>
            {busy ? t('productImages.uploading', { done: doneCount, total: doneCount + pending.length }) : t('productImages.upload')}
          </button>
        </div>
      </form>
    </section>
  );
}

/** One uploaded image: thumbnail from the marketplace CDN, details and copy buttons. */
function ImageCard({ image }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(null); // 'url' | 'uri' | 'failed' | null

  async function copy(kind, value) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(kind);
    } catch {
      setCopied('failed');
    }
    setTimeout(() => setCopied(null), 2000);
  }

  const name = image.file_name || `#${image.id}`;
  return (
    <li className={styles.card}>
      <a className={styles.thumbLink} href={image.url} target="_blank" rel="noopener noreferrer" aria-label={t('productImages.open', { name })}>
        <img className={styles.thumb} src={image.url} alt="" loading="lazy" referrerPolicy="no-referrer" />
      </a>
      <div className={styles.cardBody}>
        <div className={styles.fileName} title={name}>
          {name}
        </div>
        <div className={styles.badges}>
          <span className={ui.badge}>{marketPlaceLabel(image.market_place)}</span>
          {image.use_case && <span className={ui.badgeMuted}>{t(`productImages.useCases.${image.use_case}`, { defaultValue: image.use_case })}</span>}
        </div>
        <div className={styles.meta}>
          {t('productImages.shop')}: {image.shop_id}
        </div>
        <div className={styles.meta}>
          {image.width > 0 && `${t('productImages.dims', { width: image.width, height: image.height })} · `}
          {formatBytes(image.size_bytes)}
        </div>
        <div className={styles.meta}>
          {formatDateTime(image.created_at)}
          {image.uploaded_by && ` · ${t('productImages.by', { email: image.uploaded_by })}`}
        </div>
        <div className={styles.cardActions}>
          <button type="button" className={ui.secondaryButton} onClick={() => copy('url', image.url)}>
            {copied === 'url' ? t('productImages.copied') : t('productImages.copyUrl')}
          </button>
          {image.uri && (
            <button type="button" className={ui.secondaryButton} onClick={() => copy('uri', image.uri)}>
              {copied === 'uri' ? t('productImages.copied') : t('productImages.copyUri')}
            </button>
          )}
        </div>
        {copied === 'failed' && (
          <div className={styles.problem} role="alert">
            {t('productImages.copyFailed')}
          </div>
        )}
      </div>
    </li>
  );
}
