import { useContext, useState, useEffect, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { AuthProvider, UserManager } from 'oidc-react'
import { BrowserRouter } from 'react-router-dom'
import { useSelector } from 'react-redux'
import { ThemeProvider } from '@emotion/react'

import { security } from '@shared-ui/common/services/security'
import { translate } from '@shared-ui/common/services/translate'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'
import ConditionalWrapper from '@shared-ui/components/Atomic/ConditionalWrapper'
import { useLocalStorage } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'
import { useAppTheme } from '@shared-ui/common/hooks/use-app-theme'
import { getTheme } from '@shared-ui/app/clientApp/App/AppRest'
import { defaultTheme } from '@shared-ui/components/Atomic/_theme'
import FullPageLoader from '@shared-ui/components/Atomic/FullPageLoader'

import './App.scss'
import { messages as t } from './App.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { getAppWellKnownConfiguration } from '@/containers/App/AppRest'
import AppInner from '@/containers/App/AppInner/AppInner'
import AppLayout from '@/containers/App/AppLayout/AppLayout'
import { setTheme, setThemes } from './slice'
import { CombinedStoreType } from '@/store/store'
import { defaultMenu } from '@/routes'
import { updateSidebarVisibility } from '@shared-ui/common/services/sidebar'

const App = (props: { mockApp: boolean }) => {
    const { formatMessage: _ } = useIntl()
    const [wellKnownConfig, setWellKnownConfig] = useState<any>(null)
    const [wellKnownConfigFetched, setWellKnownConfigFetched] = useState(false)
    const [configError, setConfigError] = useState<any>(null)
    const appStore = useSelector((state: CombinedStoreType) => state.app)

    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', false)

    process.env.NODE_ENV !== 'development' && openTelemetry.init('hub')

    useEffect(() => {
        if (!wellKnownConfig && !wellKnownConfigFetched) {
            const fetchWellKnownConfig = async () => {
                try {
                    const { data: wellKnown } = await openTelemetry.withTelemetry(
                        () => getAppWellKnownConfiguration(process.env.REACT_APP_HTTP_WELL_KNOW_CONFIGURATION_ADDRESS || window.location.origin),
                        'get-hub-configuration'
                    )

                    const { webOauthClient, deviceOauthClient, ...generalConfig } = wellKnown

                    if (!wellKnown?.ui?.visibility?.mainSidebar) {
                        wellKnown.ui = { visibility: { mainSidebar: defaultMenu } }
                    }

                    wellKnown.ui = updateSidebarVisibility(wellKnown)

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
                        security.setWellKnownConfig(wellKnown)

                        setWellKnownConfigFetched(true)
                        setWellKnownConfig(wellKnown)
                    }
                } catch (e) {
                    console.error(e)
                    setConfigError(new Error('Could not retrieve the well-known configuration.'))
                }
            }

            fetchWellKnownConfig().then()
        }
    }, [wellKnownConfig, wellKnownConfigFetched])

    useEffect(() => {
        // translate helper for non-component files ( functions etc )
        translate.setTranslator(_)
    }, [_])

    const [theme, themeError, getThemeData] = useAppTheme({
        getTheme,
        setTheme,
        setThemes,
    })

    const currentTheme = useMemo(() => appStore.configuration?.theme ?? defaultTheme, [appStore.configuration?.theme])

    // Render an error box with an auth error
    if (configError || themeError) {
        return <div className='client-error-message'>{`${_(t.authError)}: ${configError?.message}`}</div>
    }

    // Placeholder loader while waiting for the auth status
    if (!wellKnownConfig || !theme) {
        return (
            <ThemeProvider theme={getThemeData(currentTheme)}>
                <FullPageLoader i18n={{ loading: _(g.loading) }} />
            </ThemeProvider>
        )
    }

    const oidcCommonSettings = {
        authority: wellKnownConfig.authority,
        scope: wellKnownConfig.webOauthClient.scopes.join?.(' '),
    }

    const onSignIn = async () => {
        const storedPathname = window.localStorage.getItem('storedPathname')
        window.location.href = storedPathname ?? window.location.href.split('?')[0]
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
            <ThemeProvider theme={getThemeData(currentTheme)}>
                <BrowserRouter>
                    <AppLayout mockApp buildInformation={wellKnownConfig?.buildInfo} collapsed={collapsed} setCollapsed={setCollapsed} />
                </BrowserRouter>
            </ThemeProvider>
        )
    }

    return (
        <ThemeProvider theme={getThemeData(currentTheme)}>
            <ConditionalWrapper condition={!props.mockApp} wrapper={Wrapper}>
                <AppInner collapsed={collapsed} openTelemetry={openTelemetry} setCollapsed={setCollapsed} wellKnownConfig={wellKnownConfig} />
            </ConditionalWrapper>
        </ThemeProvider>
    )
}

export const useAppConfig = () => useContext(AppContext)

export default App
