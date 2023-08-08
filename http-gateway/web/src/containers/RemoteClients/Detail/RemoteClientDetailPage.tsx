import { FC } from 'react'
import { Helmet } from 'react-helmet'
import { useIntl } from 'react-intl'

import DevicesListPage from '@shared-ui/app/clientApp/Devices/List/DevicesListPage'
import { clientAppSetings } from '@shared-ui/common/services'

import * as styles from './RemoteClientDetailPage.styles'
import { getClientUrl } from '../utils'
import { useClientAppPage } from '@/containers/RemoteClients/use-client-app-page'
import { messages as t } from '@/containers/RemoteClients/RemoteClients.i18n'
import { messages as menuT } from '@shared-ui/components/Atomic/Menu/Menu.i18n'

const RemoteClientDetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const [clientData, error, errorElement] = useClientAppPage()

    if (error) {
        return errorElement
    }

    clientAppSetings.setGeneralConfig({
        httpGatewayAddress: getClientUrl(clientData?.clientUrl),
    })

    return (
        <div css={styles.detailPage}>
            <Helmet title={`${clientData.clientName}`} />
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
        </div>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
