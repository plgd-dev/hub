/**
 * Returns the current mode of the app.
 */
export const getAppMode = () => process?.env?.NODE_ENV || 'production'
