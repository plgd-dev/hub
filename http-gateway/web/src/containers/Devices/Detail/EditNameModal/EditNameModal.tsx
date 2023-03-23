import React, { FC, useEffect, useState } from 'react'
import { Props } from './EditNameModal.types'
import * as styles from './EditNameModal.styles'
import Modal from '@shared-ui/components/new/Modal'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { useIntl } from 'react-intl'
import Button from '@plgd/shared-ui/src/components/new/Button'
import FormGroup from '@shared-ui/components/new/FormGroup'
import FormLabel from '@shared-ui/components/new/FormLabel'
import FormInput from '@shared-ui/components/new/FormInput'
import isFunction from 'lodash/isFunction'

const EditNameModal: FC<Props> = (props) => {
    const { deviceName, deviceNameLoading, show, handleClose, handleSubmit } = props
    const { formatMessage: _ } = useIntl()
    const [name, setName] = useState(deviceName)

    useEffect(() => {
        setName(deviceName)
    }, [deviceName])

    const handleReset = () => {
        setName(deviceName)
    }

    const handleSubmitFunc = () => {
        isFunction(handleSubmit) && handleSubmit(name)
    }

    const renderBody = () => (
        <div css={styles.body}>
            <FormGroup id='device-name'>
                <FormLabel text={_(t.name)} />
                <FormInput onChange={(e) => setName(e.target.value)} placeholder={_(t.deviceName)} value={name} />
            </FormGroup>
        </div>
    )

    const renderFooter = () => (
        <div className='w-100 d-flex justify-content-between align-items-center'>
            <div />
            <div className='modal-buttons'>
                <Button className='modal-button' onClick={handleReset} variant='secondary'>
                    {_(t.reset)}
                </Button>

                <Button className='modal-button' loading={deviceNameLoading} loadingText={_(t.savingChanges)} onClick={handleSubmitFunc} variant='primary'>
                    {_(t.saveChange)}
                </Button>
            </div>
        </div>
    )

    return (
        <Modal
            appRoot={document.getElementById('root')}
            closeButtonText={_(t.close)}
            contentPadding={false}
            onClose={handleClose}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            renderFooter={renderFooter}
            show={show}
            title={`${_(t.edit)} ${deviceName}`}
        />
    )
}

EditNameModal.displayName = 'EditNameModal'

export default EditNameModal
