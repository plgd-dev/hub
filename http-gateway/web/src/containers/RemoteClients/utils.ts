export const getClientUrl = (clientUrl: string) => (clientUrl.endsWith('/') ? clientUrl.slice(0, -1) : clientUrl)
