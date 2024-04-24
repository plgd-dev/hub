import { FC, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { useDocumentTitle } from 'usehooks-ts'
import { useDispatch } from 'react-redux'

import { useWellKnownConfiguration, WellKnownConfigType } from '@shared-ui/common/hooks'
import { clientAppSettings, security } from '@shared-ui/common/services'
import AppContext from '@shared-ui/app/share/AppContext'
import { getClientUrl } from '@shared-ui/app/clientApp/utils'
import { useClientAppPage } from '@shared-ui/app/clientApp/RemoteClients/use-client-app-page'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'
import { hasDifferentOwner } from '@shared-ui/common/services/api-utils'
import { remoteClientStatuses } from '@shared-ui/app/clientApp/RemoteClients/constants'
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'

import { Props } from './RemoteClientsPage.types'
import * as styles from './RemoteClientsPage.styles'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../RemoteClients.i18n'
import RemoteClientsAuthProvider from '@/containers/RemoteClients/RemoteClientsAuthProvider'
import { updateRemoteClient } from '@/containers/RemoteClients/slice'
import appConfig from '@/config'

const RemoteClientsPage: FC<Props> = (props) => {
    const { children } = props
    const { formatMessage: _ } = useIntl()

    const hubWellKnownConfig = security.getWellKnowConfig()
    const dispatch = useDispatch()
    const parentalContext = useContext(AppContext)

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
            if (clientData.status === remoteClientStatuses.UNREACHABLE || clientData.status === remoteClientStatuses.DIFFERENT_OWNER) {
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
            wellKnownConfig.deviceAuthenticationMode !== clientData.deviceAuthenticationMode,
        [wellKnownConfig, clientData]
    )

    const differentOwner = useCallback(
        (wellKnownConfig: WellKnownConfigType, clientData: RemoteClientType) => hasDifferentOwner(wellKnownConfig, clientData),
        []
    )

    useEffect(() => {
        clientAppSettings.setUseToken(!differentOwner(wellKnownConfig!, clientData) && clientData.deviceAuthenticationMode === DEVICE_AUTH_MODE.X509)
    }, [wellKnownConfig, clientData.deviceAuthenticationMode, differentOwner, clientData])

    useEffect(() => {
        if (wellKnownConfig) {
            const diffOwner = differentOwner(wellKnownConfig, clientData)

            if (diffOwner && !initializedByAnother) {
                setInitializedByAnother(true)
            } else if (!diffOwner && initializedByAnother) {
                setInitializedByAnother(false)
            }
        }
    }, [differentOwner, initializedByAnother, wellKnownConfig, clientData])

    const unauthorizedCallback = useCallback(() => {
        setSuspectedUnauthorized(true)

        reFetchConfig()
            .then((newWellKnownConfig: WellKnownConfigType) => {
                if (differentOwner(newWellKnownConfig, clientData)) {
                    setInitializedByAnother(true)
                }
            })
            .then(() => {
                setSuspectedUnauthorized(false)
            })
    }, [differentOwner, reFetchConfig, clientData])

    const contextValue = useMemo(
        () => ({
            ...parentalContext,
            unauthorizedCallback,
            updateRemoteClient: updateRemoteClient,
        }),
        [parentalContext, unauthorizedCallback]
    )

    useDocumentTitle(`${clientData.clientName} | ${appConfig.appName}`)

    // just config page with context ( isHub and updateRemoteClient)
    if (clientData.status === remoteClientStatuses.UNREACHABLE) {
        return <AppContext.Provider value={contextValue}>{children(clientData, false, false, initializedByAnother)}</AppContext.Provider>
    }

    if (error) {
        return errorElement
    }

    if (wellKnownConfigError) {
        return <div className='client-error-message'>{wellKnownConfigError?.message}</div>
    }

    if (!wellKnownConfig || !clientData) {
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
                <RemoteClientsAuthProvider
                    clientData={clientData}
                    loading={loading}
                    reInitialization={reInitialization}
                    setAuthError={setAuthError}
                    setInitialize={setInitialize}
                    unauthorizedCallback={unauthorizedCallback}
                    wellKnownConfig={wellKnownConfig}
                >
                    {(reInitializationLoading, initializationLoading, reInitializationError) => {
                        if (suspectedUnauthorized) {
                            return <FullPageLoader i18n={{ loading: _(g.loading) }} />
                        } else {
                            return children(
                                clientData,
                                reInitializationError,
                                initializationLoading || reInitializationLoading || loading || !!reInitialization,
                                initializedByAnother && !initializationLoading && !reInitializationLoading && wellKnownConfig?.isInitialized
                            )
                        }
                    }}
                </RemoteClientsAuthProvider>
            </div>
        </AppContext.Provider>
    )
}

RemoteClientsPage.displayName = 'RemoteClientsPage'

export default RemoteClientsPage
