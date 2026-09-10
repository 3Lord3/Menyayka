const TOKEN_KEY = 'menyayka_token';
const USER_KEY = 'menyayka_user';
const THEME_KEY = 'menyayka_theme';

export const tokenStorage = {
  get(): string | null {
    return localStorage.getItem(TOKEN_KEY);
  },
  set(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
  },
  remove(): void {
    localStorage.removeItem(TOKEN_KEY);
  },
};

// the key is also read by the inline script in index.html (before the first frame) — changing
// the value requires editing both places
export const themeStorage = {
  key: THEME_KEY,
  get(): 'light' | 'dark' | null {
    const raw = localStorage.getItem(THEME_KEY);
    return raw === 'light' || raw === 'dark' ? raw : null;
  },
  set(mode: 'light' | 'dark'): void {
    localStorage.setItem(THEME_KEY, mode);
  },
};

export const userStorage = {
  get<T>(): T | null {
    const raw = localStorage.getItem(USER_KEY);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  },
  set<T>(user: T): void {
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  },
  remove(): void {
    localStorage.removeItem(USER_KEY);
  },
};

// returns false if the record didn't fit the localStorage quota: the caller decides whether
// that's critical
export const jsonStorage = {
  get<T>(key: string): T | null {
    const raw = localStorage.getItem(key);
    if (!raw) return null;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return null;
    }
  },
  set<T>(key: string, value: T): boolean {
    try {
      localStorage.setItem(key, JSON.stringify(value));
      return true;
    } catch {
      return false;
    }
  },
  remove(key: string): void {
    localStorage.removeItem(key);
  },
};
