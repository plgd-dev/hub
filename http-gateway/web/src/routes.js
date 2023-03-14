import { Switch, Route } from 'react-router-dom'
import DevicesListPage from '@/containers/Devices/List/DevicesListPage'
import DevicesDetailsPage from '@/containers/Devices/Detail/DevicesDetailsPage'
import { PendingCommandsListPage } from '@/containers/PendingCommands'
import Notifications from '@/containers/Notifications'
import NotFoundPage from '@/containers/NotFoundPage'

export const menu = [
    {
        title: 'Main menu',
        items: [
            {
                icon: 'devices',
                id: '1',
                title: 'Devices',
                link: '/',
            },
        ],
    },
]

export const Routes = () => (
    <Switch>
        <Route exact component={DevicesListPage} path='/' />
        <Route component={DevicesDetailsPage} path={['/devices/:id', '/devices/:id/:href*']} />
        <Route component={PendingCommandsListPage} path='/pending-commands' />
        <Route path='/notifications'>
            <Notifications />
        </Route>
        <Route path='*'>
            <NotFoundPage />
        </Route>
    </Switch>
)
