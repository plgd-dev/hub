import { FC } from 'react'
import { Helmet } from 'react-helmet'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import DevicesDetailsPage from '@shared-ui/app/clientApp/Devices/Detail/DevicesDetailsPage'
import { clientAppSetings } from '@shared-ui/common/services'

import { Props } from './RemoteClientDevicesDetailPage.types'
import { useClientAppPage } from '@/containers/RemoteClients/use-client-app-page'
import { getClientIp } from '@/containers/RemoteClients/utils'
import * as styles from './RemoteClientDevicesDetailPage.styles'
import { messages as t } from '../../RemoteClients.i18n'

const RemoteClientDevicesDetailPage: FC<Props> = (props) => {
    const { defaultActiveTab } = props
    const { formatMessage: _ } = useIntl()
    const [clientData, error, errorElement] = useClientAppPage()
    const { deviceId: routerDeviceId } = useParams()
    const deviceId = routerDeviceId || ''

    if (error) {
        return errorElement
    }

    clientAppSetings.setGeneralConfig({
        httpGatewayAddress: getClientIp(clientData?.clientIP),
    })

    return (
        <div css={styles.detailPage}>
            <Helmet title={`${clientData.clientName}`} />
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
        </div>
    )
}

RemoteClientDevicesDetailPage.displayName = 'RemoteClientDevicesDetailPage'

export default RemoteClientDevicesDetailPage
