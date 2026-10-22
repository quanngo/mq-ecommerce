import { apiRequest, apiUpload } from './client';

// Product images (EC-10): uploaded straight to TikTok Shop or Lazada; the
// backend keeps only the URL (and TikTok's uri) the marketplace returns.

/** Same limits as the backend (services.MaxProductImageBytes, checkImage). */
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
export const ACCEPTED_IMAGE_TYPES = ['image/jpeg', 'image/png'];

/** Marketplaces images can be uploaded to, in display order (utils.Plaform values). */
export const IMAGE_MARKET_PLACES = ['Tiktok', 'Lazada'];

/**
 * models.ProductImage in the backend.
 * @typedef {object} ProductImage
 * @property {number} id
 * @property {'Tiktok' | 'Lazada'} market_place
 * @property {string} shop_id
 * @property {string} use_case     TikTok only, e.g. MAIN_IMAGE; '' for Lazada
 * @property {string} uri          TikTok image uri (used by TikTok's product APIs); '' for Lazada
 * @property {string} url          image URL on the marketplace CDN
 * @property {number} width
 * @property {number} height
 * @property {string} file_name
 * @property {number} size_bytes
 * @property {string | null} uploaded_by  uploader's email
 * @property {string} created_at
 */

/**
 * @typedef {object} ImageShops
 * @property {{ market_place: 'Tiktok' | 'Lazada', shop_id: string }[]} shops
 * @property {string[]} tiktok_use_cases
 */

/** Query keys GET /product-images accepts. */
export const IMAGE_QUERY_KEYS = ['market_place', 'shop_id', 'page', 'page_size'];

export const productImagesApi = {
  /** @returns {Promise<ImageShops>} */
  shops: async () => (await apiRequest('GET', '/product-images/shops')) ?? { shops: [], tiktok_use_cases: [] },

  /**
   * @param {Record<string, string>} query
   * @returns {Promise<{ images: ProductImage[], total: number, page: number, page_size: number }>}
   */
  list: async (query = {}) => {
    const params = new URLSearchParams();
    for (const k of IMAGE_QUERY_KEYS) {
      if (query[k]) params.set(k, query[k]);
    }
    const qs = params.toString();
    return (await apiRequest('GET', `/product-images${qs ? `?${qs}` : ''}`)) ?? { images: [], total: 0, page: 1, page_size: 24 };
  },

  /**
   * Uploads one image to a shop.
   * @param {{ marketPlace: string, shopId: string, useCase?: string, file: File }} input
   * @returns {Promise<ProductImage>}
   */
  upload: ({ marketPlace, shopId, useCase, file }) => {
    const form = new FormData();
    form.append('market_place', marketPlace);
    form.append('shop_id', shopId);
    if (marketPlace === 'Tiktok' && useCase) form.append('use_case', useCase);
    form.append('file', file, file.name);
    return apiUpload('/product-images', form);
  },
};

/**
 * Client-side check matching the backend, so a bad file is caught before upload.
 * @param {File} file
 * @returns {null | 'type' | 'size'}
 */
export function imageFileProblem(file) {
  if (!ACCEPTED_IMAGE_TYPES.includes(file.type)) return 'type';
  if (file.size === 0 || file.size > MAX_IMAGE_BYTES) return 'size';
  return null;
}
