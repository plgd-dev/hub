import React, { useEffect, useState, useRef, FC } from 'react'
import { useIntl } from 'react-intl'

import Editor from '@shared-ui/components/new/Editor'
import FormSelect from '@shared-ui/components/new/FormSelect'
import Button from '@shared-ui/components/new/Button'
import Modal from '@shared-ui/components/new/Modal'

import DevicesResourcesModalNotifications from '../DevicesResourcesModalNotifications'
import { resourceModalTypes } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesResourcesModal.types'
import ModalStrippedLine from '@plgd/shared-ui/src/components/new/Modal/ModalStrippedLine'
import isFunction from 'lodash/isFunction'

const { UPDATE_RESOURCE } = resourceModalTypes

const DevicesResourcesModal: FC<Props> = (props) => {
    const {
        data,
        deviceId,
        deviceName,
        resourceData,
        onClose,
        retrieving,
        isDeviceOnline,
        isUnregistered,
        loading,
        updateResource,
        createResource,
        type,
        ttlControl,
        confirmDisabled,
        fetchResource,
        show,
    } = { ...defaultProps, ...props }
    const { formatMessage: _ } = useIntl()
    const editor = useRef()
    const [jsonData, setJsonData] = useState<object | undefined>(undefined)
    const [interfaceJsonError, setInterfaceJsonError] = useState(false)

    const disabled = retrieving || loading
    const isUpdateModal = type === UPDATE_RESOURCE
    const updateLabel = !loading ? _(t.update) : _(t.updating)
    const createLabel = !loading ? _(t.create) : _(t.creating)
    const initialInterfaceValue = { value: '', label: _(t.resourceInterfaces) }
    const [selectedInterface, setSelectedInterface] = useState(initialInterfaceValue)

    useEffect(() => {
        const dataToDisplay = resourceData?.data?.content
        setJsonData(dataToDisplay)

        if (resourceData && editor.current) {
            // Set the retrieved JSON object to the editor
            if (typeof resourceData === 'object') {
                // @ts-ignore
                editor?.current?.current?.set(dataToDisplay)
            } else if (typeof resourceData === 'string') {
                // @ts-ignore
                editor?.current?.current?.setText(dataToDisplay)
            }
        }
    }, [resourceData])

    const handleRetrieve = () => {
        fetchResource({
            href: data?.href || '',
            currentInterface: selectedInterface.value,
        })
    }

    const handleSubmit = () => {
        const params = {
            href: data?.href as string,
            currentInterface: selectedInterface.value,
        }

        if (isUpdateModal) {
            updateResource(params, jsonData)
        } else {
            createResource(params, jsonData)
        }
    }

    const handleCleanup = () => {
        setSelectedInterface(initialInterfaceValue)
        setJsonData(undefined)
    }

    const handleOnEditorChange = (json: object) => {
        json && setJsonData(json)
    }

    const handleOnEditorError = (error: any) => setInterfaceJsonError(error.length > 0)

    const renderBody = () => {
        const interfaces = data?.interfaces?.map?.((ifs) => ({ value: ifs, label: ifs })) || []
        interfaces.unshift(initialInterfaceValue)

        return (
            <>
                {data && isUpdateModal && (
                    <DevicesResourcesModalNotifications
                        deviceId={deviceId as string}
                        deviceName={deviceName}
                        href={data?.href as string}
                        isUnregistered={isUnregistered}
                    />
                )}

                <ModalStrippedLine component={deviceId} label={_(t.deviceId)} />

                <ModalStrippedLine component={data?.types?.join(', ')} label={isUpdateModal ? _(t.types) : _(t.supportedTypes)} />

                {isUpdateModal && <ModalStrippedLine component={data?.interfaces?.join(', ')} label={_(t.interfaces)} />}

                <ModalStrippedLine component={ttlControl} label={_(t.commandTimeout)} smallPadding={true} />

                {isUpdateModal && (
                    <ModalStrippedLine
                        component={
                            <FormSelect disabled={disabled || !isDeviceOnline} onChange={setSelectedInterface} options={interfaces} value={selectedInterface} />
                        }
                        componentSize={200}
                        label={_(t.resourceInterfaces)}
                        smallPadding={true}
                    />
                )}

                <div className='m-t-20 m-b-0'>
                    {jsonData && (
                        <Editor
                            disabled={disabled}
                            editorRef={(node: any) => {
                                editor.current = node
                            }}
                            json={jsonData}
                            onChange={handleOnEditorChange}
                            onError={handleOnEditorError}
                        />
                    )}
                </div>
            </>
        )
    }

    const renderFooter = () => {
        const interfaces = data?.interfaces?.map?.((ifs) => ({ value: ifs, label: ifs })) || []
        interfaces.unshift(initialInterfaceValue)

        return (
            <div className='w-100 d-flex justify-content-between align-items-center'>
                <div />
                <div className='modal-buttons'>
                    {isUpdateModal && (
                        <Button className='modal-button' disabled={disabled} loading={retrieving} onClick={handleRetrieve} variant='secondary'>
                            {!retrieving ? _(t.retrieve) : _(t.retrieving)}
                        </Button>
                    )}

                    <Button
                        className='modal-button'
                        disabled={disabled || interfaceJsonError || confirmDisabled}
                        loading={loading}
                        onClick={handleSubmit}
                        variant='primary'
                    >
                        {isUpdateModal ? updateLabel : createLabel}
                    </Button>
                </div>
            </div>
        )
    }

    const handleClose = () => {
        isFunction(onClose) && onClose()
        handleCleanup()
    }

    return (
        <Modal
            appRoot={document.getElementById('root')}
            closeButton={!disabled}
            closeButtonText={_(t.close)}
            onClose={!disabled ? handleClose : undefined}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            renderFooter={renderFooter}
            show={show && !!data && !!jsonData}
            title={data?.href}
        />
    )
}

DevicesResourcesModal.displayName = 'DevicesResourcesModal'
DevicesResourcesModal.defaultProps = defaultProps

export default DevicesResourcesModal
