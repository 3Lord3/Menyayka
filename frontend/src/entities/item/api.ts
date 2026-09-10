import { apiClient } from '@shared/api';

import type { Item, ItemPayload, ItemsList } from './model';

// The owner's personal list: arrives together with ARCHIVED — these are exchanged items,
// shown as history (see ITEM_STATUS_META). We request the max page // so nothing is lost
// on "My items" and in the item picker.
export async function fetchItems(): Promise<ItemsList> {
  const { data } = await apiClient.get<ItemsList>('/items', { params: { pageSize: 100 } });
  return data;
}

export async function fetchItem(itemId: number): Promise<Item> {
  const { data } = await apiClient.get<Item>(`/items/${itemId}`);
  return data;
}

// The backend creates an item via JSON and uploads the photo through a separate endpoint. If the
// upload fails the item already exists — we throw ItemImageUploadError with the created item,
// so the form can show the error and direct to editing without losing what was created.
export async function createItem(payload: ItemPayload, image: File | null): Promise<Item> {
  const { data: created } = await apiClient.post<Item>('/items', payload);
  if (!image) return created;

  try {
    return await uploadItemImage(created.id, image);
  } catch (error) {
    throw new ItemImageUploadError(created, error);
  }
}

export async function updateItem(
  itemId: number,
  payload: ItemPayload,
  image: File | undefined,
): Promise<Item> {
  const { data: updated } = await apiClient.patch<Item>(`/items/${itemId}`, payload);
  if (!image) return updated;
  try {
    return await uploadItemImage(itemId, image);
  } catch (error) {
    throw new ItemImageUploadError(updated, error);
  }
}

export async function archiveItem(itemId: number): Promise<void> {
  await apiClient.delete(`/items/${itemId}`);
}

export class ItemImageUploadError extends Error {
  readonly item: Item;

  constructor(item: Item, cause: unknown) {
    super('Товар сохранён, но фото не загрузилось', { cause });
    this.name = 'ItemImageUploadError';
    this.item = item;
  }
}

async function uploadItemImage(itemId: number, image: File): Promise<Item> {
  const form = new FormData();
  form.append('image', image);
  const { data } = await apiClient.post<Item>(`/items/${itemId}/image`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return data;
}
