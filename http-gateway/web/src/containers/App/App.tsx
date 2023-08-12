import { useContext, useState, useEffect } from 'react'
import { useIntl } from 'react-intl'
import { AuthProvider, UserManager } from 'oidc-react'
import { BrowserRouter } from 'react-router-dom'

import PageLoader from '@shared-ui/components/Atomic/PageLoader'
import { security } from '@shared-ui/common/services/security'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'
import ConditionalWrapper from '@shared-ui/components/Atomic/ConditionalWrapper'

import './App.scss'
import { messages as t } from './App.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { AppContext } from './AppContext'
import { getAppWellKnownConfiguration } from '@/containers/App/AppRest'
import AppInner from '@/containers/App/AppInner/AppInner'
import AppLayout from '@/containers/App/AppLayout/AppLayout'
import { useLocalStorage } from '@shared-ui/common/hooks'

const App = (props: { mockApp: boolean }) => {
    const { formatMessage: _ } = useIntl()
    const [wellKnownConfig, setWellKnownConfig] = useState<any>(null)
    const [wellKnownConfigFetched, setWellKnownConfigFetched] = useState(false)
    const [configError, setConfigError] = useState<any>(null)

    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', true)

    openTelemetry.init('hub')

    useEffect(() => {
        if (!wellKnownConfig && !wellKnownConfigFetched) {
            const fetchWellKnownConfig = async () => {
                try {
                    const { data: wellKnown } = await openTelemetry.withTelemetry(
                        () => getAppWellKnownConfiguration(process.env.REACT_APP_HTTP_WELL_NOW_CONFIGURATION_ADDRESS || window.location.origin),
                        'get-hub-configuration'
                    )

                    const { webOauthClient, deviceOauthClient, ...generalConfig } = wellKnown

                    const clientId = webOauthClient?.clientId
                    const httpGatewayAddress = wellKnown.httpGatewayAddress
                    const authority = wellKnown.authority

                    if (!clientId || !authority || !httpGatewayAddress) {
                        throw new Error('clientId, authority, audience and httpGatewayAddress must be set in webOauthClient of web_configuration.json')
                    } else {
                        generalConfig.cancelRequestDeadlineTimeout = 10000
                        // Set the auth configurations
                        security.setGeneralConfig(generalConfig)
                        security.setWebOAuthConfig(webOauthClient)
                        security.setDeviceOAuthConfig(deviceOauthClient)
                        security.setWellKnowConfig(wellKnown)

                        setWellKnownConfigFetched(true)
                        setWellKnownConfig(wellKnown)
                    }
                } catch (e) {
                    setConfigError(new Error('Could not retrieve the well-known configuration.'))
                }
            }

            fetchWellKnownConfig().then()
        }
    }, [wellKnownConfig, wellKnownConfigFetched])

    // Render an error box with an auth error
    if (configError) {
        return <div className='client-error-message'>{`${_(t.authError)}: ${configError?.message}`}</div>
    }

    // Placeholder loader while waiting for the auth status
    if (!wellKnownConfig) {
        return (
            <>
                <PageLoader loading className='auth-loader' />
                <div className='page-loading-text'>{`${_(g.loading)}...`}</div>
            </>
        )
    }

    const oidcCommonSettings = {
        authority: wellKnownConfig.authority,
        scope: wellKnownConfig.webOauthClient.scopes.join?.(' '),
    }

    const onSignIn = async () => {
        window.location.href = window.location.href.split('?')[0]
    }

    const Wrapper = (child: any) => (
        <AuthProvider
            {...oidcCommonSettings}
            automaticSilentRenew={true}
            clientId={wellKnownConfig.webOauthClient.clientId}
            onSignIn={onSignIn}
            redirectUri={window.location.href}
            userManager={
                new UserManager({
                    ...oidcCommonSettings,
                    client_id: wellKnownConfig.webOauthClient.clientId,
                    redirect_uri: window.location.href,
                    extraQueryParams: {
                        audience: wellKnownConfig.webOauthClient.audience || undefined,
                    },
                })
            }
        >
            {child}
        </AuthProvider>
    )

    if (props.mockApp) {
        return (
            <BrowserRouter>
                <AppLayout buildInformation={wellKnownConfig?.buildInfo} collapsed={collapsed} setCollapsed={setCollapsed} />
            </BrowserRouter>
        )
    }

    return (
        <ConditionalWrapper condition={!props.mockApp} wrapper={Wrapper}>
            <AppInner collapsed={collapsed} openTelemetry={openTelemetry} setCollapsed={setCollapsed} wellKnownConfig={wellKnownConfig} />
        </ConditionalWrapper>
    )
}

export const useAppConfig = () => useContext(AppContext)

export default App
