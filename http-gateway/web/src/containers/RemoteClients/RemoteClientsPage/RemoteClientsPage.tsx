import { FC, useCallback, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import jwtDecode from 'jwt-decode'
import get from 'lodash/get'

import { useWellKnownConfiguration, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSetings, security } from '@shared-ui/common/services'
import PageLoader from '@shared-ui/components/Atomic/PageLoader'
import AppContext from '@shared-ui/app/clientApp/App/AppContext'
import InitializedByAnother from '@shared-ui/app/clientApp/App/InitializedByAnother'
import { getClientUrl } from '@shared-ui/app/clientApp/utils'
import { useClientAppPage } from '@shared-ui/app/clientApp/RemoteClients/use-client-app-page'

import { Props } from './RemoteClientsPage.types'
import * as styles from './RemoteClientsPage.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../RemoteClients.i18n'
import RemoteClientsAuthProvider from '@/containers/RemoteClients/RemoteClientsAuthProvider'
import { AppAuthProviderRefType } from '../RemoteClientsAuthProvider/RemoteClientsAuthProvider.types'

const RemoteClientsPage: FC<Props> = (props) => {
    const { children } = props
    const { formatMessage: _ } = useIntl()

    const hubWellKnownConfig = security.getWellKnowConfig()

    console.log(hubWellKnownConfig)

    const [clientData, error, errorElement] = useClientAppPage({
        i18n: {
            notFoundRemoteClientMessage: _(t.notFoundRemoteClientMessage),
            pageNotFound: _(g.pageNotFound),
        },
    })
    const [httpGatewayAddress] = useState(getClientUrl(clientData?.clientUrl))
    const [wellKnownConfig, setWellKnownConfig, reFetchConfig, wellKnownConfigError] = useWellKnownConfiguration(httpGatewayAddress, hubWellKnownConfig)

    console.log(wellKnownConfig)

    const [authError, setAuthError] = useState<string | undefined>(undefined)
    const [initializedByAnother, setInitializedByAnother] = useState(false)
    const [suspectedUnauthorized, setSuspectedUnauthorized] = useState(false)
    const authProviderRef = useRef<AppAuthProviderRefType | null>(null)

    const setInitialize = useCallback((value = true) => {
        setWellKnownConfig({
            ...wellKnownConfig,
            isInitialized: value,
        } as WellKnownConfigType)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])

    clientAppSetings.setGeneralConfig({
        httpGatewayAddress,
    })

    const unauthorizedCallback = useCallback(() => {
        setSuspectedUnauthorized(true)

        reFetchConfig().then((newWellKnownConfig: WellKnownConfigType) => {
            const userData = clientAppSetings.getUserData()
            if (userData) {
                const parsedData = jwtDecode(userData.access_token)
                const ownerId = get(parsedData, newWellKnownConfig.remoteProvisioning?.jwtOwnerClaim as string, '')

                if (ownerId !== newWellKnownConfig?.owner) {
                    setInitializedByAnother(true)
                }
            }

            setSuspectedUnauthorized(false)
        })
    }, [reFetchConfig])

    const contextValue = useMemo(
        () => ({
            unauthorizedCallback,
        }),
        [unauthorizedCallback]
    )

    if (error) {
        return errorElement
    }

    if (wellKnownConfigError) {
        return <div className='client-error-message'>{wellKnownConfigError?.message}</div>
    }

    if (!wellKnownConfig) {
        return (
            <>
                <PageLoader loading className='auth-loader' />
                <div className='page-loading-text'>{`${_(g.loading)}...`}</div>
            </>
        )
    } else {
        clientAppSetings.setWellKnowConfig(wellKnownConfig)

        if (wellKnownConfig.remoteProvisioning) {
            clientAppSetings.setWebOAuthConfig({
                authority: wellKnownConfig.remoteProvisioning.authority,
                certificateAuthority: wellKnownConfig.remoteProvisioning.certificateAuthority,
                clientId: wellKnownConfig.remoteProvisioning.webOauthClient?.clientId,
                redirect_uri: window.location.origin,
            })
        }
    }

    if (authError) {
        return <div className='client-error-message'>{`${_(t.authError)}: ${authError}`}</div>
    }

    return (
        <AppContext.Provider value={contextValue}>
            <div css={styles.detailPage}>
                <Helmet title={`${clientData.clientName}`} />
                <RemoteClientsAuthProvider
                    clientData={clientData}
                    ref={authProviderRef}
                    setAuthError={setAuthError}
                    setInitialize={setInitialize}
                    wellKnownConfig={wellKnownConfig}
                >
                    <>
                        <InitializedByAnother show={initializedByAnother} />
                        {!initializedByAnother && !suspectedUnauthorized ? children(clientData) : <div />}
                    </>
                </RemoteClientsAuthProvider>
            </div>
        </AppContext.Provider>
    )
}

RemoteClientsPage.displayName = 'RemoteClientsPage'

export default RemoteClientsPage
