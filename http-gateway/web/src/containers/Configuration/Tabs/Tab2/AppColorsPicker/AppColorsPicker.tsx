import React, { FC, useCallback, useEffect, useState } from 'react'
import { useDispatch, useSelector } from 'react-redux'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import Modal from '@shared-ui/components/Atomic/Modal'

import { Props } from './AppColorsPicker.types'
import { setThemeModal } from '@/containers/App/slice'
import { CONFIGURATION_PAGE_FRAME } from '@/constants'
import { messages as t } from '../../../ConfigurationPage.i18n'

const AppColorsPicker: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()

    const [show, setShow] = useState(false)

    const appStore = useSelector((state: any) => state.app)
    const dispatch = useDispatch()

    useEffect(() => {
        if (appStore.configuration.themeModal !== show) {
            setShow(appStore.configuration.themeModal)
        }
    }, [appStore.configuration.themeModal, show])

    const changeView = useCallback(
        (view: boolean) => {
            setShow(view)
            dispatch(setThemeModal(view))
        },
        [dispatch]
    )

    return (
        <>
            <Button onClick={() => changeView(true)} variant='primary'>
                {_(t.colorConfigurator)}
            </Button>
            <Modal
                appRoot={document.getElementById('root')}
                bodyStyle={{
                    overflow: 'hidden',
                }}
                fullSize={true}
                onClose={() => changeView(false)}
                portalTarget={document.getElementById('modal-root')}
                renderBody={
                    <div
                        style={{
                            height: '100%',
                            width: '100%',
                            display: 'flex',
                            flexDirection: 'column',
                            overflow: 'hidden',
                        }}
                    >
                        <iframe
                            src={`${window.location.origin}/${CONFIGURATION_PAGE_FRAME}`}
                            style={{
                                height: '100%',
                                width: '100%',
                                display: 'flex',
                                flexDirection: 'column',
                                overflow: 'hidden',
                                border: 0,
                            }}
                            title={_(t.colorConfigurator)}
                        />
                    </div>
                }
                show={show}
                title={_(t.colorConfigurator)}
            />
        </>
    )
}

AppColorsPicker.displayName = 'AppColorsPicker'

export default AppColorsPicker
