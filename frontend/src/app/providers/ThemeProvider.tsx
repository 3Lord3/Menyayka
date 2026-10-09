import { useEffect, type ReactNode } from 'react';

import { App as AntApp, ConfigProvider, theme as antdTheme, type ThemeConfig } from 'antd';
import ruRU from 'antd/locale/ru_RU';

import { useAppSelector } from '@app/store/hooks';
import type { ThemeMode } from '@app/store/slices/themeSlice';

import { useReducedMotion } from '@shared/lib/useReducedMotion';

// values match tokens.scss — when the palette changes, edit both places
const baseTokens = {
  colorPrimary: '#1677FF',
  colorSuccess: '#52C41A',
  colorWarning: '#FAAD14',
  colorError: '#FF4D4F',
  borderRadius: 8,
  borderRadiusLG: 12,
  fontFamily: "'Inter', system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif",
  controlHeight: 44,
  controlHeightLG: 52,
} satisfies ThemeConfig['token'];

// colorBgContainer — Input/Select/Card background; colorBgElevated — elevated surfaces such
// as AuthCard: swapped, they kill the field/card contrast
const themeTokens: Record<ThemeMode, ThemeConfig['token']> = {
  light: {
    ...baseTokens,
    colorBgLayout: '#FFFFFF',
    colorBgContainer: '#FFFFFF',
    colorBgElevated: '#F7F9FC',
    colorText: '#000000',
    colorTextSecondary: '#595959',
    colorBorder: '#D9D9D9',
    colorBorderSecondary: '#D9D9D9',
  },
  dark: {
    ...baseTokens,
    colorBgLayout: '#141414',
    colorBgContainer: '#141414',
    colorBgElevated: '#1F1F1F',
    colorText: '#FFFFFF',
    colorTextSecondary: '#A6A6A6',
    colorBorder: '#3A3A3A',
    colorBorderSecondary: '#3A3A3A',
  },
};

const sharedComponents: ThemeConfig['components'] = {
  // Button Label — 16px/600; antd draws 14px by default
  Button: { fontWeight: 600, fontSize: 16 },
  // Body 16px — antd draws 14px input text by default, which centers poorly in a 44px field
  Input: { inputFontSize: 16 },
  // Select uses the global fontSize token (14px), not inputFontSize: keep it at 16 to
  // match Input/TextArea in the same form.
  Select: { fontSize: 16 },
  Segmented: { borderRadius: 8, borderRadiusSM: 4 },
  Card: { borderRadiusLG: 12 },
  Modal: { borderRadiusLG: 12 },
};

// antd uses colorBgLayout for the track and colorBgElevated for the active segment, i.e. the
// reverse — without overrides the active tab blends into the card; translucent skeleton fills
// in dark mode are noticeably lighter than cards and fall off the page background
const themeComponents: Record<ThemeMode, ThemeConfig['components']> = {
  light: {
    ...sharedComponents,
    Segmented: { ...sharedComponents.Segmented, trackBg: '#F7F9FC', itemSelectedBg: '#FFFFFF' },
    Skeleton: { gradientFromColor: '#F7F9FC', gradientToColor: '#D9D9D9' },
  },
  dark: {
    ...sharedComponents,
    Skeleton: { gradientFromColor: '#1F1F1F', gradientToColor: '#3A3A3A' },
  },
};

export function ThemeProvider({ children }: { children: ReactNode }) {
  const mode = useAppSelector((state) => state.theme.mode);
  const isDark = mode === 'dark';
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = mode;
    root.style.colorScheme = mode;
  }, [mode]);

  return (
    <ConfigProvider
      locale={ruRU}
      theme={{
        algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        // antd motion is built on CSSMotion, so the prefers-reduced-motion CSS rule
        // from global.scss can't suppress it — disable it via the seed token
        token: { ...themeTokens[mode], motion: !reducedMotion },
        components: themeComponents[mode],
      }}
    >
      <AntApp>{children}</AntApp>
    </ConfigProvider>
  );
}
