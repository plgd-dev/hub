// Case insensitive sort function
export const compareIgnoreCase = (a, b) => {
  const item1 = Array.isArray(a) ? a[0] : a
  const item2 = Array.isArray(b) ? b[0] : b

  return item1.localeCompare(item2, 'en', { numeric: true, sensitivity: 'base' })
}
