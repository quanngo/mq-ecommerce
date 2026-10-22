import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { backend } from '../test/server';
import { renderApp } from '../test/renderApp';

const png = (name, size = 2048) => new File([new Uint8Array(size)], name, { type: 'image/png' });
const jpg = (name) => new File([new Uint8Array(1024)], name, { type: 'image/jpeg' });

async function openPage(path = '/product-images', role = 'operator') {
  backend.signedIn = true;
  backend.role = role;
  renderApp(path);
  await screen.findByRole('heading', { name: 'Ảnh sản phẩm', level: 1 });
  await screen.findByRole('heading', { name: 'Tải ảnh lên sàn' });
  // accept is enforced by the backend and our own check; let tests pick any file.
  return userEvent.setup({ applyAccept: false });
}

const fileInput = () => screen.getByLabelText('Ảnh');
const uploadSection = () => screen.getByRole('heading', { name: 'Tải ảnh lên sàn' }).closest('section');

describe('access', () => {
  it('shows the menu with orders.write', async () => {
    backend.signedIn = true;
    backend.role = 'operator';
    renderApp('/profile');
    await screen.findByRole('heading', { name: /Xin chào/ });
    expect(screen.getByRole('link', { name: 'Ảnh sản phẩm' })).toHaveAttribute('href', '/product-images');
  });

  it('hides the menu and blocks the page without orders.write', async () => {
    backend.signedIn = true;
    backend.role = 'viewer';
    renderApp('/product-images');
    expect(await screen.findByRole('heading', { name: 'Không có quyền truy cập' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Ảnh sản phẩm' })).not.toBeInTheDocument();
    expect(backend.imageUploads).toEqual([]);
    expect(backend.lastImagesQuery).toBeNull();
  });
});

describe('upload', () => {
  it('uploads to a TikTok shop with the chosen image type and lists the result', async () => {
    const user = await openPage();
    const form = within(uploadSection());
    expect(form.getByLabelText('Sàn')).toHaveDisplayValue('TikTok Shop');
    await user.selectOptions(form.getByLabelText('Shop'), 'tt-shop');
    await user.selectOptions(form.getByLabelText('Loại ảnh (TikTok)'), 'DESCRIPTION_IMAGE');
    await user.upload(fileInput(), [png('serum.png'), jpg('toner.jpg')]);
    expect(form.getAllByText('Chờ tải')).toHaveLength(2);

    await user.click(form.getByRole('button', { name: 'Tải lên' }));

    expect(await form.findByRole('status')).toHaveTextContent('Đã tải lên 2 ảnh.');
    expect(backend.imageUploads).toEqual([
      { market_place: 'Tiktok', shop_id: 'tt-shop', use_case: 'DESCRIPTION_IMAGE', file_name: 'serum.png', size: 2048 },
      { market_place: 'Tiktok', shop_id: 'tt-shop', use_case: 'DESCRIPTION_IMAGE', file_name: 'toner.jpg', size: 1024 },
    ]);
    expect(form.getAllByText('Đã tải lên')).toHaveLength(2);

    // The list reloads and shows the marketplace URL.
    const card = (await screen.findByText('toner.jpg', { selector: '[title]' })).closest('li');
    expect(within(card).getByText('Ảnh mô tả')).toBeInTheDocument();
    expect(within(card).getByRole('link', { name: 'Mở ảnh toner.jpg' })).toHaveAttribute('href', 'https://cdn.example/tiktok/2.png');
    expect(screen.getByText('2 ảnh')).toBeInTheDocument();

    await user.click(within(card).getByRole('button', { name: 'Sao chép URL' }));
    expect(await navigator.clipboard.readText()).toBe('https://cdn.example/tiktok/2.png');
    expect(within(card).getByRole('button', { name: 'Đã sao chép' })).toBeInTheDocument();
    await user.click(within(card).getByRole('button', { name: 'Sao chép URI' }));
    expect(await navigator.clipboard.readText()).toBe('tos/img-2');
  });

  it('flags bad files before upload and never sends them', async () => {
    const user = await openPage();
    const form = within(uploadSection());
    await user.selectOptions(form.getByLabelText('Shop'), 'tt-shop');
    const gif = new File([new Uint8Array(10)], 'anim.gif', { type: 'image/gif' });
    const huge = png('huge.png', 5 * 1024 * 1024 + 1);
    await user.upload(fileInput(), [gif, huge]);

    expect(form.getByText('Chỉ nhận JPG hoặc PNG.')).toBeInTheDocument();
    expect(form.getByText('Ảnh phải nhỏ hơn 5 MB.')).toBeInTheDocument();
    expect(form.getByRole('button', { name: 'Tải lên' })).toBeDisabled();

    await user.upload(fileInput(), png('ok.png'));
    await user.click(form.getByRole('button', { name: 'Tải lên' }));
    await form.findByRole('status');
    expect(backend.imageUploads.map((u) => u.file_name)).toEqual(['ok.png']);
  });

  it('needs a shop, and Lazada has no image type', async () => {
    const user = await openPage();
    const form = within(uploadSection());
    await user.upload(fileInput(), png('a.png'));
    expect(form.getByRole('button', { name: 'Tải lên' })).toBeDisabled(); // no shop chosen yet

    await user.selectOptions(form.getByLabelText('Sàn'), 'Lazada');
    expect(form.queryByLabelText('Loại ảnh (TikTok)')).not.toBeInTheDocument();
    expect(form.getByLabelText('Shop')).toHaveDisplayValue('lz-shop'); // the only Lazada shop is picked
    await user.click(form.getByRole('button', { name: 'Tải lên' }));
    await form.findByRole('status');
    expect(backend.imageUploads).toEqual([{ market_place: 'Lazada', shop_id: 'lz-shop', use_case: null, file_name: 'a.png', size: 2048 }]);
  });

  it('shows a marketplace refusal on that file only and lets it be retried', async () => {
    backend.imageFailNames = ['bad.png'];
    const user = await openPage();
    const form = within(uploadSection());
    await user.selectOptions(form.getByLabelText('Shop'), 'tt-shop');
    await user.upload(fileInput(), [png('good.png'), png('bad.png')]);
    await user.click(form.getByRole('button', { name: 'Tải lên' }));

    expect(await form.findByRole('alert')).toHaveTextContent('Sàn từ chối ảnh hoặc không phản hồi. Vui lòng thử lại.');
    expect(form.getByRole('status')).toHaveTextContent('Đã tải lên 1 ảnh.');

    backend.imageFailNames = [];
    await user.click(form.getByRole('button', { name: 'Tải lên' }));
    await waitFor(() => expect(form.getAllByText('Đã tải lên')).toHaveLength(2));
    expect(backend.imageUploads.map((u) => u.file_name)).toEqual(['good.png', 'bad.png', 'bad.png']);
  });

  it('explains a rejected image in the current language', async () => {
    backend.imageError = 'INVALID_IMAGE';
    const user = await openPage();
    const form = within(uploadSection());
    await user.selectOptions(form.getByLabelText('Shop'), 'tt-shop');
    await user.upload(fileInput(), png('fake.png'));
    await user.click(form.getByRole('button', { name: 'Tải lên' }));
    expect(await form.findByRole('alert')).toHaveTextContent('Ảnh không hợp lệ: chỉ nhận JPG hoặc PNG, tối đa 5 MB.');

    await i18n.changeLanguage('en');
    expect(await form.findByRole('alert')).toHaveTextContent('Invalid image: only JPG or PNG up to 5 MB.');
  });

  it('says so when no shop is connected', async () => {
    backend.imageShops = [];
    backend.signedIn = true;
    backend.role = 'operator';
    renderApp('/product-images');
    expect(await screen.findByText('Chưa có shop TikTok Shop hoặc Lazada nào được kết nối.')).toBeInTheDocument();
    expect(screen.queryByLabelText('Ảnh')).not.toBeInTheDocument();
  });
});

describe('list', () => {
  it('filters by marketplace and shop through the URL', async () => {
    backend.images = [
      { id: 2, market_place: 'Lazada', shop_id: 'lz-shop', use_case: '', uri: '', url: 'https://cdn/l.png', width: 0, height: 0, file_name: 'l.png', size_bytes: 1500000, uploaded_by: null, created_at: '2026-10-18T03:00:00Z' },
      { id: 1, market_place: 'Tiktok', shop_id: 'tt-shop', use_case: 'MAIN_IMAGE', uri: 'tos/1', url: 'https://cdn/t.png', width: 800, height: 600, file_name: 't.png', size_bytes: 2048, uploaded_by: 'op@mq.com', created_at: '2026-10-17T03:00:00Z' },
    ];
    const user = await openPage();
    expect(await screen.findByText('2 ảnh')).toBeInTheDocument();
    const tiktokCard = screen.getByText('t.png', { selector: '[title]' }).closest('li');
    expect(within(tiktokCard).getByText('800×600 px · 2 KB')).toBeInTheDocument();
    expect(within(tiktokCard).getByText(/bởi op@mq.com/)).toBeInTheDocument();
    const lazadaCard = screen.getByText('l.png', { selector: '[title]' }).closest('li');
    expect(within(lazadaCard).getByText('1,4 MB')).toBeInTheDocument();
    expect(within(lazadaCard).queryByRole('button', { name: 'Sao chép URI' })).not.toBeInTheDocument();

    const filters = within(screen.getByRole('search', { name: 'Lọc' }));
    await user.selectOptions(filters.getByLabelText('Sàn'), 'Lazada');
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/product-images?market_place=Lazada'));
    expect(await screen.findByText('1 ảnh')).toBeInTheDocument();
    expect(backend.lastImagesQuery).toEqual({ market_place: 'Lazada' });

    await user.selectOptions(filters.getByLabelText('Shop'), 'lz-shop');
    await waitFor(() => expect(backend.lastImagesQuery).toEqual({ market_place: 'Lazada', shop_id: 'lz-shop' }));
  });

  it('shows an empty state', async () => {
    await openPage();
    expect(await screen.findByText('Chưa có ảnh nào.')).toBeInTheDocument();
  });
});
