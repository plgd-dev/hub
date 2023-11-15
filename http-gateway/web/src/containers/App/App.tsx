import { useContext, useState, useEffect, useCallback, useMemo } from 'react'
import { useIntl } from 'react-intl'
import { AuthProvider, UserManager } from 'oidc-react'
import { BrowserRouter } from 'react-router-dom'
import { useDispatch, useSelector } from 'react-redux'
import get from 'lodash/get'
import { ThemeProvider } from '@emotion/react'

import PageLoader from '@shared-ui/components/Atomic/PageLoader'
import { security } from '@shared-ui/common/services/security'
import { openTelemetry } from '@shared-ui/common/services/opentelemetry'
import ConditionalWrapper from '@shared-ui/components/Atomic/ConditionalWrapper'
import { useLocalStorage } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'

import './App.scss'
import { messages as t } from './App.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { getAppWellKnownConfiguration, getTheme } from '@/containers/App/AppRest'
import AppInner from '@/containers/App/AppInner/AppInner'
import AppLayout from '@/containers/App/AppLayout/AppLayout'
import { setTheme as setDefaultTheme, setThemes } from './slice'
import { CombinedStoreType } from '@/store/store'

const App = (props: { mockApp: boolean }) => {
    const { formatMessage: _ } = useIntl()
    const [wellKnownConfig, setWellKnownConfig] = useState<any>(null)
    const [wellKnownConfigFetched, setWellKnownConfigFetched] = useState(false)
    const [configError, setConfigError] = useState<any>(null)
    const [theme, setTheme] = useState<null | object[]>(null)
    const appStore = useSelector((state: CombinedStoreType) => state.app)

    const [collapsed, setCollapsed] = useLocalStorage('leftPanelCollapsed', false)
    const dispatch = useDispatch()

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

    useEffect(() => {
        if (!theme) {
            const getThemeData = async () => {
                try {
                    const { data: themeData } = await getTheme(window.location.origin)

                    if (themeData) {
                        let themeNames: string[] = []
                        let themes: any = {}

                        themeData.themes.forEach((t: any) => {
                            themeNames = themeNames.concat(Object.keys(t))
                            themes[Object.keys(t)[0]] = t
                        })

                        if (appStore.configuration?.theme === '') {
                            dispatch(setDefaultTheme(themeData.defaultTheme))
                        }

                        dispatch(setThemes(themeNames))
                        setTheme(themeData.themes)
                    }
                } catch (e) {
                    console.log(e)
                    setConfigError(new Error('Could not retrieve the theme file.'))
                }
            }

            getThemeData().then()
        }
    }, [appStore.configuration?.theme, dispatch, theme])

    const currentTheme = useMemo(() => appStore.configuration?.theme ?? 'plgd', [appStore.configuration?.theme])

    const getThemeData = useCallback(() => {
        if (theme) {
            const index = theme.findIndex((i) => Object.keys(i)[0] === currentTheme)
            if (index >= 0) {
                return get(theme[index], `${currentTheme}`, {})
            }
        }

        return {}
    }, [theme, currentTheme])

    // Render an error box with an auth error
    if (configError) {
        return <div className='client-error-message'>{`${_(t.authError)}: ${configError?.message}`}</div>
    }

    // Placeholder loader while waiting for the auth status
    if (!wellKnownConfig || !theme) {
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
            <ThemeProvider theme={getThemeData()}>
                <BrowserRouter>
                    <AppLayout buildInformation={wellKnownConfig?.buildInfo} collapsed={collapsed} mockApp={true} setCollapsed={setCollapsed} />
                </BrowserRouter>
            </ThemeProvider>
        )
    }

    return (
        <ThemeProvider theme={getThemeData()}>
            <ConditionalWrapper condition={!props.mockApp} wrapper={Wrapper}>
                <AppInner collapsed={collapsed} openTelemetry={openTelemetry} setCollapsed={setCollapsed} wellKnownConfig={wellKnownConfig} />
            </ConditionalWrapper>
        </ThemeProvider>
    )
}

export const useAppConfig = () => useContext(AppContext)

export default App
