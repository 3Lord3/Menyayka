import { useState } from 'react';

import { jsonStorage } from '@shared/lib/storage';

const MAX_SIDE = 512;
const JPEG_QUALITY = 0.7;

function photoKey(chainId: number, requestId: number): string {
  return `deal:photo:${chainId}:${requestId}`;
}

// Compress the package photo before sending: canvas shrinks the file to 512px on the long
// side and outputs JPEG 0.7 — otherwise the shot won't fit the localStorage quota
async function compressImageFile(file: File): Promise<string> {
  const url = URL.createObjectURL(file);
  try {
    const image = await loadImage(url);
    const scale = Math.min(1, MAX_SIDE / Math.max(image.width, image.height));
    const canvas = document.createElement('canvas');
    canvas.width = Math.max(1, Math.round(image.width * scale));
    canvas.height = Math.max(1, Math.round(image.height * scale));
    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('canvas is not supported');
    ctx.drawImage(image, 0, 0, canvas.width, canvas.height);
    return canvas.toDataURL('image/jpeg', JPEG_QUALITY);
  } finally {
    URL.revokeObjectURL(url);
  }
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error('image failed to load'));
    image.src = src;
  });
}

// the localStorage quota may run out: then the photo stays only in page state,
// but we still enable the send button — silently
export function useDealPhoto(chainId: number, requestId: number) {
  const key = photoKey(chainId, requestId);
  const [photo, setPhoto] = useState<string | null>(() => jsonStorage.get<string>(key));

  const setPhotoFile = async (file: File) => {
    const dataUrl = await compressImageFile(file);
    jsonStorage.set(key, dataUrl);
    setPhoto(dataUrl);
  };

  const removePhoto = () => {
    jsonStorage.remove(key);
    setPhoto(null);
  };

  return { photo, setPhotoFile, removePhoto, isPhotoSet: photo !== null };
}
