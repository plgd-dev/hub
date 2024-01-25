export const getStatusFromCode = (code: number) => ([67, 68, 69, 95].includes(code) ? 'success' : 'error')
