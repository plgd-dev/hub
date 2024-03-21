import React, { forwardRef, lazy, useCallback, useEffect, useImperativeHandle, useMemo, useState } from 'react'
import { useResizeDetector } from 'react-resize-detector'
import { ThemeProvider } from '@emotion/react'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'
import debounce from 'lodash/debounce'
import { useDispatch, useSelector } from 'react-redux'
import { RGBColor } from 'react-color'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import ColorPicker from '@shared-ui/components/Atomic/ColorPicker'
import VersionMark from '@shared-ui/components/Atomic/VersionMark'
import { severities } from '@shared-ui/components/Atomic/VersionMark/constants'
import LeftPanel from '@shared-ui/components/Layout/LeftPanel'
import { IconDashboard, IconDevices, IconDeviceUpdate, IconLoader, IconNetwork } from '@shared-ui/components/Atomic'
import Logo from '@shared-ui/components/Atomic/Logo'
import Layout from '@shared-ui/components/Layout'
import Header from '@shared-ui/components/Layout/Header'
import UserWidget from '@shared-ui/components/Layout/Header/UserWidget'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import { getThemeTemplate } from '@shared-ui/components/Atomic/_theme/template'
import Tabs from '@shared-ui/components/Atomic/Tabs'
import PageLayout from '@shared-ui/components/Atomic/PageLayout'
import Button from '@shared-ui/components/Atomic/Button'
import Footer from '@shared-ui/components/Layout/Footer'
import { useIsMounted } from '@shared-ui/common/hooks'
import { getTheme } from '@shared-ui/app/clientApp/App/AppRest'

