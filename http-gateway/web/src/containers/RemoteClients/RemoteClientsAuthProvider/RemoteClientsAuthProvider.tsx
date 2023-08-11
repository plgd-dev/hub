import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'

import { clientAppSetings } from '@shared-ui/common/services'
import { getJwksData, getOpenIdConfiguration, initializeFinal, initializeJwksData, signIdentityCsr } from '@shared-ui/app/clientApp/App/AppRest'
import { REMOTE_PROVISIONING_MODE } from '@shared-ui/app/clientApp/constants'
import AppLoader from '@shared-ui/app/clientApp/App/AppLoader'

import { AppAuthProviderRefType, Props } from './RemoteClientsAuthProvider.types'

const RemoteClientsAuthProvider = forwardRef<AppAuthProviderRefType, Props>((props, ref) => {
    const { wellKnownConfig, children, setAuthError, setInitialize } = props
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
        if (wellKnownConfig && !wellKnownConfig.isInitialized && wellKnownConfig.remoteProvisioning?.mode === REMOTE_PROVISIONING_MODE.USER_AGENT) {
            try {
                getOpenIdConfiguration(wellKnownConfig.remoteProvisioning?.authority).then((result) => {
                    getJwksData(result.data.jwks_uri).then((result) => {
                        initializeJwksData(result.data).then((result) => {
                            const state = result.data.identityCertificateChallenge.state

                            console.log('TU')
                            console.log(wellKnownConfig.remoteProvisioning?.certificateAuthority)

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
