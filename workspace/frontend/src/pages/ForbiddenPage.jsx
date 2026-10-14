import { Link } from 'react-router-dom';
import ui from './ui.module.css';

export function ForbiddenPage() {
  return (
    <main className={ui.main}>
      <section className={ui.card} aria-labelledby="forbidden-title">
        <h1 id="forbidden-title" className={ui.title}>
          Không có quyền truy cập
        </h1>
        <p className={ui.muted}>Vai trò của bạn không được phép xem trang này. Liên hệ quản trị viên nếu bạn cần quyền.</p>
        <Link to="/profile" className={ui.secondaryButton}>
          Về trang hồ sơ
        </Link>
      </section>
    </main>
  );
}