import { messages as g } from '@/containers/Global.i18n'
import { PreviewAppRefType } from './PreviewApp.types'
import * as styles from './PreviewApp.styles'
import { setPreviewTheme, setThemeModal } from '@/containers/App/slice'
import isEqual from 'lodash/isEqual'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const PreviewApp = forwardRef<PreviewAppRefType, any>((props, ref) => {
    const { formatMessage: _ } = useIntl()
    const { ref: resizerRef, height } = useResizeDetector({
        refreshRate: 500,
        handleHeight: true,
        handleWidth: false,
    })

    const dispatch = useDispatch()
    const appStore = useSelector((state: any) => state.app)

    const defaultLogo = useMemo(
        () => ({
            height: 32,
            source: 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTQ3IiBoZWlnaHQ9IjMyIiB2aWV3Qm94PSIwIDAgMTQ3IDMyIiBmaWxsPSJub25lIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHJvbGU9ImltZyIgY2xhc3M9ImNzcy1qbHBlb3UiPjxwYXRoIGZpbGwtcnVsZT0iZXZlbm9kZCIgY2xpcC1ydWxlPSJldmVub2RkIiBkPSJNODUuMTA2IDExLjgzNWMxLjQ5NCAwIDIuNjMuNDg1IDMuNDA4IDEuNDU3di0xLjMwNWgyLjgzNnY4LjQ4M2MwIDEuODY3LS40OTMgMy4yNTgtMS40NzYgNC4xNjgtLjk4NC45MDctMi40MiAxLjM2Mi00LjMwOCAxLjM2Mi0uOTk2IDAtMS45NDItLjEyLTIuODM0LS4zNi0uODk1LS4yNC0xLjYzNi0uNTg4LTIuMjIyLTEuMDQyLjExOS0uMjEgMS4wNy0xLjkwNyAxLjE4OC0yLjEyLjQzMi4zNTUuOTguNjM1IDEuNjQ2Ljg0Mi42NjUuMjEgMS4zMjcuMzEyIDEuOTkyLjMxMiAxLjAzNCAwIDEuNzk3LS4yMjkgMi4yODgtLjY5LjQ5My0uNDYyLjczOC0xLjE2LjczOC0yLjA5NXYtLjQzNGMtLjc3OS44NDctMS44NjYgMS4yNy0zLjI1NiAxLjI3YTUuNTMgNS41MyAwIDAgMS0yLjU5Ni0uNjE1IDQuNzUzIDQuNzUzIDAgMCAxLTEuODY2LTEuNzM2Yy0uNDYtLjc0Mi0uNjktMS42MDItLjY5LTIuNTc0IDAtLjk3My4yMy0xLjgzLjY5LTIuNTc1YTQuNzI1IDQuNzI1IDAgMCAxIDEuODY2LTEuNzMzYy43ODYtLjQxIDEuNjUtLjYxNSAyLjU5Ni0uNjE1Wm0tMTkuMDE2LS4wMDNjLjk1OCAwIDEuODMuMjE4IDIuNjE3LjY1NWE0LjcxIDQuNzEgMCAwIDEgMS44NDUgMS44MzVjLjQ0Ny43OS42NzIgMS43MDguNjcyIDIuNzU4IDAgMS4wNDctLjIyNSAxLjk2NS0uNjcyIDIuNzU1YTQuNzE5IDQuNzE5IDAgMCAxLTEuODQ1IDEuODM4Yy0uNzg3LjQzNS0xLjY1OS42NTItMi42MTcuNjUyLTEuMzE0IDAtMi4zNDktLjQxLTMuMTAyLTEuMjMzdjQuNzU2SDYwVjExLjk4M2gyLjg1NHYxLjE3NWMuNzQtLjg4MyAxLjgyLTEuMzI1IDMuMjM2LTEuMzI1Wm02My44NzEgMGMuOTgzIDAgMS44NzYuMjA1IDIuNjguNjE1YTQuNzgzIDQuNzgzIDAgMCAxIDEuOTI2IDEuNzgzYy40NzguNzc1LjcxOCAxLjY5NS43MTggMi43NTVsLTcuNjIyIDEuNDU4Yy4yMTcuNTA1LjU1OS44ODQgMS4wMjQgMS4xMzcuNDY1LjI1MyAxLjAzNy4zNzggMS43MTQuMzc4LjUzNiAwIDEuMDExLS4wNzggMS40MjgtLjIzNWEzLjQ4IDMuNDggMCAwIDAgMS4xNTgtLjc1bDEuNTkgMS43MDVjLS45NzEgMS4wOTctMi4zODkgMS42NDctNC4yNTIgMS42NDctMS4xNjMgMC0yLjE5MS0uMjIyLTMuMDg0LS42Ny0uODk1LS40NS0xLjU4My0xLjA3My0yLjA2OC0xLjg2Ny0uNDg1LS43OTYtLjcyOC0xLjY5OC0uNzI4LTIuNzA4IDAtLjk5OC4yNC0xLjg5Ny43MTgtMi43YTUuMDA0IDUuMDA0IDAgMCAxIDEuOTcyLTEuODc1Yy44MzctLjQ0OCAxLjc3OS0uNjczIDIuODI2LS42NzNabS03LjQyMi0zLjcxMnYxNC4wNTVoLTIuODU0VjIxYy0uNzQxLjg4NS0xLjgxNiAxLjMyNS0zLjIxOSAxLjMyNS0uOTcxIDAtMS44NDgtLjIxNS0yLjYzNC0uNjQyYTQuNjU3IDQuNjU3IDAgMCAxLTEuODQ4LTEuODM4Yy0uNDQ3LS43OTUtLjY3LTEuNzE4LS42Ny0yLjc2NSAwLTEuMDQ3LjIyNS0xLjk3LjY3LTIuNzY1YTQuNjU4IDQuNjU4IDAgMCAxIDEuODQ4LTEuODM3Yy43ODYtLjQzIDEuNjYzLS42NDYgMi42MzQtLjY0NiAxLjMxNSAwIDIuMzQ0LjQxMyAzLjA4NSAxLjIzM1Y4LjEyaDIuOTg4Wm0tMTQuMjk3IDEwLjUzYy41MjQgMCAuOTY0LjE3IDEuMzIzLjUwMy4zNTYuMzM0LjUzNi43NzUuNTM2IDEuMzE3IDAgLjUzLS4xOC45NzMtLjUzNiAxLjMyNi0uMzU5LjM1NC0uNzk5LjUzLTEuMzIzLjUzLS41MjMgMC0uOTY1LS4xNzYtMS4zMjItLjUzLS4zNTYtLjM1NC0uNTM2LS43OTYtLjUzNi0xLjMyNiAwLS41NDIuMTgtLjk4My41MzYtMS4zMTcuMzU3LS4zMzMuNzk5LS41MDMgMS4zMjItLjUwM1pNNzYuMjAyIDguMTJ2MTAuNTEyYzAgLjQzLjExNC43Ni4zMzcuOTk2LjIyMi4yMzIuNTM4LjM1Ljk0Ny4zNS4xNTQgMCAuMzA5LS4wMi40Ny0uMDU4LjE2LS4wMzcuMjg0LS4wODMuMzcyLS4xMzMuMDE1LjIzLjEyMiAyLjA2My4xMzQgMi4yOTNhNC45NiA0Ljk2IDAgMCAxLTEuNTUuMjQ1Yy0xLjE2MiAwLTIuMDctLjMwNS0yLjcyLS45MTctLjY1Mi0uNjEzLS45NzUtMS40OC0uOTc1LTIuNjA1VjguMTJoMi45ODVaTTEwNC40NzEgOHYxNC4wNTVoLTIuODU0VjIwLjg4Yy0uNzQxLjg4NS0xLjgxMyAxLjMyNS0zLjIxOSAxLjMyNS0uOTY4IDAtMS44NDgtLjIxMi0yLjYzMi0uNjQyYTQuNjUzIDQuNjUzIDAgMCAxLTEuODQ4LTEuODM4Yy0uNDQ3LS43OTUtLjY3LTEuNzE4LS42Ny0yLjc2NSAwLTEuMDQ3LjIyMy0xLjk3LjY3LTIuNzY1YTQuNjQ2IDQuNjQ2IDAgMCAxIDEuODQ4LTEuODM4Yy43ODQtLjQzIDEuNjY0LS42NDIgMi42MzItLjY0MiAxLjMxNyAwIDIuMzQ0LjQxIDMuMDg1IDEuMjNWOGgyLjk4OFptMzQuMDg4IDMuOTQzIDIuNzMgNy42NzcgMi43My03LjY3N0gxNDdsLTQuMDk4IDEwLjIyNWgtMy4yODRsLTQuMDk4LTEwLjIyNWgzLjAzOVptLTIxLjU3NyAyLjMxNWMtLjc2NiAwLTEuMzk4LjI1NS0xLjg5Ni43NjctLjQ5OC41MS0uNzQ2IDEuMTk1LS43NDYgMi4wNTUgMCAuODU4LjI0OCAxLjU0Mi43NDYgMi4wNTUuNS41MSAxLjEzLjc2NSAxLjg5Ni43NjUuNzU2IDAgMS4zOC0uMjU1IDEuODc4LS43NjUuNDk5LS41MTMuNzQ2LTEuMTk3Ljc0Ni0yLjA1NSAwLS44Ni0uMjQ3LTEuNTQ1LS43NDYtMi4wNTUtLjQ5OC0uNTEyLTEuMTI1LS43NjctMS44NzgtLjc2N1ptLTUxLjQwOCAwYy0uNzY2IDAtMS4zOTUuMjU1LTEuODg2Ljc2Ny0uNDkzLjUxLS43MzggMS4xOTUtLjczOCAyLjA1NSAwIC44NTguMjQ1IDEuNTQyLjczOCAyLjA1NS40OS41MSAxLjEyLjc2NSAxLjg4Ni43NjUuNzY3IDAgMS4zOTYtLjI1NSAxLjg4Ni0uNzY1LjQ5LS41MTMuNzM5LTEuMTk3LjczOS0yLjA1NSAwLS44Ni0uMjQ4LTEuNTQ1LS43NC0yLjA1NS0uNDktLjUxMi0xLjExOC0uNzY3LTEuODg1LS43NjdabTMzLjM0My0uMTJjLS43NjYgMC0xLjM5OC4yNTUtMS44OTYuNzY3LS40OTguNTEtLjc0OSAxLjE5NS0uNzQ5IDIuMDU1IDAgLjg1OC4yNSAxLjU0My43NDkgMi4wNTUuNDk4LjUxIDEuMTMuNzY1IDEuODk1Ljc2NS43NTQuMDAzIDEuMzgxLS4yNTUgMS44NzYtLjc2NS40OTktLjUxMi43NDktMS4xOTcuNzQ5LTIuMDU1IDAtLjg2LS4yNS0xLjU0NS0uNzQ5LTIuMDU1LS40OTgtLjUxMi0xLjEyMi0uNzY4LTEuODc2LS43NjhabS0xMy4yMTcuMTJjLS43OTEgMC0xLjQ0My4yMy0xLjk1NC42OTItLjUwOC40Ni0uNzYzIDEuMDYzLS43NjMgMS44MDggMCAuNzQ0LjI1NSAxLjM0Ny43NjMgMS44MS41MS40NiAxLjE2My42OSAxLjk1NC42OS43OTEgMCAxLjQzOC0uMjMgMS45NDQtLjY5LjUwNi0uNDYzLjc1Ni0xLjA2Ni43NTYtMS44MSAwLS43NDUtLjI1LTEuMzQ4LS43NTYtMS44MDgtLjUwNi0uNDYzLTEuMTUzLS42OTItMS45NDQtLjY5MlptNDQuMjYxLS4xOWMtLjc1MyAwLTEuMzY4LjI0LTEuODM4LjcyLS40NzMuNDgtLjcyMyAxLjE0Mi0uNzQ4IDEuOTkuNTAxLS4wOTggNC41MTUtLjg3IDUuMDE4LS45NjhhMi4yMjYgMi4yMjYgMCAwIDAtLjg2Mi0xLjI2OGMtLjQzNS0uMzE3LS45NTgtLjQ3NC0xLjU3LS40NzRaIiBmaWxsPSIjMTkxQTFBIj48L3BhdGg+PHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik00Ny40MDUgMjcuMDA5QTEyLjY4IDEyLjY4IDAgMCAwIDUwIDE5LjMwMmMwLTcuMDA3LTUuNjQ2LTEyLjY4OC0xMi42MS0xMi42ODgtLjU2NiAwLTEuMTIyLjA0Mi0xLjY2OS4xMTUtLjQ1LS43MjMtLjgwNi0xLjI2NS0xLjAyNC0xLjU0NEMzMi4yMzMgMi4wMjcgMjguNDA1IDAgMjQuMTA2IDBjLTYuMzcgMC0xMS43MDQgNC40NTYtMTMuMDk4IDEwLjQzOCAwLS4wMDgtLjAwOC0uMDE4LS4wMDYtLjAyNS0uMS0uMDAzLS4xOTktLjAxLS4zLS4wMUM0Ljc5MyAxMC40MDMgMCAxNS4yMjYgMCAyMS4xNzMgMCAyNy4xMTggNC43OTIgMzEuOTQgMTAuNzAzIDMxLjk0YzMuOTM2IDAgOS4yOC4wNDcgMTIuOTguMDQ3IDIuMjA3IDAgMy40MDItLjAyNiAxMi41MTMgMGwuMjItLjAwMmMxLjE4NC4wNTYgMy43MjkuMDExIDYuMTQ3LTEuMTA5LjM4OC0uMTg0IDIuNzQ1LTEuMTc2IDQuODQzLTMuODY2bC0uMDAxLS4wMDFaIiBmaWxsPSIjRkVCRjQwIj48L3BhdGg+PHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik00Ny40MDUgMjcuMDA5QTEyLjY4IDEyLjY4IDAgMCAwIDUwIDE5LjMwMmMwLTcuMDA3LTUuNjQ2LTEyLjY4OC0xMi42MS0xMi42ODgtLjU2NiAwLTEuMTIyLjA0Mi0xLjY2OC4xMTUtLjQ1LS43MjMtLjgwNi0xLjI2NS0xLjAyNS0xLjU0NEMzMi4yMzMgMi4wMjcgMjguNDA2IDAgMjQuMTA2IDBjLTcuNDMgMC0xMy40NTcgNi4wNjEtMTMuNDU3IDEzLjU0IDAgNy40NzggNS42MDIgMTguNDQ3IDEzLjAzNCAxOC40NDcgMi4yMDcgMCAzLjQwMy0uMDI2IDEyLjUxMyAwbC4yMjEtLjAwMmMxLjE4My4wNTYgMy43MjguMDExIDYuMTQ3LTEuMTA5LjM4OC0uMTg0IDIuNzQ0LTEuMTc2IDQuODQyLTMuODY2bC0uMDAxLS4wMDFaIiBmaWxsPSIjODdDRUYyIj48L3BhdGg+PHBhdGggZmlsbC1ydWxlPSJldmVub2RkIiBjbGlwLXJ1bGU9ImV2ZW5vZGQiIGQ9Ik00Ni45MyAyNi4yOTZDNDQuODgyIDIyLjYyMiAzNi4yNzggNy4yMSAzNC42OTYgNS4xODQgMzIuMjMzIDIuMDI3IDI4LjQwNCAwIDI0LjEwNSAwYy03LjQzIDAtMTMuNDU3IDYuMDYxLTEzLjQ1NyAxMy41NCAwIDcuNDc4IDUuNjAyIDE4LjQ0NyAxMy4wMzQgMTguNDQ3IDIuMjA4IDAgMy40MDMtLjAyNiAxMi41MTMgMGwuMjIxLS4wMDJjMS4xODQuMDU2IDMuNzI5LjAxMiA2LjE0Ny0xLjEwOS4zODgtLjE4NCAyLjc0NS0xLjE3NiA0Ljg0Mi0zLjg2NmwtLjQ3Ni0uNzE0WiIgZmlsbD0iIzIyNjFBRSI+PC9wYXRoPjwvc3ZnPgo=',
            width: 147,
        }),
        []
    )

    const [activeId, setActiveId] = useState('1')
    const [collapsed, setCollapsed] = useState(false)
    const [colorPalette, setColorPalette] = useState<any>(undefined)
    const [defaultColorPalette, setDefaultColorPalette] = useState<any>(undefined)
    const [activeTabItem, setActiveTabItem] = useState(0)
    const [logoData, setLogoData] = useState(appStore.configuration.previewTheme?.logo || defaultLogo)

    const isMounted = useIsMounted()

    useEffect(() => {
        const getThemeData = async (theme: string) => {
            try {
                const { data: themeData } = await getTheme(window.location.origin)

                if (themeData) {
                    let themeNames: string[] = []
                    let themes: any = {}

                    themeData.themes.forEach((t: any) => {
                        themeNames = themeNames.concat(Object.keys(t))
                        themes[Object.keys(t)[0]] = t[Object.keys(t)[0]]
                    })

                    if (themes.hasOwnProperty(theme)) {
                        return themes[theme].colorPalette
                    }
                }
            } catch (e) {
                console.log(e)
            }
        }

        if (colorPalette === undefined) {
            getThemeData(appStore.configuration.theme).then((palette) => {
                setColorPalette(palette)
                setDefaultColorPalette(palette)
            })
        }
    }, [colorPalette, appStore.configuration])

    useEffect(() => {
        if (appStore.configuration.previewTheme) {
            const { logo, colorPalette } = appStore.configuration.previewTheme

            if (colorPalette) {
                setColorPalette(colorPalette)
            }

            if (logo && !isEqual(logoData, logo)) {
                setLogoData(logo)
            }
        }
    }, [appStore.configuration.previewTheme, logoData])

    useImperativeHandle(ref, () => ({
        getThemeData: () => getThemeTemplate(colorPalette, logoData),
    }))

    const handleColorChange = debounce((colorKey: string, color: string) => {
        setColorPalette({ ...colorPalette, [colorKey]: color })
    }, 300)

    const onTabItemChange = useCallback((i: number) => {
        setActiveTabItem(i)
    }, [])

    const handleThemeUpdate = useCallback(() => {
        dispatch(setPreviewTheme(getThemeTemplate(colorPalette, logoData)))
        dispatch(setThemeModal(false))
    }, [colorPalette, logoData, dispatch])

    const tabs = useMemo(
        () => [
            {
                name: 'List page',
                id: 0,
                content: <Tab1 isActiveTab={activeTabItem === 0} />,
            },
            {
                name: 'Detail page',
                id: 1,
                content: <Tab2 />,
            },
            {
                name: 'Components',
                id: 2,
                content: <Tab3 />,
            },
        ],
        [activeTabItem]
    )

    const handleReset = useCallback(() => {
        setColorPalette(defaultColorPalette)
        dispatch(setPreviewTheme(undefined))
        dispatch(setThemeModal(false))
    }, [defaultColorPalette, dispatch])

    const handleLogoResize = useCallback(
        (height: number, width: number) => {
            setLogoData({
                ...logoData,
                height,
                width,
            })
        },
        [logoData]
    )

    return (
        <div
            ref={resizerRef}
            style={{
                height: '100%',
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
            }}
        >
            {colorPalette && (
                <Row>
                    <Column size={9}>
                        <div style={{ height, position: 'relative' }}>
                            <ThemeProvider theme={getThemeTemplate(colorPalette, logoData)}>
                                <Layout
                                    content={
                                        <PageLayout
                                            footer={
                                                <Footer
                                                    css={styles.relative}
                                                    footerExpanded={false}
                                                    paginationComponent={<div id='paginationPortalTargetPreviewApp'></div>}
                                                />
                                            }
                                            header={
                                                <div css={styles.itemWrapper}>
                                                    <Button css={styles.item}>Secondary CTA</Button>
                                                    <Button css={styles.item} variant='primary'>
                                                        Primary CTA
                                                    </Button>
                                                </div>
                                            }
                                            title='Example page'
                                            xPadding={false}
                                        >
                                            {isMounted &&
                                                document.querySelector('#breadcrumbsPortalTargetPreviewApp') &&
                                                ReactDOM.createPortal(
                                                    <Breadcrumbs
                                                        items={[
                                                            {
                                                                label: 'Parent page',
                                                                link: '/',
                                                            },
                                                            { label: 'Current page' },
                                                        ]}
                                                    />,
                                                    document.querySelector('#breadcrumbsPortalTargetPreviewApp') as Element
                                                )}

                                            <Tabs fullHeight innerPadding isAsync activeItem={activeTabItem} onItemChange={onTabItemChange} tabs={tabs} />
                                        </PageLayout>
                                    }
                                    header={
                                        <Header
                                            breadcrumbs={<div id='breadcrumbsPortalTargetPreviewApp'></div>}
                                            userWidget={
                                                <UserWidget
                                                    description='Description'
                                                    image='https://place-hold.it/300x300?text=UN&fontsize=56'
                                                    logoutTitle={_(g.logOut)}
                                                    name='User name'
                                                    onLogout={() => console.log('logout')}
                                                />
                                            }
                                        />
                                    }
                                    leftPanel={
                                        <LeftPanel
                                            activeId={activeId}
                                            collapsed={collapsed}
                                            logo={<Logo collapsed={collapsed} logo={logoData} onResized={handleLogoResize} />}
                                            menu={[
                                                {
                                                    title: 'Main menu',
                                                    items: [
                                                        {
                                                            icon: <IconDashboard />,
                                                            id: '1',
                                                            title: 'Item 1',
                                                            visibility: true,
                                                        },
                                                        {
                                                            icon: <IconDevices />,
                                                            id: '2',
                                                            title: 'Item 2',
                                                            visibility: true,
                                                        },
                                                    ],
                                                },
                                                {
                                                    title: 'Other',
                                                    items: [
                                                        {
                                                            icon: <IconNetwork />,
                                                            id: '10',
                                                            title: 'Sub menu',
                                                            visibility: true,
                                                            children: [
                                                                {
                                                                    id: '101',
                                                                    title: 'Sub item 1',
                                                                    tag: { variant: 'success', text: 'New' },
                                                                },
                                                                { id: '102', title: 'Sub item 2' },
                                                                { id: '103', title: 'Sub item 3' },
                                                                {
                                                                    id: '104',
                                                                    title: 'Sub item 4',
                                                                    tag: { variant: 'info', text: 'Soon!' },
                                                                },
                                                            ],
                                                        },
                                                        {
                                                            icon: <IconDeviceUpdate />,
                                                            id: '3',
                                                            title: 'Item 4',
                                                            visibility: true,
                                                        },
                                                    ],
                                                },
                                            ]}
                                            onItemClick={(item, e) => {
                                                e.preventDefault()
                                                e.stopPropagation()
                                                setActiveId(item.id)
                                            }}
                                            setCollapsed={setCollapsed}
                                            versionMark={<VersionMark severity={severities.SUCCESS} versionText='Version 2.02' />}
                                        />
                                    }
                                />
                                <div id='rootPreviewApp' />
                            </ThemeProvider>
                        </div>
                    </Column>
                    <Column css={styles.border} size={3}>
                        <div css={styles.rightPanel} style={{ height }}>
                            <div css={styles.colors}>
                                <SimpleStripTable
                                    rows={Object.entries(colorPalette).map((colorArray) => ({
                                        attribute: colorArray[0],
                                        value: (
                                            <ColorPicker
                                                defaultColor={(colorArray[1] === undefined ? 'rgba(255,255,255,0)' : colorArray[1]) as RGBColor}
                                                onColorChange={(color) => handleColorChange(colorArray[0], color)}
                                            />
                                        ),
                                    }))}
                                />
                            </div>
                            <div css={styles.buttons}>
                                {defaultColorPalette && (
                                    <ThemeProvider theme={getThemeTemplate(defaultColorPalette, defaultLogo)}>
                                        <Button onClick={handleReset}>{_(g.reset)}</Button>
                                        <Button onClick={handleThemeUpdate} variant='primary'>
                                            {_(g.saveChanges)}
                                        </Button>
                                    </ThemeProvider>
                                )}
                            </div>
                        </div>
                    </Column>
                </Row>
            )}
            {!colorPalette && (
                <div
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: '100%',
                        height: '100%',
                    }}
                >
                    <IconLoader size={40} type='secondary' />
                </div>
            )}
        </div>
    )
})

PreviewApp.displayName = 'PreviewApp'

export default PreviewApp
