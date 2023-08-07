import { ReactElement, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'
import get from 'lodash/get'
import { useSelector } from 'react-redux'

import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'

import { messages as t } from './RemoteClients.i18n'
import { messages as g } from '../Global.i18n'
import { remoteClientStatuses } from './contacts'
import { CombinedStoreType } from '@/store/store'

export const useClientAppPage = (): [clientData: any, error: boolean, errorElement: ReactElement] => {
    const { formatMessage: _ } = useIntl()
    const { id: routerId } = useParams()
    const id = routerId || ''

    const clientData = useSelector((state: CombinedStoreType) => state.remoteClients.remoteClients.filter((remoteClient) => remoteClient.id === id)?.[0])
    const isTestPage = get(process.env, 'REACT_APP_TEST_REMOTE_CLIENT_DETAIL', false)
    const notFoundPage = useMemo(() => !clientData || (clientData.status === remoteClientStatuses.UNREACHABLE && !isTestPage), [clientData, isTestPage])

    return [clientData, notFoundPage, <NotFoundPage message={_(t.notFoundRemoteClientMessage)} title={_(g.pageNotFound)} />]
}
