export interface ParticipantAlias {
  name: string;
  emoji: string;
}

// chain participants are anonymous: instead of names — an animal alias with an emoji
const ALIASES: ParticipantAlias[] = [
  { name: 'Мишка', emoji: '🐻' },
  { name: 'Лиса', emoji: '🦊' },
  { name: 'Кот', emoji: '🐱' },
  { name: 'Волк', emoji: '🐺' },
  { name: 'Заяц', emoji: '🐰' },
  { name: 'Панда', emoji: '🐼' },
  { name: 'Тигр', emoji: '🐯' },
  { name: 'Лев', emoji: '🦁' },
  { name: 'Ёж', emoji: '🦔' },
  { name: 'Сова', emoji: '🦉' },
  { name: 'Пингвин', emoji: '🐧' },
  { name: 'Олень', emoji: '🦌' },
];

// by position, not by user id: positions within a chain are unique, so two equal animals
// can't occur. Positions are 0-based; the ring's first position is "Teddy"
export function participantAlias(position: number): ParticipantAlias {
  const index = Math.trunc(position) % ALIASES.length;
  return ALIASES[index < 0 ? index + ALIASES.length : index];
}
