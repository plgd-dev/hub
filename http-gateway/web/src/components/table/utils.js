// Case insensitive sort function
export const compareIgnoreCase = (a, b) => {
  let r1 = a.toLowerCase()
  let r2 = b.toLowerCase()
  if (r1 < r2) {
    return -1
  }
  if (r1 > r2) {
    return 1
  }
  return 0
}
