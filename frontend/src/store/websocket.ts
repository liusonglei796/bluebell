import { defineStore } from 'pinia';

export interface Notification {
  type: 'comment' | 'vote' | 'follow' | 'mention' | 'system' | 'online_users';
  title: string;
  content: string;
  data?: Record<string, any>;
  timestamp?: number;
}

interface WebSocketState {
  connected: boolean
  notifications: Notification[]
  onlineUsers: number
}

export const useWebSocketStore = defineStore('websocket', {
  state: (): WebSocketState => ({
    connected: false,
    notifications: [],
    onlineUsers: 0,
  }),

  actions: {
    setConnected(connected: boolean) {
      this.connected = connected;
    },

    addNotification(notification: Notification) {
      this.notifications.unshift({
        ...notification,
        timestamp: Date.now(),
      });
      if (this.notifications.length > 50) {
        this.notifications.pop();
      }
    },

    setOnlineUsers(count: number) {
      this.onlineUsers = count;
    },

    clearNotifications() {
      this.notifications = [];
    },

    removeNotification(index: number) {
      this.notifications.splice(index, 1);
    },
  },
});

class WebSocketService {
  private ws: WebSocket | null = null;
  private url: string = '';
  private token: string = '';
  private _reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;

  
  get reconnectAttempts(): number {
    return this._reconnectAttempts;
  }
  private reconnectDelay: number = 3000;
  private pingInterval: number | null = null;
  private store: ReturnType<typeof useWebSocketStore> | null = null;

  initialize(url: string, token: string, store: ReturnType<typeof useWebSocketStore>) {
    this.url = url;
    this.token = token;
    this.store = store;
  }

  connect() {
    if (!this.token) {
      console.warn('WebSocket: No token available, skipping connection');
      return;
    }

    const wsUrl = `${this.url}?token=${this.token}`;
    console.log('WebSocket: Connecting to', wsUrl);

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket: Connected');
        this._reconnectAttempts = 0;
        this.store?.setConnected(true);
        this.startPing();
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (e) {
          console.error('WebSocket: Failed to parse message', e);
        }
      };

      this.ws.onclose = () => {
        console.log('WebSocket: Disconnected');
        this.store?.setConnected(false);
        this.stopPing();
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket: Error', error);
      };
    } catch (e) {
      console.error('WebSocket: Failed to connect', e);
    }
  }

  private handleMessage(message: { type: string; payload?: any }) {
    switch (message.type) {
      case 'notification':
        this.store?.addNotification(message.payload);
        break;
      case 'pong':
        break;
      case 'online_users':
        if (message.payload?.count !== undefined) {
          this.store?.setOnlineUsers(message.payload.count);
        }
        break;
      default:
        console.log('WebSocket: Unknown message type', message.type);
    }
  }

  private startPing() {
    this.pingInterval = window.setInterval(() => {
      this.send({ type: 'ping' });
    }, 25000);
  }

  private stopPing() {
    if (this.pingInterval !== null) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private attemptReconnect() {
    if (this._reconnectAttempts < this.maxReconnectAttempts) {
      this._reconnectAttempts++;
      console.log(`WebSocket: Reconnecting (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
      setTimeout(() => this.connect(), this.reconnectDelay);
    } else {
      console.log('WebSocket: Max reconnection attempts reached');
    }
  }

  send(data: object) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
    }
  }

  disconnect() {
    this.stopPing();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.store?.setConnected(false);
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

export const wsService = new WebSocketService();
