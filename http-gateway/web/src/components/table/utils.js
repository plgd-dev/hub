// Case insensitive sort function
export const compareIgnoreCase = (a, b) =>
  a.localeCompare(b, 'en', { numeric: true, sensitivity: 'base' })
