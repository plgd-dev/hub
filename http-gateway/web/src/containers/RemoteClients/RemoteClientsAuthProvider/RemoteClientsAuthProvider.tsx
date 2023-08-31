import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import { useIntl } from 'react-intl'

import { clientAppSettings } from '@shared-ui/common/services'
import {
    getJwksData,
    getOpenIdConfiguration,
    initializedByPreShared,
    initializeFinal,
    initializeJwksData,
    reset,
    signIdentityCsr,
} from '@shared-ui/app/clientApp/App/AppRest'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'
import AppLoader from '@shared-ui/app/clientApp/App/AppLoader'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { messages as t } from '../RemoteClients.i18n'
import { AppAuthProviderRefType, Props } from './RemoteClientsAuthProvider.types'
import notificationId from '@/notificationId'

const RemoteClientsAuthProvider = forwardRef<AppAuthProviderRefType, Props>((props, ref) => {
    const { wellKnownConfig, reInitialization, clientData, children, setAuthError, setInitialize, unauthorizedCallback } = props
    const { id, clientUrl, authenticationMode, preSharedSubjectId, preSharedKey } = clientData
    const { formatMessage: _ } = useIntl()
    const [userData] = useState(clientAppSettings.getUserData())
    const [signOutRedirect] = useState(clientAppSettings.getSignOutRedirect())
    const [reInitializationLoading, setReInitializationLoading] = useState(false)
    const [initializationLoading, setInitializationLoading] = useState(false)

    useImperativeHandle(ref, () => ({
        getSignOutMethod: () =>
            signOutRedirect({
                post_logout_redirect_uri: window.location.origin,
            }),
        getUserData: () => userData,
    }))

    useEffect(() => {
        if (reInitialization && !reInitializationLoading) {
            setReInitializationLoading(true)
            console.log('%c reInitializationProp start! ', 'background: #f0000; color: #bada55')
            reset(clientUrl, unauthorizedCallback)
                .then(() => {
                    console.log('%c reset done! ', 'background: #222; color: #bada55')
                    setInitialize(false)
                    setReInitializationLoading(false)
                })
                .catch(() => {})
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [reInitialization, clientUrl, id, setInitialize, wellKnownConfig?.isInitialized, reInitializationLoading, unauthorizedCallback])

    useEffect(() => {
        if (wellKnownConfig && !wellKnownConfig.isInitialized && !initializationLoading) {
            if (authenticationMode === DEVICE_AUTH_MODE.X509) {
                try {
                    setInitializationLoading(true)
                    getOpenIdConfiguration(wellKnownConfig.remoteProvisioning?.authority!).then((result) => {
                        getJwksData(result.data.jwks_uri).then((result) => {
                            initializeJwksData(result.data).then((result) => {
                                const identityCertificateChallenge = result.data.identityCertificateChallenge

                                signIdentityCsr(
                                    wellKnownConfig.remoteProvisioning?.certificateAuthority as string,
                                    identityCertificateChallenge.certificateSigningRequest
                                ).then((result) => {
                                    initializeFinal(identityCertificateChallenge.state, result.data.certificate).then(() => {
                                        console.log('%c init done x509! ', 'background: #bada55; color: #1a1a1a')
                                        setInitialize(true)
                                        setInitializationLoading(false)
                                    })
                                })
                            })
                        })
                    })
                } catch (e) {
                    console.error(e)
                    setInitializationLoading(false)
                    setAuthError(e as string)
                }
            } else if (authenticationMode === DEVICE_AUTH_MODE.PRE_SHARED_KEY) {
                if (preSharedSubjectId && preSharedKey) {
                    try {
                        initializedByPreShared(preSharedSubjectId, preSharedKey)
                            .then((r) => {
                                if (r.status === 200) {
                                    console.log('%c init done PSK! ', 'background: #bada55; color: #1a1a1a')
                                    setInitialize(true)
                                    setInitializationLoading(false)
                                }
                            })
                            .catch((e) => {
                                Notification.error(
                                    {
                                        title: _(t.error),
                                        message: e.response.data.message,
                                    },
                                    { notificationId: notificationId.HUB_REMOTE_CLIENTS_AUTH_PROVIDER_PRE_SHARED_KEY }
                                )
                            })
                    } catch (e) {
                        console.error(e)
                        setAuthError(e as string)
                        setInitializationLoading(false)
                    }
                } else {
                    setAuthError('Wrong parameters for PRE_SHARED_KEY mode')
                }
            }
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [wellKnownConfig, setAuthError, setInitialize])

    if (!wellKnownConfig || !wellKnownConfig?.isInitialized || initializationLoading) {
        return (
            <AppLoader
                i18n={{
                    loading: 'Loading',
                }}
            />
        )
    }

    return children
})

RemoteClientsAuthProvider.displayName = 'RemoteClientsAuthProvider'

export default RemoteClientsAuthProvider
