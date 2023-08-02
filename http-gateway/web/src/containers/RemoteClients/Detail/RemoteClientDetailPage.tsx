import { FC, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import get from 'lodash/get'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'

import { CombinedStoreType } from '@/store/store'
import { messages as t } from '../RemoteClients.i18n'
import { messages as g } from '../../Global.i18n'
import * as styles from './RemoteClientDetailPage.styles'
import { remoteClientStatuses } from '@/containers/RemoteClients/contacts'

const RemoteClientDetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const { id: routerId } = useParams()
    const id = routerId || ''

    const clientData = useSelector((state: CombinedStoreType) => state.remoteClients.remoteClients.filter((remoteClient) => remoteClient.id === id)?.[0])
    const isTestPage = get(process.env, 'REACT_APP_TEST_REMOTE_CLIENT_DETAIL', false)
    const notFoundPage = useMemo(() => !clientData || (clientData.status === remoteClientStatuses.UNREACHABLE && !isTestPage), [clientData, isTestPage])

    if (notFoundPage) {
        return <NotFoundPage message={_(t.notFoundRemoteClientMessage)} title={_(g.pageNotFound)} />
    }

    const getClientIp = (clientIp: string) =>
        `${clientIp.endsWith('/') ? clientIp.slice(0, -1) : clientIp}?wellKnownConfigUrl=${
            process.env.REACT_APP_HTTP_WELL_NOW_CONFIGURATION_ADDRESS || window.location.origin
        }`

    return (
        <div css={styles.detailPage}>
            <Helmet title={`${clientData.clientName}`} />
            <iframe css={styles.remoteClientFrame} id='iframe_id' src={getClientIp(clientData.clientIP)} title={clientData.clientName}></iframe>
        </div>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
