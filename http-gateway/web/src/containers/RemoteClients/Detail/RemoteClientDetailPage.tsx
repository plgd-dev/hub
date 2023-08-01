import { FC, useCallback, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import get from 'lodash/get'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'

import { CombinedStoreType } from '@/store/store'
import { messages as t } from '../RemoteClients.i18n'
import { messages as g } from '../../Global.i18n'
import * as styles from './RemoteClientDetailPage.styles'
import { remoteClientStatuses } from '@/containers/RemoteClients/contacts'

const RemoteClientDetailPage: FC<any> = () => {
    const { formatMessage: _ } = useIntl()
    const { id: routerId } = useParams()
    const id = routerId || ''

    const wellKnownConfig = security.getWellKnowConfig() as WellKnownConfigType

    const clientData = useSelector((state: CombinedStoreType) => state.remoteClients.remoteClients.filter((remoteClient) => remoteClient.id === id)?.[0])
    const isTestPage = get(process.env, 'REACT_APP_TEST_REMOTE_CLIENT_DETAIL', false)
    const notFoundPage = useMemo(() => !clientData || (clientData.status === remoteClientStatuses.UNREACHABLE && !isTestPage), [clientData, isTestPage])

    const forwardDataToClientApp = useCallback(() => {
        // @ts-ignore
        const iframeWindow = document?.getElementById('iframe_id')?.contentWindow
        iframeWindow && iframeWindow.postMessage({ PLGD_HUB_REMOTE_PROVISIONING_DATA: wellKnownConfig, key: 'PLGD_EVENT_MESSAGE' }, '*')
    }, [wellKnownConfig])

    if (notFoundPage) {
        return <NotFoundPage message={_(t.notFoundRemoteClientMessage)} title={_(g.pageNotFound)} />
    }

    window.onmessage = (event) => {
        if (event.data.hasOwnProperty('key') && event.data.key === 'PLGD_EVENT_MESSAGE') {
            if (event.data.hasOwnProperty('clientReady') && event.data.clientReady) {
                forwardDataToClientApp()
            }
        }
    }

    return (
        <div css={styles.detailPage}>
            <Helmet title={`${clientData.clientName}`} />
            <iframe css={styles.remoteClientFrame} id='iframe_id' src={clientData.clientIP} title={clientData.clientName}></iframe>
        </div>
    )
}

RemoteClientDetailPage.displayName = 'RemoteClientDetailPage'

export default RemoteClientDetailPage
