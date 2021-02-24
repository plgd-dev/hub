/**
 * Translates the Toast if the given string is a translate object,
 * otherwise return the given string.
 * @param data
 * @param _ - formattedMessage
 * @returns {*}
 */
export const translateToastString = (data, _) => {
  if (!data) return null

  // If is component
  if (data?.props) {
    return data
  }

  // If its an object it can contain .message, meaning it is a nested translate object
  if (typeof data === 'object') {
    return data.message ? _(data.message, data.params) : _(data)
  }

  return data
}
