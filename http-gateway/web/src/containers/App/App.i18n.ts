import { defineMessages } from '@formatjs/intl'

export const messages = defineMessages({
    loading: {
        id: 'app.loading',
        defaultMessage: 'Loading',
    },
    authError: {
        id: 'app.authError',
        defaultMessage: 'Authorization server error',
    },
    pageTitle: {
        id: 'not-found-page.pageTitle',
        defaultMessage: 'Page not found',
    },
    notFoundPageDefaultMessage: {
        id: 'not-found-page.notFoundPageDefaultMessage',
        defaultMessage: 'The page you are looking for does not exist.',
    },
    notifications: {
        id: 'app.notifications',
        defaultMessage: 'Notifications',
    },
    noNotifications: {
        id: 'app.noNotifications',
        defaultMessage: 'Empty Notifications',
    },
    markAllAsRead: {
        id: 'app.markAllAsRead',
        defaultMessage: 'mark all as read',
    },
    version: {
        id: 'app.version',
        defaultMessage: 'Version',
    },
    newUpdateIsAvailable: {
        id: 'app.newUpdateIsAvailable',
        defaultMessage: 'New update is available.',
    },
})
