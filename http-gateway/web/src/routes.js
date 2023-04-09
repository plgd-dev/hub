import { Switch, Route, matchPath } from 'react-router-dom'
import { useIntl } from 'react-intl'

import NotFoundPage from '@shared-ui/components/templates/NotFoundPage'

import DevicesListPage from '@/containers/Devices/List/DevicesListPage'
import DevicesDetailsPage from '@/containers/Devices/Detail/DevicesDetailsPage'
import { PendingCommandsListPage } from '@/containers/PendingCommands'
import Notifications from '@/containers/Notifications'
import { messages as t } from './containers/App/App.i18n'

export const menu = [
    {
        title: 'Main menu',
        items: [
            {
                icon: 'devices',
                id: '1',
                title: 'Devices',
                link: '/',
                paths: ['/', '/devices/:id', '/devices/:id/:href*'],
            },
        ],
    },
]

export const mather = (location, item) =>
    matchPath(location, {
        path: item.paths,
        exact: false,
        strict: false,
    })

export const Routes = () => {
    const { formatMessage: _ } = useIntl()
    return (
        <Switch>
            <Route exact component={DevicesListPage} path='/' />
            <Route component={DevicesDetailsPage} path={['/devices/:id', '/devices/:id/:href*']} />
            <Route component={PendingCommandsListPage} path='/pending-commands' />
            <Route path='/notifications'>
                <Notifications />
            </Route>
            <Route path='*'>
                <NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />
            </Route>
        </Switch>
    )
}
