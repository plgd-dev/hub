import { FC } from 'react'
import { useIntl } from 'react-intl'

import DevicesListPage from '@shared-ui/app/clientApp/Devices/List/DevicesListPage'

import { messages as t } from '@/containers/RemoteClients/RemoteClients.i18n'
import RemoteClientsPage from '@/containers/RemoteClients/RemoteClientsPage/RemoteClientsPage'

type Props = {
    defaultActiveTab?: number
}

const RemoteClientDetailPage: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    return (
        <RemoteClientsPage>
            {(clientData, reInitializationError, loading, initializedByAnother) => {
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
                        defaultActiveTab={reInitializationError || initializedByAnother ? 1 : props.defaultActiveTab}
                        detailLinkPrefix={`/remote-clients/${clientData?.id}`}
                        initializedByAnother={initializedByAnother}
                        loading={loading}
                        reInitializationError={reInitializationError}
                        title={clientData.clientName}
                    />
                )
            }}
        </RemoteClientsPage>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
