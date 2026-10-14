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

// Codes from user management, password changes and permission checks.
const ACTION_MESSAGES = {
  FORBIDDEN: 'Bạn không có quyền thực hiện thao tác này.',
  NOT_FOUND: 'Không tìm thấy người dùng.',
  WEAK_PASSWORD: 'Mật khẩu mới quá ngắn (hoặc trùng mật khẩu cũ).',
  WRONG_PASSWORD: 'Mật khẩu hiện tại không đúng.',
  EMAIL_TAKEN: 'Email này đã có tài khoản.',
  INVALID_ROLE: 'Vai trò không hợp lệ.',
  LAST_ADMIN: 'Không thể thực hiện: đây là quản trị viên cuối cùng có quyền quản lý người dùng.',
  CANNOT_MODIFY_SELF: 'Bạn không thể tự đổi vai trò, khoá hoặc đặt lại mật khẩu của chính mình ở đây.',
  INVALID_REQUEST: 'Dữ liệu không hợp lệ. Vui lòng kiểm tra lại.',
  NETWORK_ERROR: 'Không kết nối được máy chủ. Kiểm tra mạng và thử lại.',
};

/**
 * Message for an error from a non-login action (user management, password change).
 * @param {string | null | undefined} code
 * @returns {string}
 */
export function actionErrorMessage(code) {
  return ACTION_MESSAGES[code] ?? 'Đã có lỗi xảy ra. Vui lòng thử lại.';
}

// Codes from the Orders pages that read differently from user management.
const ORDER_MESSAGES = {
  NOT_FOUND: 'Không tìm thấy đơn hàng.',
  INVALID_REQUEST: 'Bộ lọc không hợp lệ. Kiểm tra lại ngày (từ ngày phải trước đến ngày) và từ khoá.',
  FORBIDDEN: 'Bạn không có quyền xem nội dung này.',
};

/**
 * Message for an error from the Orders pages.
 * @param {string | null | undefined} code
 * @returns {string}
 */
export function orderErrorMessage(code) {
  return ORDER_MESSAGES[code] ?? actionErrorMessage(code);
}

/** Vietnamese labels for permission codes (models.Perm* in the backend). */
export const PERMISSION_LABELS = {
  'users.read': 'Xem người dùng',
  'users.manage': 'Quản lý người dùng',
  'orders.read': 'Xem đơn hàng',
  'orders.detail': 'Xem chi tiết đơn hàng',
  'orders.write': 'Cập nhật đơn hàng',
};

/**
 * @param {string | null | undefined} code
 * @returns {string | null}
 */
export function authErrorMessage(code) {
  if (!code || code === 'UNAUTHENTICATED') return null;
  return MESSAGES[code] ?? 'Đã có lỗi xảy ra. Vui lòng thử lại.';
}
