import { Routes as RoutesGroup, Route, matchPath } from 'react-router-dom'
import { useIntl } from 'react-intl'
import React, { lazy, Suspense } from 'react'

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
    IconCertificate,
} from '@shared-ui/components/Atomic/Icon/'
import { MenuGroup } from '@shared-ui/components/Layout/LeftPanel/LeftPanel.types'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'

import { messages as t } from './containers/App/App.i18n'
import { messages as g } from './containers/Global.i18n'
import testId from '@/testId'
import EnrollmentGroupsDetailPage from '@/containers/DeviceProvisioning/EnrollmentGroups/DetailPage/EnrollmentGroupsDetailPage'

// Devices
const DevicesListPage = lazy(() => import('./containers/Devices/List/DevicesListPage'))
const DevicesDetailsPage = lazy(() => import('./containers/Devices/Detail/DevicesDetailsPage'))

// Remote Clients
const RemoteClientsListPage = lazy(() => import('./containers/RemoteClients/List/RemoteClientsListPage'))
const RemoteClientDetailPage = lazy(() => import('./containers/RemoteClients/Detail/RemoteClientDetailPage'))
const RemoteClientDevicesDetailPage = lazy(() => import('./containers/RemoteClients/Device/Detail/RemoteClientDevicesDetailPage'))

// DPS
const ProvisioningRecordsListPage = lazy(() => import('./containers/DeviceProvisioning/ProvisioningRecords/ListPage/ProvisioningRecordsListPage'))
const ProvisioningRecordsDetailPage = lazy(() => import('@/containers/DeviceProvisioning/ProvisioningRecords/DetailPage/ProvisioningRecordsDetailPage'))
const EnrollmentGroupsListPage = lazy(() => import('./containers/DeviceProvisioning/EnrollmentGroups/ListPage'))
const LinkedHubsListPage = lazy(() => import('./containers/DeviceProvisioning/LinkedHubs'))
const LinkedHubsDetailPage = lazy(() => import('./containers/DeviceProvisioning/LinkedHubs/DetailPage'))
const CertificatesListPage = lazy(() => import('./containers/Certificates'))
const CertificatesDetailPage = lazy(() => import('@/containers/Certificates/DetailPage'))

// Pending commands
const PendingCommandsListPage = lazy(() => import('./containers/PendingCommands/PendingCommandsListPage'))

// Internal
const MockApp = lazy(() => import('@shared-ui/app/clientApp/MockApp'))
const ConfigurationPage = lazy(() => import('./containers/Configuration'))

const MenuTranslate = (props: { id: string }) => {
    const { id } = props
    const { formatMessage: _ } = useIntl()

    // @ts-ignore
    return <span>{_(t[id])}</span>
}

export const defaultMenu = {
    devices: true,
    configuration: true,
    remoteClients: true,
    pendingCommands: true,
    certificates: true,
    deviceProvisioning: true,
    docs: true,
    chatRoom: true,
}

