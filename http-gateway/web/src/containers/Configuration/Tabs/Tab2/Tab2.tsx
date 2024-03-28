import React, { FC, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { Controller, SubmitHandler, useForm } from 'react-hook-form'
import { useIntl } from 'react-intl'
import ReactDOM from 'react-dom'
import debounce from 'lodash/debounce'
import { useDispatch, useSelector } from 'react-redux'
import { useTheme } from '@emotion/react'

import { getThemeTemplate } from '@shared-ui/components/Atomic/_theme/template'
import Editor from '@shared-ui/components/Atomic/Editor'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import Button from '@shared-ui/components/Atomic/Button'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { useIsMounted } from '@shared-ui/common/hooks'
import AppContext from '@shared-ui/app/share/AppContext'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput, { inputAligns } from '@shared-ui/components/Atomic/FormInput'
import { isValidHex, ThemeType } from '@shared-ui/components/Atomic/_theme'
import { EditorRefType } from '@shared-ui/components/Atomic/Editor/Editor.types'
import Switch from '@shared-ui/components/Atomic/Switch'
import { getNumberFromPx } from '@shared-ui/components/Atomic/_utils/commonStyles'

import { Props, Inputs } from './Tab2.types'
import { messages as t } from '../../ConfigurationPage.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { setPreviewTheme } from '@/containers/App/slice'
import AppColorsPicker from './AppColorsPicker'

const Tab2: FC<Props> = (props) => {
    const { isTabActive, resetForm } = props
    const { formatMessage: _ } = useIntl()
    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()
    const dispatch = useDispatch()

    const appStore = useSelector((state: any) => state.app)

    const editorRef = useRef<EditorRefType>(null)

    const [loading, setLoading] = useState(false)

    const theme: ThemeType = useTheme()

    const defaultColorPalette = useMemo(() => {
        if (appStore.configuration.previewTheme?.colorPalette) {
            return appStore.configuration.previewTheme?.colorPalette
        } else {
            return theme.colorPalette ?? {}
        }
    }, [appStore.configuration.previewTheme?.colorPalette, theme.colorPalette])

    const {
        handleSubmit,
        formState: { errors },
        getValues,
        reset,
        control,
        register,
        setValue,
        watch,
    } = useForm<Inputs>({
        mode: 'all',
        reValidateMode: 'onSubmit',
        values: {
            themeName: 'custom theme',
            colorPalette: defaultColorPalette,
            logoHeight: 32,
            logoWidth: 140,
            logoSource: '',
            themeFormat: true,
        },
    })

    useEffect(() => {
        if (resetForm) {
            reset()
        }
    }, [reset, resetForm])

    useEffect(() => {
        const values = getValues()
        if (appStore.configuration.previewTheme) {
            const { logo, colorPalette } = appStore.configuration.previewTheme

            if (colorPalette) {
                editorRef?.current?.setValue(colorPalette)
                setValue('colorPalette', colorPalette)
            }

            if (logo && logo.height && logo.height !== values.logoHeight) {
                setValue('logoHeight', typeof logo.height === 'string' ? getNumberFromPx(logo.height) : logo.height)
            }

            if (logo && logo.width && logo.width !== values.logoWidth) {
                setValue('logoWidth', typeof logo.width === 'string' ? getNumberFromPx(logo.width) : logo.width)
            }

            if (logo && logo.source && logo.source !== values.logoSource) {
                setValue('logoSource', logo.source)
            }
        }
    }, [appStore.configuration.previewTheme, getValues, setValue])

    const getBase64 = useCallback(
        (file: any) =>
            new Promise((resolve) => {
                let baseURL: any = ''
                // Make new FileReader
                let reader = new FileReader()

                // Convert the file to base64 text
                reader.readAsDataURL(file)

                // on reader load something...
                reader.onload = () => {
                    // Make a fileInfo Object
                    baseURL = reader.result
                    resolve(baseURL)
                }
            }),
        []
    )

    const handleFileInput = (e: any) => {
        getBase64(e.target.files[0]).then((result) => {
            if (typeof result === 'string') {
                setValue('logoSource', result)
            }
        })
    }

    const onPreviewSubmit = debounce((jsonPalette) => {
        const values = getValues()

        if (Object.values(jsonPalette).every(isValidHex)) {
            dispatch(
                setPreviewTheme(
                    getThemeTemplate(values.colorPalette, {
                        height: values.logoHeight,
                        width: values.logoWidth,
                        source: values.logoSource,
                    })
                )
            )
        }
    }, 1000)

    const rows: Row[] = [
        {
            attribute: _(t.themeName),
            value: (
                <FormGroup errorTooltip fullSize error={errors.themeName ? _(t.themeNameError) : undefined} id='theme-name' marginBottom={false}>
                    <FormInput
                        inlineStyle
                        align={inputAligns.RIGHT}
                        placeholder={_(t.themeName)}
                        {...register('themeName', { required: true, validate: (val) => val !== '' })}
                    />
                </FormGroup>
            ),
        },
        {
            attribute: _(t.colorPalette),
            autoHeight: true,
            value: (
                <Controller
                    control={control}
                    name='colorPalette'
                    render={({ field: { onChange, value } }) => (
                        <Spacer style={{ width: '100%' }} type='py-4'>
                            <Editor
                                height='500px'
                                json={value}
                                onBlur={(data) => {
                                    const json = JSON.parse(data)
                                    onChange(json)
                                    onPreviewSubmit(json)
                                }}
                                ref={editorRef}
                            />
                        </Spacer>
                    )}
                />
            ),
        },
        {
            attribute: _(t.logoSource),
            value: <input name='file' onChange={handleFileInput} type='file' />,
        },
        {
            attribute: _(t.logoHeight),
            value: (
                <FormGroup errorTooltip fullSize error={errors.logoHeight ? _(t.logoHeightError) : undefined} id='logo-height' marginBottom={false}>
                    <FormInput
                        inlineStyle
                        align={inputAligns.RIGHT}
                        placeholder={_(t.logoHeight)}
                        type='number'
                        {...register('logoHeight', { required: true, valueAsNumber: true })}
                    />
                </FormGroup>
            ),
        },
        {
            attribute: _(t.logoWidth),
            value: (
                <FormGroup errorTooltip fullSize error={errors.logoWidth ? _(t.logoWidthError) : undefined} id='logo-width' marginBottom={false}>
                    <FormInput
                        inlineStyle
                        align={inputAligns.RIGHT}
                        placeholder={_(t.logoWidth)}
                        type='number'
                        {...register('logoWidth', { required: true, valueAsNumber: true })}
                    />
                </FormGroup>
            ),
        },
        {
            attribute: _(t.generateThemeJson),
            value: <Switch {...register('themeFormat')} />,
        },
    ]

    const logoSource = watch('logoSource')
    const logoHeight = watch('logoHeight')
    const logoWidth = watch('logoWidth')

    useEffect(() => {
        if (logoSource && logoHeight > 0 && logoWidth) {
            const values = getValues()

            onPreviewSubmit(values.colorPalette)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [logoHeight, logoSource, logoWidth])

    const onSubmit: SubmitHandler<Inputs> = () => {
        const values = getValues()
        setLoading(true)
        const customThemeName = values.themeName.replace(/\s+/g, '_').toLowerCase()
        const themeData = {
            [customThemeName]: getThemeTemplate(values.colorPalette, {
                height: values.logoHeight,
                width: values.logoWidth,
                source: values.logoSource,
            }),
        }

        const fileName = `${customThemeName}.json`
        const data = new Blob(
            [
                JSON.stringify(
                    values.themeFormat
                        ? {
                              defaultTheme: customThemeName,
                              themes: [themeData],
                          }
                        : themeData
                ),
            ],
            { type: 'text/json' }
        )
        const jsonURL = window.URL.createObjectURL(data)
        const link = document.createElement('a')
        document.body.appendChild(link)
        link.href = jsonURL
        link.setAttribute('download', fileName)
        link.click()
        document.body.removeChild(link)
        setLoading(false)
    }

    const handleReset = useCallback(() => {
        reset()
        dispatch(setPreviewTheme(undefined))
        setValue('colorPalette', theme.colorPalette)
        editorRef?.current?.setValue(theme.colorPalette)
    }, [dispatch, reset, setValue, theme.colorPalette])

    return (
        <Spacer type='pb-8'>
            <Spacer type='mb-4'>
                <AppColorsPicker onChange={(c: any) => console.log(c)} />
            </Spacer>
            <form onSubmit={handleSubmit(onSubmit)}>
                <SimpleStripTable leftColSize={6} rightColSize={6} rows={rows} />
            </form>
            {isMounted &&
                document.querySelector('#innerFooterPortalTarget') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button
                                disabled={Object.keys(errors).length > 0}
                                loading={loading}
                                loadingText={_(g.loading)}
                                onClick={() => onSubmit(getValues())}
                                variant='primary'
                            >
                                {_(t.generate)}
                            </Button>
                        }
                        actionSecondary={
                            <Button disabled={loading} onClick={handleReset} variant='secondary'>
                                {_(t.reset)}
                            </Button>
                        }
                        leftPanelCollapsed={collapsed}
                        show={isTabActive}
                    />,
                    document.querySelector('#innerFooterPortalTarget') as Element
                )}
        </Spacer>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
