import { FC } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import DevicesDetailsPage from '@shared-ui/app/clientApp/Devices/Detail/DevicesDetailsPage'

import { Props } from './RemoteClientDevicesDetailPage.types'
import { messages as t } from '../../RemoteClients.i18n'
import RemoteClientsPage from '@/containers/RemoteClients/RemoteClientsPage/RemoteClientsPage'

const RemoteClientDevicesDetailPage: FC<Props> = (props) => {
    const { defaultActiveTab } = props
    const { formatMessage: _ } = useIntl()
    const { deviceId: routerDeviceId } = useParams()
    const deviceId = routerDeviceId || ''

    return (
        <RemoteClientsPage>
            {(clientData) => (
                <DevicesDetailsPage
                    breadcrumbs={[
                        {
                            link: '/remote-clients',
                            label: _(t.remoteClients),
                        },
                        {
                            link: `/remote-clients/${clientData.id}`,
                            label: clientData.clientName,
                        },
                    ]}
                    defaultActiveTab={defaultActiveTab}
                    defaultDeviceId={deviceId}
                    detailLinkPrefix={`/remote-clients/${clientData.id}`}
                />
            )}
        </RemoteClientsPage>
    )
}

RemoteClientDevicesDetailPage.displayName = 'RemoteClientDevicesDetailPage'

export default RemoteClientDevicesDetailPage
