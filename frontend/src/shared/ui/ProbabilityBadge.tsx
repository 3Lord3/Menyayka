import { theme } from 'antd';

import { probabilityLevel, PROBABILITY_META } from './ProbabilityBadge.model';

import './ui.scss';

interface ProbabilityBadgeProps {
  score: number;
}

// the probability isn't conveyed by color alone — level and percentages are always text
export function ProbabilityBadge({ score }: ProbabilityBadgeProps) {
  const { token } = theme.useToken();
  const meta = PROBABILITY_META[probabilityLevel(score)];
  const color = token[meta.token] as string;
  const percent = Math.round(score * 100);

  return (
    <span
      className="probability-badge"
      style={{ color, backgroundColor: `color-mix(in srgb, ${color} 15%, transparent)` }}
    >
      {meta.label} · {percent}%
    </span>
  );
}
