import { FC, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'

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
import { Props } from './RemoteClientsAuthProvider.types'
import notificationId from '@/notificationId'

const RemoteClientsAuthProvider: FC<Props> = (props) => {
    const { wellKnownConfig, reInitialization, clientData, children, setAuthError, setInitialize, unauthorizedCallback } = props
    const { clientUrl, authenticationMode, preSharedSubjectId, preSharedKey } = clientData
    const { formatMessage: _ } = useIntl()
    const [reInitializationLoading, setReInitializationLoading] = useState(false)
    const [initializationLoading, setInitializationLoading] = useState(false)

    useEffect(() => {
        if (reInitialization && !reInitializationLoading) {
            setReInitializationLoading(true)
            reset(clientUrl, unauthorizedCallback)
                .then(() => {
                    setInitialize(false)
                    setReInitializationLoading(false)
                })
                .catch(() => {})
        }
    }, [reInitialization, clientUrl, setInitialize, reInitializationLoading, unauthorizedCallback])

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
}

RemoteClientsAuthProvider.displayName = 'RemoteClientsAuthProvider'

export default RemoteClientsAuthProvider
