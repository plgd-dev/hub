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
import { useTheme } from '@emotion/react'

// Devices
const DevicesListPage = lazy(() => import('./containers/Devices/List/DevicesListPage'))
const DevicesDetailsPage = lazy(() => import('./containers/Devices/Detail/DevicesDetailsPage'))

// Remote Clients
const RemoteClientsListPage = lazy(() => import('./containers/RemoteClients/List/RemoteClientsListPage'))
const RemoteClientDetailPage = lazy(() => import('./containers/RemoteClients/Detail/RemoteClientDetailPage'))
const RemoteClientDevicesDetailPage = lazy(() => import('./containers/RemoteClients/Device/Detail/RemoteClientDevicesDetailPage'))

// DPS
// Provisioning Records
const ProvisioningRecordsListPage = lazy(() => import('./containers/DeviceProvisioning/ProvisioningRecords/ListPage/ProvisioningRecordsListPage'))
const ProvisioningRecordsDetailPage = lazy(() => import('@/containers/DeviceProvisioning/ProvisioningRecords/DetailPage/ProvisioningRecordsDetailPage'))

// EnrollmentGroups
const EnrollmentGroupsListPage = lazy(() => import('./containers/DeviceProvisioning/EnrollmentGroups/ListPage'))
const EnrollmentGroupsDetailPage = lazy(() => import('./containers/DeviceProvisioning/EnrollmentGroups/DetailPage'))
const NewEnrollmentGroupsPage = lazy(() => import('./containers/DeviceProvisioning/EnrollmentGroups/NewEnrollmentGroupsPage'))

// Linked Hubs
const LinkedHubsListPage = lazy(() => import('./containers/DeviceProvisioning/LinkedHubs'))
const LinkedHubsDetailPage = lazy(() => import('./containers/DeviceProvisioning/LinkedHubs/DetailPage'))
const LinkNewHubPage = lazy(() => import('./containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage'))

// Certificates
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

