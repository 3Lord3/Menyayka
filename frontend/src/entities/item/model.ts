import type { StatusTone } from '@shared/ui';

type ItemStatus = 'ACTIVE' | 'UNAVAILABLE' | 'ARCHIVED';

export interface Item {
  id: number;
  title: string;
  description: string;
  // textual category name from the @shared/config/categories dictionary; items created
  // before the textual-category migration get an empty string
  category: string;
  imageUrl: string | null;
  status: ItemStatus;
  createdAt: string;
  updatedAt: string;
}

export interface ItemPayload {
  title: string;
  description: string;
  category?: string;
}

// the item list page: the server returns total so the client sees pageSize truncation
export interface ItemsList {
  items: Item[];
  total: number;
}

// the list mirrors what the backend accepts
export const ITEM_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const;

// max item-photo size — matches the backend limit (5 MiB)
const ITEM_IMAGE_MAX_SIZE_BYTES = 5 * 1024 * 1024;

// accept filters only the picker dialog: a file can be dropped and then a 422
// arrives after a successful item save
export function getItemImageError(file: { type: string; size: number }): string | null {
  if (!ITEM_IMAGE_TYPES.includes(file.type as (typeof ITEM_IMAGE_TYPES)[number])) {
    return 'Фото должно быть в формате JPG, PNG или WEBP';
  }
  if (file.size > ITEM_IMAGE_MAX_SIZE_BYTES) {
    return 'Фото не больше 5 МБ';
  }
  return null;
}

// ARCHIVED is a completed exchange, not an "archive": deleting an item doesn't create it

export const ITEM_STATUS_META: Record<ItemStatus, { label: string; tone: StatusTone }> = {
  ACTIVE: { label: 'Активен', tone: 'success' },
  UNAVAILABLE: { label: 'Зарезервирован', tone: 'warning' },
  ARCHIVED: { label: 'Обменян', tone: 'neutral' },
};

// an exchanged item stays in the list as history, but it's already given: neither edit
// nor offer it in a new request (the backend rejects mutations of archived ones)
export function isItemExchanged(status: ItemStatus): boolean {
  return status === 'ARCHIVED';
}