export const getMenu = (menuConfig: any): MenuGroup[] => [
    {
        title: <MenuTranslate id='menuMainMenu' />,
        items: [
            {
                icon: <IconDashboard />,
                id: '0',
                title: <MenuTranslate id='menuDashboard' />,
                link: '/',
                paths: ['/'],
                exact: true,
                visibility: menuConfig?.dashboard === false ? false : 'disabled',
            },
            {
                icon: <IconDevices />,
                id: '1',
                title: <MenuTranslate id='menuDevices' />,
                link: '/devices',
                paths: ['/devices', '/devices/:id', '/devices/:id/resources', '/devices/:id/resources/:href', '/devices/:id/certificates', '/devices/:id/dps'],
                exact: true,
                dataTestId: testId.menu.devices,
                visibility: menuConfig.devices,
            },
            {
                icon: <IconIntegrations />,
                id: '2',
                title: <MenuTranslate id='menuIntegrations' />,
                link: '/integrations',
                paths: ['/integrations'],
                exact: true,
                visibility: menuConfig.integrations === false ? false : 'disabled',
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
                    '/remote-clients/:id/configuration',
                ],
                exact: true,
                visibility: menuConfig.remoteClients,
            },
            {
                icon: <IconPendingCommands />,
                id: '4',
                title: <MenuTranslate id='menuPendingCommands' />,
                link: '/pending-commands',
                paths: ['/pending-commands'],
                exact: true,
                visibility: menuConfig.pendingCommands,
            },
            {
                icon: <IconCertificate />,
                id: '5',
                title: <MenuTranslate id='menuCertificates' />,
                link: '/certificates',
                paths: ['/certificates', '/certificates/:certificateId'],
                exact: true,
                visibility: menuConfig.certificates,
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
                visibility: menuConfig.deviceProvisioning,
                children: [
                    {
                        id: '101',
                        title: <MenuTranslate id='enrollmentGroups' />,
                        link: '/enrollment-groups',
                        paths: ['/device-provisioning/enrollment-groups', '/device-provisioning/enrollment-groups/:enrollmentId'],
                    },
                    {
                        id: '102',
                        title: <MenuTranslate id='provisioningRecords' />,
                        link: '/provisioning-records',
                        paths: ['/device-provisioning/provisioning-records', '/device-provisioning/provisioning-records/:recordId'],
                    },
                    {
                        id: '103',
                        title: <MenuTranslate id='menuLinkedHubs' />,
                        link: '/linked-hubs',
                        paths: ['/device-provisioning/linked-hubs', '/device-provisioning/linked-hubs/:hubId'],
                    },
                ],
            },
            {
                icon: <IconDeviceUpdate />,
                id: '11',
                title: <MenuTranslate id='menuDeviceFirmwareUpdate' />,
                link: '/device-firmware-update',
                paths: ['/device-firmware-update'],
                exact: true,
                visibility: menuConfig.deviceFirmwareUpdate === false ? false : 'disabled',
                children: [
                    { id: '111', title: 'Quickstart 2' },
                    { id: '112', title: 'Manage enrollments 2' },
                    { id: '113', title: 'Linked hubs 2' },
                    { id: '114', title: 'Certificates 2' },
                    { id: '115', title: 'Registration records 2' },
                ],
            },
            {
                icon: <IconLog />,
                id: '12',
                title: <MenuTranslate id='menuDeviceLogs' />,
                link: '/device-logs',
                paths: ['/device-logs'],
                exact: true,
                visibility: menuConfig.deviceLogs === false ? false : 'disabled',
            },
            {
                icon: <IconLock />,
                id: '13',
                title: <MenuTranslate id='menuApiTokens' />,
                link: '/api-tokens',
                paths: ['/api-tokens'],
                exact: true,
                visibility: menuConfig.apiTokens === false ? false : 'disabled',
            },
            {
                icon: <IconNet />,
                id: '14',
                title: <MenuTranslate id='menuSchemaHub' />,
                link: '/schema-hub',
                paths: ['/schema-hub'],
                exact: true,
                visibility: menuConfig.schemaHub === false ? false : 'disabled',
            },
            {
                icon: <IconSettings />,
                id: '15',
                title: <MenuTranslate id='menuConfiguration' />,
                link: '/configuration',
                paths: ['/configuration', '/configuration/theme-generator'],
                exact: true,
                visibility: menuConfig.configuration,
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
                link: '//docs.plgd.dev',
                visibility: menuConfig.docs,
            },
            {
                icon: <IconChat />,
                id: '21',
                title: <MenuTranslate id='menuChatRoom' />,
                link: '//discord.com/channels/978923432056066059/978923432836218882',
                visibility: menuConfig.chatRoom,
            },
        ],
    },
]

export const mather = (pathname: string, pattern: string) => matchPath(pattern, pathname)

const Loader = () => {
    const { formatMessage: _ } = useIntl()
    return <FullPageLoader i18n={{ loading: _(g.loading) }} />
}

export const Routes = () => {
    const { formatMessage: _ } = useIntl()
    return (
        <RoutesGroup>
            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesListPage />
                    </Suspense>
                }
                path='/devices'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesDetailsPage defaultActiveTab={0} />
                    </Suspense>
                }
                path='/devices/:id'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesDetailsPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/devices/:id/resources'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesDetailsPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/devices/:id/resources/*'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesDetailsPage defaultActiveTab={2} />
                    </Suspense>
                }
                path='/devices/:id/certificates'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <DevicesDetailsPage defaultActiveTab={3} />
                    </Suspense>
                }
                path='/devices/:id/dps'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientsListPage />
                    </Suspense>
                }
                path='/remote-clients'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientDetailPage />
                    </Suspense>
                }
                path='/remote-clients/:id'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientDetailPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/remote-clients/:id/configuration'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientDevicesDetailPage defaultActiveTab={0} />
                    </Suspense>
                }
                path='/remote-clients/:id/devices/:deviceId'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientDevicesDetailPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/remote-clients/:id/devices/:deviceId/resources'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <RemoteClientDevicesDetailPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/remote-clients/:id/devices/:deviceId/resources/*'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <CertificatesListPage />
                    </Suspense>
                }
                path='/certificates'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <CertificatesDetailPage />
                    </Suspense>
                }
                path='/certificates/:certificateId'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <EnrollmentGroupsListPage />
                    </Suspense>
                }
                path='/device-provisioning/enrollment-groups'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <EnrollmentGroupsDetailPage />
                    </Suspense>
                }
                path='/device-provisioning/enrollment-groups/:enrollmentId'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <ProvisioningRecordsListPage />
                    </Suspense>
                }
                path='/device-provisioning/provisioning-records'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <ProvisioningRecordsDetailPage />
                    </Suspense>
                }
                path='/device-provisioning/provisioning-records/:recordId'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <LinkedHubsListPage />
                    </Suspense>
                }
                path='/device-provisioning/linked-hubs'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <LinkedHubsDetailPage />
                    </Suspense>
                }
                path='/device-provisioning/linked-hubs/:hubId'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <PendingCommandsListPage />
                    </Suspense>
                }
                path='/pending-commands'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <MockApp />
                    </Suspense>
                }
                path='/devices-code-redirect/*'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <ConfigurationPage />
                    </Suspense>
                }
                path='/configuration'
            />

            <Route
                element={
                    <Suspense fallback={<Loader />}>
                        <ConfigurationPage defaultActiveTab={1} />
                    </Suspense>
                }
                path='/configuration/theme-generator'
            />

            <Route element={<NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />} path='*' />
        </RoutesGroup>
    )
}
