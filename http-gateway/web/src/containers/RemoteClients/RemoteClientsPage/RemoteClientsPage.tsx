import { FC, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import jwtDecode from 'jwt-decode'
import get from 'lodash/get'

import { useWellKnownConfiguration, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSettings, security } from '@shared-ui/common/services'
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
    const [httpGatewayAddress] = useState(clientData ? getClientUrl(clientData?.clientUrl) : '')
    const [loading, setLoading] = useState(false)
    const [wellKnownConfig, setWellKnownConfig, reFetchConfig, wellKnownConfigError] = useWellKnownConfiguration(
        httpGatewayAddress,
        hubWellKnownConfig,
        () => loading && setLoading(false)
    )

    const [authError, setAuthError] = useState<string | undefined>(undefined)
    const [initializedByAnother, setInitializedByAnother] = useState(false)
    const [suspectedUnauthorized, setSuspectedUnauthorized] = useState(false)
    const authProviderRef = useRef<AppAuthProviderRefType | null>(null)

    const setInitialize = useCallback(
        (value = true) => {
            setLoading(true)
            setWellKnownConfig(
                {
                    isInitialized: value,
                } as WellKnownConfigType,
                'update'
            )

            reFetchConfig().then(() => setLoading(false))
        },
        [reFetchConfig, setWellKnownConfig]
    )

    clientAppSettings.setGeneralConfig({
        httpGatewayAddress,
    })

    const reInitialization = useMemo(
        () =>
            wellKnownConfig &&
            wellKnownConfig.deviceAuthenticationMode !== DEVICE_AUTH_MODE.UNINITIALIZED &&
            wellKnownConfig.deviceAuthenticationMode !== clientData.authenticationMode,
        [wellKnownConfig, clientData]
    )

    const compareOwners = useCallback((wellKnownConfig?: WellKnownConfigType) => {
        const userData = clientAppSettings.getUserData()
        if (userData && wellKnownConfig) {
            if (!wellKnownConfig.isInitialized) {
                return true
            }

            const parsedData = jwtDecode(userData.access_token)
            const ownerId = get(parsedData, wellKnownConfig?.remoteProvisioning?.jwtOwnerClaim as string, '')

            if (ownerId === wellKnownConfig?.owner) {
                return true
            }
        }

        return false
    }, [])

    useEffect(() => {
        if (!compareOwners(wellKnownConfig) && !initializedByAnother) {
            setInitializedByAnother(true)
        }
    }, [compareOwners, initializedByAnother, wellKnownConfig])

    const unauthorizedCallback = useCallback(() => {
        setSuspectedUnauthorized(true)

        reFetchConfig().then((newWellKnownConfig: WellKnownConfigType) => {
            if (compareOwners(newWellKnownConfig)) {
                setSuspectedUnauthorized(false)
            } else {
                setInitializedByAnother(true)
            }
        })
    }, [compareOwners, reFetchConfig])

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

    if (!wellKnownConfig || !clientData || loading) {
        return <FullPageLoader i18n={{ loading: _(g.loading) }} />
    } else {
        clientAppSettings.setWellKnowConfig(wellKnownConfig)
        clientAppSettings.setUseToken(compareOwners(wellKnownConfig) && clientData.authenticationMode === DEVICE_AUTH_MODE.X509)

        if (wellKnownConfig.remoteProvisioning) {
            clientAppSettings.setWebOAuthConfig({
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
                {initializedByAnother && <InitializedByAnother description={_(t.initializedByAnotherDesc)} show={true} />}
                {!initializedByAnother && suspectedUnauthorized && <FullPageLoader i18n={{ loading: _(g.loading) }} />}
                {!initializedByAnother && !suspectedUnauthorized && (
                    <RemoteClientsAuthProvider
                        clientData={clientData}
                        reInitialization={reInitialization}
                        ref={authProviderRef}
                        setAuthError={setAuthError}
                        setInitialize={setInitialize}
                        unauthorizedCallback={unauthorizedCallback}
                        wellKnownConfig={wellKnownConfig}
                    >
                        {reInitialization ? <FullPageLoader i18n={{ loading: _(g.loading) }} /> : children(clientData, wellKnownConfig)}
                    </RemoteClientsAuthProvider>
                )}
            </div>
        </AppContext.Provider>
    )
}

RemoteClientsPage.displayName = 'RemoteClientsPage'

export default RemoteClientsPage
