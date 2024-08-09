import React, { FC, useContext, useEffect, useMemo } from 'react'
import ReactDOM from 'react-dom'
import { Controller, SubmitHandler, useForm } from 'react-hook-form'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'

import BottomPanel from '@shared-ui/components/Layout/BottomPanel/BottomPanel'
import { Row } from '@shared-ui/components/Atomic/SimpleStripTable/SimpleStripTable.types'
import FormSelect, { selectAligns, selectSizes } from '@shared-ui/components/Atomic/FormSelect'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Button from '@shared-ui/components/Atomic/Button'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import AppContext from '@shared-ui/app/share/AppContext'
import { useIsMounted } from '@shared-ui/common/hooks'

import { Props } from './Tab1.types'
import { messages as t } from '@/containers/Configuration/ConfigurationPage.i18n'
import { messages as g } from '../../../Global.i18n'
import { Inputs } from '@/containers/Configuration/ConfigurationPage.types'
import { CombinedStoreType } from '@/store/store'
import { setTheme } from '@/containers/App/slice'
import notificationId from '@/notificationId'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { resetForm } = props
    const { collapsed } = useContext(AppContext)
    const isMounted = useIsMounted()
    const dispatch = useDispatch()

    const appStore = useSelector((state: CombinedStoreType) => state.app)

    const options = useMemo(() => appStore.configuration.themes.map((t) => ({ value: t, label: t })), [appStore.configuration.themes])
    const defTheme = useMemo(() => options.find((o) => o.value === appStore.configuration?.theme) || options[0], [appStore, options])

    const {
        handleSubmit,
        formState: { errors, isDirty, dirtyFields },
        getValues,
        reset,
        control,
    } = useForm<Inputs>({
        mode: 'all',
        reValidateMode: 'onSubmit',
        values: {
            theme: defTheme,
        },
    })

    useEffect(() => {
        if (resetForm) {
            reset()
        }
    }, [reset, resetForm])

    const rows: Row[] = [
        {
            attribute: _(t.theme),
            value: (
                <Controller
                    control={control}
                    name='theme'
                    render={({ field: { onChange, name, value, ref } }) => (
                        <FormSelect
                            inlineStyle
                            align={selectAligns.RIGHT}
                            defaultValue={defTheme}
                            name={name}
                            onChange={onChange}
                            options={options}
                            ref={ref}
                            size={selectSizes.SMALL}
                            value={value}
                        />
                    )}
                />
            ),
        },
    ]

    const onSubmit: SubmitHandler<Inputs> = (data) => {
        dispatch(setTheme(data.theme.value))

        Notification.success(
            { title: _(t.configurationUpdated), message: _(t.configurationUpdatedMessage) },
            { notificationId: notificationId.HUB_CONFIGURATION_UPDATE }
        )
    }

    return (
        <div>
            <form onSubmit={handleSubmit(onSubmit)}>
                <SimpleStripTable rows={rows} />
            </form>
            {isMounted &&
                document.querySelector('#modal-root') &&
                ReactDOM.createPortal(
                    <BottomPanel
                        actionPrimary={
                            <Button disabled={Object.keys(errors).length > 0} onClick={() => onSubmit(getValues())} variant='primary'>
                                {_(g.saveChanges)}
                            </Button>
                        }
                        actionSecondary={
                            <Button onClick={() => reset()} variant='secondary'>
                                {_(g.reset)}
                            </Button>
                        }
                        attribute={_(g.changesMade)}
                        leftPanelCollapsed={collapsed}
                        show={isDirty}
                        value={`${Object.keys(dirtyFields).length} ${Object.keys(dirtyFields).length > 1 ? _(t.settings) : _(t.setting)}`}
                    />,
                    document.querySelector('#modal-root') as Element
                )}
        </div>
    )
}

export default Tab1
