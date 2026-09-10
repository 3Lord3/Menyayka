import { useCallback, useState } from 'react';

import './ui.scss';

interface FadeInImageProps {
  src: string;
  alt: string;
  className?: string;
}

// complete is needed in addition to onLoad: a cached image loaded before the handler
export function FadeInImage({ src, alt, className }: FadeInImageProps) {
  const [image, setImage] = useState({ src, loaded: false });

  // key={src} remounts <img> so the ref callback rechecks complete (cache doesn't wait for onLoad)
  if (image.src !== src) {
    setImage({ src, loaded: false });
  }

  const checkComplete = useCallback(
    (node: HTMLImageElement | null) => {
      if (node?.complete) setImage((prev) => (prev.loaded ? prev : { src, loaded: true }));
    },
    [src],
  );

  return (
    <img
      key={src}
      className={`fade-in-image${image.loaded ? ' fade-in-image--loaded' : ''}${
        className ? ` ${className}` : ''
      }`}
      src={src}
      alt={alt}
      ref={checkComplete}
      onLoad={() => setImage((prev) => ({ ...prev, loaded: true }))}
      // we show a broken URL as-is, not as a transparent placeholder
      onError={() => setImage((prev) => ({ ...prev, loaded: true }))}
    />
  );
}
