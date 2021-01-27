export const getValue = (value, emptyPlaceholder = '-') =>
  value !== undefined && value !== null ? value : emptyPlaceholder
