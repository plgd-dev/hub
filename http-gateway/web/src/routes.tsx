import { Routes as RoutesGroup, Route, matchPath } from 'react-router-dom'
import { useIntl } from 'react-intl'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import { IconDevices, IconSettings } from '@shared-ui/components/Atomic/Icon/'

import DevicesListPage from '@/containers/Devices/List/DevicesListPage'
import DevicesDetailsPage from '@/containers/Devices/Detail/DevicesDetailsPage'
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
                paths: ['/', '/devices/:id', '/devices/:id/resources', '/devices/:id/resources/:href'],
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

export const mather = (pathname: string, pattern: string) => matchPath(pattern, pathname)

export const Routes = () => {
    const { formatMessage: _ } = useIntl()
    return (
        <RoutesGroup>
            <Route element={<DevicesListPage />} path='/' />
            <Route element={<DevicesDetailsPage defaultActiveTab={0} />} path='/devices/:id' />
            <Route element={<DevicesDetailsPage defaultActiveTab={1} />} path='/devices/:id/resources' />
            <Route element={<DevicesDetailsPage defaultActiveTab={1} />} path='/devices/:id/resources/*' />
            {process.env?.REACT_APP_TEST_VIEW === 'true' && <Route element={<TestPage />} path='/test' />}
            <Route element={<NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />} path='*' />
        </RoutesGroup>
    )
}
