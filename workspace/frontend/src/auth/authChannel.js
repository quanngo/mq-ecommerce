// Cross-tab auth notifications. Messages carry no data (no user, no token):
// only 'login' or 'logout', so a receiving tab asks the server itself.

const CHANNEL_NAME = 'mq-admin-auth';
const MESSAGES = new Set(['login', 'logout']);

/**
 * @param {(message: 'login' | 'logout') => void} onMessage  called for messages from *other* tabs
 * @returns {{ post: (message: 'login' | 'logout') => void, close: () => void }}
 */
export function createAuthChannel(onMessage) {
  if (typeof BroadcastChannel === 'undefined') {
    return { post() {}, close() {} }; // very old browsers: tabs catch up on their next API call
  }
  const channel = new BroadcastChannel(CHANNEL_NAME);
  channel.onmessage = (event) => {
    if (MESSAGES.has(event.data)) onMessage(event.data);
  };
  return {
    post(message) {
      channel.postMessage(message);
    },
    close() {
      channel.close();
    },
  };
}
