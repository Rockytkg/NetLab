const AVATAR_COLORS = [
  '#1677ff',
  '#08979c',
  '#722ed1',
  '#c41d7f',
  '#d4380d',
  '#d46b08',
  '#389e0d',
]

export function getAvatarColor(nickname?: string): string {
  const value = nickname?.trim() || '-'
  let hash = 0

  for (const character of value) {
    hash = (hash * 31 + character.codePointAt(0)!) >>> 0
  }

  return AVATAR_COLORS[hash % AVATAR_COLORS.length]
}
