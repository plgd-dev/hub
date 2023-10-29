import { FC } from 'react'
import { useIntl } from 'react-intl'

import DevicesListPage from '@shared-ui/app/clientApp/Devices/List/DevicesListPage'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'

import { messages as t } from '@/containers/RemoteClients/RemoteClients.i18n'
import RemoteClientsPage from '@/containers/RemoteClients/RemoteClientsPage/RemoteClientsPage'
import { messages as g } from '@/containers/Global.i18n'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'

type Props = {
    defaultActiveTab?: number
}

const RemoteClientDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    return (
        <RemoteClientsPage>
            {(clientData, loading) => {
                if (clientData?.status === remoteClientStatuses.REACHABLE && loading) {
                    return <FullPageLoader i18n={{ loading: _(g.loading) }} />
                }

                return (
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
                        clientData={clientData}
                        defaultActiveTab={props.defaultActiveTab}
                        detailLinkPrefix={`/remote-clients/${clientData?.id}`}
                        title={clientData.clientName}
                    />
                )
            }}
        </RemoteClientsPage>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
