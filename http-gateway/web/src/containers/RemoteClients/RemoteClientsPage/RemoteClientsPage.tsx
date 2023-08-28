import { FC, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import jwtDecode from 'jwt-decode'
import get from 'lodash/get'

import { useWellKnownConfiguration, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSetings, security } from '@shared-ui/common/services'
import AppContext from '@shared-ui/app/clientApp/App/AppContext'
import InitializedByAnother from '@shared-ui/app/clientApp/App/InitializedByAnother'
import { getClientUrl } from '@shared-ui/app/clientApp/utils'
import { useClientAppPage } from '@shared-ui/app/clientApp/RemoteClients/use-client-app-page'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'

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

    const [clientData, error, errorElement] = useClientAppPage({
        i18n: {
            notFoundRemoteClientMessage: _(t.notFoundRemoteClientMessage),
            pageNotFound: _(g.pageNotFound),
        },
    })
    const [httpGatewayAddress] = useState(getClientUrl(clientData?.clientUrl))
    const [wellKnownConfig, setWellKnownConfig, reFetchConfig, wellKnownConfigError] = useWellKnownConfiguration(httpGatewayAddress, hubWellKnownConfig)

    const [authError, setAuthError] = useState<string | undefined>(undefined)
    const [initializedByAnother, setInitializedByAnother] = useState(false)
    const [suspectedUnauthorized, setSuspectedUnauthorized] = useState(false)
    const authProviderRef = useRef<AppAuthProviderRefType | null>(null)

    const setInitialize = useCallback(
        (value = true) => {
            console.log('setInitialize', value)
            setWellKnownConfig(
                {
                    isInitialized: value,
                } as WellKnownConfigType,
                'update'
            )
        },
        [setWellKnownConfig]
    )

    clientAppSetings.setGeneralConfig({
        httpGatewayAddress,
    })

    const unauthorizedCallback = useCallback(() => {
        if (clientData.authenticationMode === DEVICE_AUTH_MODE.PRE_SHARED_KEY) {
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
        }
    }, [clientData.authenticationMode, reFetchConfig])

    const contextValue = useMemo(
        () => ({
            unauthorizedCallback,
            remoteClientAuthenticationMode: clientData.authenticationMode,
        }),
        [clientData.authenticationMode, unauthorizedCallback]
    )

    const getContent = useCallback(() => {
        if (clientData.reInitialization) {
            return <FullPageLoader i18n={{ loading: _(g.loading) }} />
        } else if (initializedByAnother) {
            return <InitializedByAnother show={initializedByAnother} />
        } else if (!suspectedUnauthorized) {
            return children(clientData, wellKnownConfig)
        }

        return <div />
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [wellKnownConfig])

    if (error) {
        return errorElement
    }

    if (wellKnownConfigError) {
        return <div className='client-error-message'>{wellKnownConfigError?.message}</div>
    }

    if (!wellKnownConfig) {
        return <FullPageLoader i18n={{ loading: _(g.loading) }} />
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
                    {getContent()}
                </RemoteClientsAuthProvider>
            </div>
        </AppContext.Provider>
    )
}

RemoteClientsPage.displayName = 'RemoteClientsPage'

export default RemoteClientsPage
