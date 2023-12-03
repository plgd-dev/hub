import { FC, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import isEqual from 'lodash/isEqual'

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
import { RemoteClientType } from '@shared-ui/app/clientApp/RemoteClients/RemoteClients.types'

import { messages as t } from '../RemoteClients.i18n'
import { Props } from './RemoteClientsAuthProvider.types'
import notificationId from '@/notificationId'

const RemoteClientsAuthProvider: FC<Props> = (props) => {
    const { wellKnownConfig, reInitialization, clientData, loading, children, setAuthError, setInitialize, unauthorizedCallback } = props
    const { clientUrl, deviceAuthenticationMode, preSharedSubjectId, preSharedKey } = clientData
    const { formatMessage: _ } = useIntl()
    const [reInitializationLoading, setReInitializationLoading] = useState(false)
    const [initializationLoading, setInitializationLoading] = useState(false)
    const [reInitializationError, setReInitializationError] = useState(false)

    const getData = (data: RemoteClientType) => ({
        deviceAuthenticationMode: data.deviceAuthenticationMode,
        preSharedKey: data.preSharedKey,
        preSharedSubjectId: data.preSharedSubjectId,
    })

    const [prevClientData, setPrevClientData] = useState(getData(clientData))

    useEffect(() => {
        if (!isEqual(prevClientData, getData(clientData))) {
            setPrevClientData(getData(clientData))
            reInitializationError && setReInitializationError(false)
        }
    }, [clientData, prevClientData, reInitializationError])

    useEffect(() => {
        if (reInitialization && !reInitializationLoading && !reInitializationError && !loading) {
            setReInitializationLoading(true)
            reset(clientUrl, unauthorizedCallback)
                .then(() => {
                    setInitialize(false)
                    setReInitializationLoading(false)
                })
                .catch(() => {
                    setReInitializationLoading(false)
                    setReInitializationError(true)
                })
        }
    }, [reInitialization, clientUrl, setInitialize, reInitializationLoading, unauthorizedCallback, reInitializationError, loading])

    useEffect(() => {
        if (wellKnownConfig && !wellKnownConfig.isInitialized && !initializationLoading && !reInitializationError && !loading) {
            if (deviceAuthenticationMode === DEVICE_AUTH_MODE.X509) {
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
            } else if (deviceAuthenticationMode === DEVICE_AUTH_MODE.PRE_SHARED_KEY) {
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
    }, [wellKnownConfig, setAuthError, setInitialize, loading])

    if (!wellKnownConfig) {
        return (
            <AppLoader
                i18n={{
                    loading: 'Loading',
                }}
            />
        )
    }

    return children(reInitializationLoading, initializationLoading, reInitializationError)
}

RemoteClientsAuthProvider.displayName = 'RemoteClientsAuthProvider'

export default RemoteClientsAuthProvider
