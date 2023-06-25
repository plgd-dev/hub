import { Switch, Route, matchPath } from 'react-router-dom'
import { useIntl } from 'react-intl'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import { MenuItem } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import { IconDevices, IconSettings } from '@shared-ui/components/Atomic/Icon/'

import DevicesListPage from '@/containers/Devices/List/DevicesListPage'
import DevicesDetailsPage from '@/containers/Devices/Detail/DevicesDetailsPage'
import { PendingCommandsListPage } from '@/containers/PendingCommands'
import Notifications from '@/containers/Notifications'
import { messages as t } from './containers/App/App.i18n'
import TestPage from './containers/Test'

export const menu = [
    {
        title: 'Main menu',
        items: [
            {
                icon: <IconDevices />,
                id: '1',
                title: 'Devices',
                link: '/',
                paths: ['/', '/devices/:id', '/devices/:id/:href'],
                exact: true,
            },
        ],
    },
]

if (process.env?.REACT_APP_TEST_VIEW === 'true') {
    menu[0].items.push({
        icon: <IconSettings />,
        id: '999',
        title: 'Test',
        link: '/test',
        paths: ['/test'],
        exact: false,
    })
}

export const mather = (location: string, item: MenuItem) =>
    matchPath(location, {
        path: item.paths,
        exact: item.exact || false,
        strict: false,
    })

export const Routes = () => {
    const { formatMessage: _ } = useIntl()
    return (
        <Switch>
            <Route exact component={DevicesListPage} path='/' />
            <Route component={DevicesDetailsPage} path={['/devices/:id/:href*']} />
            <Route component={PendingCommandsListPage} path='/pending-commands' />
            {process.env?.REACT_APP_TEST_VIEW === 'true' && <Route component={TestPage} path='/test' />}
            <Route path='/notifications'>
                <Notifications />
            </Route>
            <Route path='*'>
                <NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />
            </Route>
        </Switch>
    )
}
