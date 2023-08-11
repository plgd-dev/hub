import { FC } from 'react'
import { useIntl } from 'react-intl'

import DevicesListPage from '@shared-ui/app/clientApp/Devices/List/DevicesListPage'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'

import { messages as t } from '@/containers/RemoteClients/RemoteClients.i18n'
import RemoteClientsPage from '@/containers/RemoteClients/RemoteClientsPage/RemoteClientsPage'

const RemoteClientDetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    return (
        <RemoteClientsPage>
            {(clientData) => (
                <DevicesListPage
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
                    detailLinkPrefix={`/remote-clients/${clientData?.id}`}
                    title={`${_(t.remoteClients)} | ${clientData.clientName} | ${_(menuT.devices)}`}
                />
            )}
        </RemoteClientsPage>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
