import type { GlobalToken } from 'antd';

type ProbabilityLevel = 'high' | 'medium' | 'low';

type ProbabilityMeta = {
  label: string;
  token: keyof GlobalToken;
};

export const PROBABILITY_META: Record<ProbabilityLevel, ProbabilityMeta> = {
  high: { label: 'Высокая', token: 'colorSuccess' },
  medium: { label: 'Средняя', token: 'colorWarning' },
  low: { label: 'Низкая', token: 'colorError' },
};

// Chain success-probability level: score is a fraction 0..1, thresholds 0.8/0.6
export function probabilityLevel(score: number): ProbabilityLevel {
  if (score >= 0.8) return 'high';
  if (score >= 0.6) return 'medium';
  return 'low';
}
