export const getClientIp = (clientIp: string) => (clientIp.endsWith('/') ? clientIp.slice(0, -1) : clientIp)
