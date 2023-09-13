import { FC } from 'react'
import { useIntl } from 'react-intl'

import DevicesListPage from '@shared-ui/app/clientApp/Devices/List/DevicesListPage'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'

import { messages as t } from '@/containers/RemoteClients/RemoteClients.i18n'
import RemoteClientsPage from '@/containers/RemoteClients/RemoteClientsPage/RemoteClientsPage'
import { messages as g } from '@/containers/Global.i18n'

const RemoteClientDetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()

    return (
        <RemoteClientsPage>
            {(clientData, wellKnownConfig) => {
                if (!wellKnownConfig || !wellKnownConfig.isInitialized) {
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
