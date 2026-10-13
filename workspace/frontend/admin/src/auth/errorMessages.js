// Backend error codes (services/auth_errors.go) -> what the user reads.
const MESSAGES = {
  INVALID_CREDENTIALS: 'Email hoặc mật khẩu không đúng.',
  ACCOUNT_LOCKED: 'Tài khoản tạm khoá do đăng nhập sai nhiều lần. Vui lòng thử lại sau 15 phút.',
  INVALID_REQUEST: 'Vui lòng nhập email hợp lệ và mật khẩu.',
  CONSENT_DENIED: 'Bạn đã huỷ đăng nhập bằng Google.',
  DOMAIN_NOT_ALLOWED: 'Chỉ tài khoản Google @mq.com được phép đăng nhập.',
  USER_NOT_ALLOWED: 'Tài khoản chưa được cấp quyền truy cập. Vui lòng liên hệ quản trị viên.',
  EMAIL_NOT_VERIFIED: 'Email Google của bạn chưa được xác minh.',
  INVALID_STATE: 'Phiên đăng nhập đã hết hạn. Vui lòng thử lại.',
  GOOGLE_DISABLED: 'Đăng nhập bằng Google hiện không khả dụng.',
  FORBIDDEN_ORIGIN: 'Yêu cầu không hợp lệ. Vui lòng tải lại trang.',
  NETWORK_ERROR: 'Không kết nối được máy chủ. Kiểm tra mạng và thử lại.',
  EXCHANGE_FAILED: 'Không thể đăng nhập lúc này. Vui lòng thử lại.',
  INVALID_ID_TOKEN: 'Không thể đăng nhập lúc này. Vui lòng thử lại.',
  INTERNAL_ERROR: 'Không thể đăng nhập lúc này. Vui lòng thử lại.',
};

/**
 * @param {string | null | undefined} code
 * @returns {string | null}
 */
export function authErrorMessage(code) {
  if (!code || code === 'UNAUTHENTICATED') return null;
  return MESSAGES[code] ?? 'Đã có lỗi xảy ra. Vui lòng thử lại.';
}
