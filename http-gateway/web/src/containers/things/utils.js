// Returns the extension for resources API for the selected interface
export const interfaceGetParam = currentInterface =>
  currentInterface ? `?interface=${currentInterface}` : ''
