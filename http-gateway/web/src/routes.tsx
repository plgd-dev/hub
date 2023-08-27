import { Routes as RoutesGroup, Route, matchPath } from 'react-router-dom'
import { useIntl } from 'react-intl'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import {
    IconDevices,
    IconSettings,
    IconDashboard,
    IconIntegrations,
    IconRemoteClients,
    IconPendingCommands,
    IconNetwork,
    IconDeviceUpdate,
    IconLog,
    IconLock,
    IconNet,
    IconDocs,
    IconChat,
} from '@shared-ui/components/Atomic/Icon/'
import { MenuGroup } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import MockApp from '@shared-ui/app/clientApp/MockApp'

import DevicesListPage from '@/containers/Devices/List/DevicesListPage'
import DevicesDetailsPage from '@/containers/Devices/Detail/DevicesDetailsPage'
import { messages as t } from './containers/App/App.i18n'
import TestPage from './containers/Test'
import RemoteClientsListPage from '@/containers/RemoteClients/List/RemoteClientsListPage'
import RemoteClientDetailPage from '@/containers/RemoteClients/Detail/RemoteClientDetailPage'
import RemoteClientDevicesDetailPage from '@/containers/RemoteClients/Device/Detail/RemoteClientDevicesDetailPage'
import testId from '@/testId'

const MenuTranslate = (props: { id: string }) => {
    const { id } = props
    const { formatMessage: _ } = useIntl()

    // @ts-ignore
    return <span>{_(t[id])}</span>
}

export const menu: MenuGroup[] = [
    {
        title: <MenuTranslate id='menuMainMenu' />,
        items: [
            {
                icon: <IconDashboard />,
                id: '0',
                title: <MenuTranslate id='menuDashboard' />,
                link: '/dashboard',
                paths: ['/dashboard'],
                exact: true,
                disabled: true,
            },
            {
                icon: <IconDevices />,
                id: '1',
                title: <MenuTranslate id='menuDevices' />,
                link: '/',
                paths: ['/', '/devices/:id', '/devices/:id/resources', '/devices/:id/resources/:href'],
                exact: true,
                dataTestId: testId.menu.devices,
            },
            {
                icon: <IconIntegrations />,
                id: '2',
                title: <MenuTranslate id='menuIntegrations' />,
                link: '/integrations',
                paths: ['/integrations'],
                exact: true,
                disabled: true,
            },
            {
                icon: <IconRemoteClients />,
                id: '3',
                title: <MenuTranslate id='menuRemoteClients' />,
                link: '/remote-clients',
                paths: [
                    '/remote-clients',
                    '/remote-clients/:id',
                    '/remote-clients/:id/devices/:deviceId',
                    '/remote-clients/:id/devices/:deviceId/resources',
                    '/remote-clients/:id/devices/:deviceId/resources/:href',
                ],
                exact: true,
            },
            {
                icon: <IconPendingCommands />,
                id: '4',
                title: <MenuTranslate id='menuPendingCommands' />,
                link: '/pending-commands',
                paths: ['/pending-commands'],
                exact: true,
                disabled: true,
            },
        ],
    },
    {
        title: <MenuTranslate id='menuOthers' />,
        items: [
            {
                icon: <IconNetwork />,
                id: '10',
                title: <MenuTranslate id='menuDeviceProvisioning' />,
                link: '/device-provisioning',
                paths: ['/device-provisioning'],
                exact: true,
                disabled: true,
                children: [
                    { id: '101', title: <MenuTranslate id='menuQuickstart' />, tag: { variant: 'success', text: 'New' } },
                    { id: '102', title: <MenuTranslate id='menuManageEnrollments' /> },
                    { id: '103', title: <MenuTranslate id='menuLinkedHubs' /> },
                    { id: '104', title: <MenuTranslate id='menuCertificates' />, tag: { variant: 'info', text: 'Soon!' } },
                    { id: '105', title: <MenuTranslate id='menuRegistrationRecords' /> },
                ],
            },
            {
                icon: <IconDeviceUpdate />,
                id: '11',
                title: <MenuTranslate id='menuDeviceFirmwareUpdate' />,
                link: '/device-firmware-update',
                paths: ['/device-firmware-update'],
                exact: true,
                disabled: true,
                children: [
                    { id: '111', title: 'Quickstart 2', tag: { variant: 'success', text: 'New 2' } },
                    { id: '112', title: 'Manage enrollments 2' },
                    { id: '113', title: 'Linked hubs 2' },
                    { id: '114', title: 'Certificates 2', tag: { variant: 'info', text: 'Soon!' } },
                    { id: '115', title: 'Registration records 2' },
                ],
            },
            {
                icon: <IconLog />,
                id: '12',
                title: <MenuTranslate id='menuDeviceLogs' />,
                link: '/device-firmware-update',
                paths: ['/device-firmware-update'],
                exact: true,
                disabled: true,
            },
            {
                icon: <IconLock />,
                id: '13',
                title: <MenuTranslate id='menuApiTokens' />,
                link: '/api-tokens',
                paths: ['/api-tokens'],
                exact: true,
                disabled: true,
            },
            {
                icon: <IconNet />,
                id: '14',
                title: <MenuTranslate id='menuSchemaHub' />,
                link: '/schema-hub',
                paths: ['/schema-hub'],
                exact: true,
                disabled: true,
            },
        ],
    },
    {
        title: <MenuTranslate id='menuSupport' />,
        items: [
            {
                icon: <IconDocs />,
                id: '20',
                title: <MenuTranslate id='menuDocs' />,
                link: '/docs',
                paths: ['/docs'],
                exact: true,
                disabled: true,
            },
            {
                icon: <IconChat />,
                id: '21',
                title: <MenuTranslate id='menuChatRoom' />,
                link: '/chat-room',
                paths: ['/hat-room'],
                exact: true,
                disabled: true,
            },
        ],
    },
]

if (process.env?.REACT_APP_TEST_VIEW === 'true') {
    menu[0]?.items?.push({
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

            <Route element={<RemoteClientsListPage />} path='/remote-clients' />
            <Route element={<RemoteClientDetailPage />} path='/remote-clients/:id' />
            <Route element={<RemoteClientDevicesDetailPage defaultActiveTab={0} />} path='/remote-clients/:id/devices/:deviceId' />
            <Route element={<RemoteClientDevicesDetailPage defaultActiveTab={1} />} path='/remote-clients/:id/devices/:deviceId/resources' />
            <Route element={<RemoteClientDevicesDetailPage defaultActiveTab={1} />} path='/remote-clients/:id/devices/:deviceId/resources/*' />

            <Route element={<MockApp />} path='/devices-code-redirect' />

            {process.env?.REACT_APP_TEST_VIEW === 'true' && <Route element={<TestPage />} path='/test' />}
            <Route element={<NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />} path='*' />
        </RoutesGroup>
    )
}
