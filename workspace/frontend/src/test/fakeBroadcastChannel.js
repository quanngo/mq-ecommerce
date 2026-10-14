// In-memory BroadcastChannel: like the real one, a message goes to every
// *other* open channel with the same name, asynchronously. Tests use
// FakeBroadcastChannel.fromOtherTab(name, message) to play another tab.

const open = new Set();

export class FakeBroadcastChannel {
  constructor(name) {
    this.name = name;
    this.onmessage = null;
    this.closed = false;
    open.add(this);
  }

  postMessage(data) {
    if (this.closed) throw new Error('InvalidStateError: channel is closed');
    FakeBroadcastChannel.sent.push({ name: this.name, data });
    for (const ch of open) {
      if (ch !== this && ch.name === this.name) {
        queueMicrotask(() => !ch.closed && ch.onmessage?.({ data }));
      }
    }
  }

  close() {
    this.closed = true;
    open.delete(this);
  }

  /** Messages posted by the app, for assertions. */
  static sent = [];

  static fromOtherTab(name, data) {
    const other = new FakeBroadcastChannel(name);
    other.postMessage(data);
    other.close();
  }

  static reset() {
    FakeBroadcastChannel.sent = [];
    for (const ch of open) ch.closed = true;
    open.clear();
  }
}
