import { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { Helmet } from 'react-helmet'
import { useDispatch } from 'react-redux'

import { useWellKnownConfiguration, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSettings, security } from '@shared-ui/common/services'
import AppContext from '@shared-ui/app/clientApp/App/AppContext'
import InitializedByAnother from '@shared-ui/app/clientApp/App/InitializedByAnother'
import { getClientUrl } from '@shared-ui/app/clientApp/utils'
import { useClientAppPage } from '@shared-ui/app/clientApp/RemoteClients/use-client-app-page'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'
import { hasDifferentOwner } from '@shared-ui/common/services/api-utils'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'

import { Props } from './RemoteClientsPage.types'
import * as styles from './RemoteClientsPage.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../RemoteClients.i18n'
import RemoteClientsAuthProvider from '@/containers/RemoteClients/RemoteClientsAuthProvider'
import { updateRemoteClient } from '@/containers/RemoteClients/slice'

const RemoteClientsPage: FC<Props> = (props) => {
    const { children } = props
    const { formatMessage: _ } = useIntl()

    const hubWellKnownConfig = security.getWellKnowConfig()
    const dispatch = useDispatch()

    const [clientData, error, errorElement] = useClientAppPage({
        i18n: {
            notFoundRemoteClientMessage: _(t.notFoundRemoteClientMessage),
            pageNotFound: _(g.pageNotFound),
        },
    })
    const [httpGatewayAddress] = useState(clientData ? getClientUrl(clientData?.clientUrl) : '')
    const [loading, setLoading] = useState(false)

    const [wellKnownConfig, setWellKnownConfig, reFetchConfig, wellKnownConfigError] = useWellKnownConfiguration(httpGatewayAddress, {
        defaultRemoteProvisioningData: hubWellKnownConfig,
        onConfigurationChange: () => {
            loading && setLoading(false)
        },
        onError: () => {
            if (clientData.status === remoteClientStatuses.REACHABLE) {
                dispatch(updateRemoteClient({ status: remoteClientStatuses.UNREACHABLE, id: clientData?.id }))
            }
        },
        onSuccess: () => {
            if (clientData.status === remoteClientStatuses.UNREACHABLE) {
                dispatch(updateRemoteClient({ status: remoteClientStatuses.REACHABLE, id: clientData?.id }))
            }
        },
    })

    const [authError, setAuthError] = useState<string | undefined>(undefined)
    const [initializedByAnother, setInitializedByAnother] = useState(false)
    const [suspectedUnauthorized, setSuspectedUnauthorized] = useState(false)

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

    const differentOwner = useCallback((wellKnownConfig?: WellKnownConfigType) => hasDifferentOwner(wellKnownConfig, clientData), [clientData])

    useEffect(() => {
        clientAppSettings.setUseToken(!differentOwner(wellKnownConfig) && clientData.authenticationMode === DEVICE_AUTH_MODE.X509)
    }, [wellKnownConfig, clientData.authenticationMode, differentOwner])

    useEffect(() => {
        if (wellKnownConfig && differentOwner(wellKnownConfig) && !initializedByAnother) {
            setInitializedByAnother(true)
        }
    }, [differentOwner, initializedByAnother, wellKnownConfig])

    const unauthorizedCallback = useCallback(() => {
        setSuspectedUnauthorized(true)

        reFetchConfig().then((newWellKnownConfig: WellKnownConfigType) => {
            if (differentOwner(newWellKnownConfig)) {
                setInitializedByAnother(true)
            } else {
                setSuspectedUnauthorized(false)
            }
        })
    }, [differentOwner, reFetchConfig])

    const contextValue = useMemo(
        () => ({
            unauthorizedCallback,
        }),
        [unauthorizedCallback]
    )

    // just config page
    if (clientData.status === remoteClientStatuses.UNREACHABLE) {
        return children(clientData, false)
    }

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
        clientAppSettings.setClientData(clientData)

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
                        setAuthError={setAuthError}
                        setInitialize={setInitialize}
                        unauthorizedCallback={unauthorizedCallback}
                        wellKnownConfig={wellKnownConfig}
                    >
                        {reInitialization ? (
                            <FullPageLoader i18n={{ loading: _(g.loading) }} />
                        ) : (
                            children(clientData, !wellKnownConfig || !wellKnownConfig.isInitialized)
                        )}
                    </RemoteClientsAuthProvider>
                )}
            </div>
        </AppContext.Provider>
    )
}

RemoteClientsPage.displayName = 'RemoteClientsPage'

export default RemoteClientsPage
