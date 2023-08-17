import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import { useIntl } from 'react-intl'

import { clientAppSetings } from '@shared-ui/common/services'
import {
    getJwksData,
    getOpenIdConfiguration,
    initializedByPreShared,
    initializeFinal,
    initializeJwksData,
    signIdentityCsr,
} from '@shared-ui/app/clientApp/App/AppRest'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'
import AppLoader from '@shared-ui/app/clientApp/App/AppLoader'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { messages as t } from '../RemoteClients.i18n'
import { AppAuthProviderRefType, Props } from './RemoteClientsAuthProvider.types'

const RemoteClientsAuthProvider = forwardRef<AppAuthProviderRefType, Props>((props, ref) => {
    const { wellKnownConfig, clientData, children, setAuthError, setInitialize } = props
    const { authenticationMode, preSharedSubjectId, preSharedKey } = clientData
    const { formatMessage: _ } = useIntl()
    const [userData] = useState(clientAppSetings.getUserData())
    const [signOutRedirect] = useState(clientAppSetings.getSignOutRedirect())

    useImperativeHandle(ref, () => ({
        getSignOutMethod: () =>
            signOutRedirect({
                post_logout_redirect_uri: window.location.origin,
            }),
        getUserData: () => userData,
    }))

    useEffect(() => {
        if (wellKnownConfig && !wellKnownConfig.isInitialized) {
            if (authenticationMode === DEVICE_AUTH_MODE.X509) {
                try {
                    getOpenIdConfiguration(wellKnownConfig.remoteProvisioning?.authority!).then((result) => {
                        getJwksData(result.data.jwks_uri).then((result) => {
                            initializeJwksData(result.data).then((result) => {
                                const state = result.data.identityCertificateChallenge.state

                                signIdentityCsr(
                                    wellKnownConfig.remoteProvisioning?.certificateAuthority as string,
                                    result.data.identityCertificateChallenge.certificateSigningRequest
                                ).then((result) => {
                                    initializeFinal(state, result.data.certificate).then(() => {
                                        setInitialize(true)
                                    })
                                })
                            })
                        })
                    })
                } catch (e) {
                    console.error(e)
                    setAuthError(e as string)
                }
            } else if (authenticationMode === DEVICE_AUTH_MODE.PRE_SHARED_KEY) {
                if (preSharedSubjectId && preSharedKey) {
                    try {
                        initializedByPreShared(preSharedSubjectId, preSharedKey)
                            .then((r) => {
                                if (r.status === 200) {
                                    setInitialize(true)
                                }
                            })
                            .catch((e) => {
                                Notification.error({
                                    title: _(t.error),
                                    message: e.response.data.message,
                                })
                            })
                    } catch (e) {
                        console.error(e)
                        setAuthError(e as string)
                    }
                } else {
                    setAuthError('Bad parameters for PRE_SHARED_KEY mode')
                }
            }
        }
    }, [wellKnownConfig, setAuthError, setInitialize])

    if (!wellKnownConfig || !wellKnownConfig?.isInitialized) {
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