export const pages = {
    CONFIGURATION: {
        LINK: '/configuration',
        THEME_GENERATOR: '/configuration/theme-generator',
    },
    DEVICES: {
        LINK: '/devices',
        DETAIL: {
            LINK: '/devices/:id/:tab/:section',
            TABS: ['', 'resources', 'certificates', 'dps'],
            SECTIONS: ['', 'credentials', 'acls'],
        },
    },
    DPS: {
        LINK: '/device-provisioning',
        LINKED_HUBS: {
            LINK: '/device-provisioning/linked-hubs',
            DETAIL: {
                LINK: '/device-provisioning/linked-hubs/:hubId/:tab/:section',
                TABS: ['', 'certificate-authority', 'authorization'],
            },
            ADD: {
                LINK: '/device-provisioning/linked-hubs/link-new-hub/:step',
                TABS: ['', 'hub-detail', 'certificate-authority', 'authorization'],
            },
        },
        PROVISIONING_RECORDS: {
            LINK: '/device-provisioning/provisioning-records',
            DETAIL: '/device-provisioning/provisioning-records/:recordId/:tab',
            TABS: ['', 'credentials', 'acls'],
        },
        ENROLLMENT_GROUPS: {
            LINK: '/device-provisioning/enrollment-groups',
            NEW: {
                LINK: '/device-provisioning/enrollment-groups/new-enrollment-group/:step',
                TABS: ['', 'device-authentication', 'device-credentials'],
            },
            DETAIL: '/device-provisioning/enrollment-groups/:enrollmentId',
        },
    },
    CERTIFICATES: {
        LINK: '/certificates',
        DETAIL: '/certificates/:certificateId',
    },
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
                paths: [
                    '/devices',
                    '/devices/:id',
                    '/devices/:id/resources',
                    '/devices/:id/resources/*',
                    '/devices/:id/certificates',
                    '/devices/:id/dps',
                    '/devices/:id/dps/:section',
                ],
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
                link: pages.CERTIFICATES.LINK,
                paths: [pages.CERTIFICATES.LINK, pages.CERTIFICATES.DETAIL],
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
                link: pages.DPS.LINK,
                paths: [pages.DPS.LINK],
                exact: true,
                visibility: menuConfig.deviceProvisioning,
                children: [
                    {
                        id: '101',
                        title: <MenuTranslate id='menuLinkedHubs' />,
                        link: '/linked-hubs',
                        paths: [
                            pages.DPS.LINKED_HUBS.LINK,
                            '/device-provisioning/linked-hubs/:hubId',
                            '/device-provisioning/linked-hubs/:hubId/:tab',
                            pages.DPS.LINKED_HUBS.DETAIL.LINK,
                            pages.DPS.LINKED_HUBS.ADD.LINK,
                        ],
                    },
                    {
                        id: '102',
                        title: <MenuTranslate id='enrollmentGroups' />,
                        link: '/enrollment-groups',
                        paths: [pages.DPS.ENROLLMENT_GROUPS.LINK, pages.DPS.ENROLLMENT_GROUPS.DETAIL],
                    },
                    {
                        id: '103',
                        title: <MenuTranslate id='provisioningRecords' />,
                        link: '/provisioning-records',
                        paths: [
                            pages.DPS.PROVISIONING_RECORDS.LINK,
                            '/device-provisioning/provisioning-records/:recordId',
                            pages.DPS.PROVISIONING_RECORDS.DETAIL,
                        ],
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
                link: pages.CONFIGURATION.LINK,
                paths: [pages.CONFIGURATION.LINK, pages.CONFIGURATION.THEME_GENERATOR],
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

export const noLayoutPages = [
    '/device-provisioning/linked-hubs/link-new-hub',
    '/device-provisioning/linked-hubs/link-new-hub/:step',
    '/device-provisioning/enrollment-groups/new-enrollment-group',
    '/device-provisioning/enrollment-groups/new-enrollment-group/:step',
]

export const mather = (pathname: string, pattern: string) => matchPath(pattern, pathname)

const Loader = () => {
    const { formatMessage: _ } = useIntl()
    const theme = useTheme()

    return <FullPageLoader i18n={{ loading: _(g.loading) }} />
}

export const NoLayoutRoutes = () => (
    <RoutesGroup>
        <Route element={withSuspense(<LinkNewHubPage />)} path='/device-provisioning/linked-hubs/link-new-hub' />
        <Route element={withSuspense(<LinkNewHubPage />)} path='/device-provisioning/linked-hubs/link-new-hub/:step' />
        <Route element={withSuspense(<NewEnrollmentGroupsPage />)} path='/device-provisioning/enrollment-groups/new-enrollment-group' />
        <Route element={withSuspense(<NewEnrollmentGroupsPage />)} path='/device-provisioning/enrollment-groups/new-enrollment-group/:step' />
    </RoutesGroup>
)

const withSuspense = (Component: any) => <Suspense fallback={<Loader />}>{Component}</Suspense>

export const Routes = () => {
    const { formatMessage: _ } = useIntl()
    return (
        <RoutesGroup>
            {/* ***** DEVICES ***** */}
            <Route path='/devices'>
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={1} />)} path=':id/resources/*' />
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={1} />)} path=':id/resources' />
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={2} />)} path=':id/certificates' />
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={3} />)} path=':id/dps' />
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={3} />)} path=':id/dps/:section' />
                <Route element={withSuspense(<DevicesDetailsPage defaultActiveTab={0} />)} path=':id' />
                <Route element={withSuspense(<DevicesListPage />)} path='' />
            </Route>

            {/* ***** REMOTE CLIENTS ***** */}
            <Route path='/remote-clients'>
                <Route element={withSuspense(<RemoteClientDevicesDetailPage defaultActiveTab={1} />)} path=':id/devices/:deviceId/resources/:resource' />
                <Route element={withSuspense(<RemoteClientDevicesDetailPage defaultActiveTab={1} />)} path=':id/devices/:deviceId/resources' />
                <Route element={withSuspense(<RemoteClientDevicesDetailPage defaultActiveTab={0} />)} path=':id/devices/:deviceId' />
                <Route element={withSuspense(<RemoteClientDetailPage defaultActiveTab={1} />)} path=':id/configuration' />
                <Route element={withSuspense(<RemoteClientDetailPage defaultActiveTab={0} />)} path=':id' />
                <Route element={withSuspense(<RemoteClientsListPage />)} path='' />
            </Route>

            {/* ***** CERTIFICATES ***** */}
            <Route path='/certificates'>
                <Route element={withSuspense(<CertificatesDetailPage />)} path=':certificateId' />
                <Route element={withSuspense(<CertificatesListPage />)} path='' />
            </Route>

            {/* ***** LINKED HUBS ***** */}
            <Route path='/device-provisioning/linked-hubs'>
                <Route element={withSuspense(<LinkedHubsDetailPage />)} path=':hubId/:tab' />
                <Route element={withSuspense(<LinkedHubsDetailPage />)} path=':hubId' />
                <Route element={withSuspense(<LinkedHubsListPage />)} path='' />
            </Route>

            {/* ***** ENROLLMENT GROUPS ***** */}
            <Route path='/device-provisioning/enrollment-groups'>
                <Route element={withSuspense(<EnrollmentGroupsDetailPage />)} path=':enrollmentId' />
                <Route element={withSuspense(<EnrollmentGroupsListPage />)} path='' />
            </Route>

            {/* ***** PROVISIONING RECORDS ***** */}
            <Route path='/device-provisioning/provisioning-records'>
                <Route element={withSuspense(<ProvisioningRecordsDetailPage />)} path=':recordId/:tab' />
                <Route element={withSuspense(<ProvisioningRecordsDetailPage />)} path=':recordId' />
                <Route element={withSuspense(<ProvisioningRecordsListPage />)} path='' />
            </Route>

            {/* ***** CONFIGURATION ***** */}
            <Route path='/configuration'>
                <Route element={withSuspense(<ConfigurationPage defaultActiveTab={1} />)} path='theme-generator' />
                <Route element={withSuspense(<ConfigurationPage />)} path='' />
            </Route>

            {/* ***** OTHERS ***** */}
            <Route element={withSuspense(<PendingCommandsListPage />)} path='/pending-commands' />
            <Route element={withSuspense(<MockApp />)} path='/devices-code-redirect/*' />
            <Route element={<NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} />} path='*' />
        </RoutesGroup>
    )
}
