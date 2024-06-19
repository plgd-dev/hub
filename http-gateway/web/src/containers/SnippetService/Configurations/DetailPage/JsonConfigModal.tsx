import React, { FC, useCallback, useEffect, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'

import Modal, { ModalStrippedLine } from '@shared-ui/components/Atomic/Modal'
import ModalFooter from '@plgd/shared-ui/src/components/Atomic/Modal/components/ModalFooter'
import Button from '@shared-ui/components/Atomic/Button'
import FormInput, { inputSizes } from '@shared-ui/components/Atomic/FormInput'
import TimeoutControl from '@shared-ui/components/Atomic/TimeoutControl'
import { security } from '@shared-ui/common/services'
import { WellKnownConfigType } from '@shared-ui/common/hooks'
import Editor from '@shared-ui/components/Atomic/Editor'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { messages as g } from '@/containers/Global.i18n'
import { messages as confT } from '../../SnippetService.i18n'
import { ResourceTypeEnhanced } from './Tabs/Tab1/Tab1.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'

type Props = {
    resource?: ResourceTypeEnhanced
    dataTestId?: string
    disabled: boolean
    isUpdateModal?: boolean
    loading?: boolean
    onClose?: () => void
    onSubmit: (data: ResourceTypeEnhanced) => void
    show: boolean
}

const JsonConfigModal: FC<Props> = (props) => {
    const { resource, dataTestId, disabled, onClose, onSubmit, loading, isUpdateModal, show } = props

    const { formatMessage: _ } = useIntl()

    const handleClose = () => {
        isFunction(onClose) && onClose()
    }

    const wellKnownConfig = security.getWellKnownConfig() as WellKnownConfigType & {
        defaultCommandTimeToLive: number
    }

    const [href, setHref] = useState(resource?.href || '')
    const [ttl, setTtl] = useState(parseInt(resource?.timeToLive || '0', 10))
    const [ttlHasError, setTtlHasError] = useState(false)
    const [modalCode, setModalCode] = useState(false)
    const [jsonData, setJsonData] = useState<object | string | undefined>(undefined)
    const [interfaceJsonError, setInterfaceJsonError] = useState(false)

    const editor = useRef()

    const handleOnEditorChange = useCallback((json: object) => {
        json && setJsonData(json)
    }, [])

    const handleModalContentViewChange = useCallback(() => {
        setModalCode((prevState) => !prevState)
    }, [])

    const handleOnEditorError = useCallback((error: any) => setInterfaceJsonError(error.length > 0), [])

    useEffect(() => {
        if (resource?.content) {
            if (typeof resource.content === 'object') {
                setJsonData(resource.content)
                // @ts-ignore
                editor?.current?.current?.set(resource.content)
            } else {
                const dataString = resource.content.toString()
                // @ts-ignore
                editor?.current?.current?.setText(dataString)
                setJsonData(dataString)
            }
        } else {
            setJsonData('')
        }
    }, [resource?.content])

    useEffect(() => {
        setHref(resource?.href || '')
    }, [resource?.href])

    useEffect(() => {
        setTtl(parseInt(resource?.timeToLive || '0'))
    }, [resource?.timeToLive, wellKnownConfig?.defaultCommandTimeToLive])

    const renderBody = () => {
        return (
            <div style={{ height: '100%', display: 'flex', flexDirection: 'column', flex: '1 1 auto' }}>
                <ModalStrippedLine
                    component={<FormInput compactFormComponentsView={false} onChange={(e) => setHref(e.target.value)} size={inputSizes.SMALL} value={href} />}
                    label={_(g.href)}
                />
                <ModalStrippedLine
                    component={
                        <TimeoutControl
                            compactFormComponentsView={false}
                            defaultTtlValue={ttl}
                            defaultValue={ttl}
                            disabled={disabled || loading}
                            i18n={{
                                default: _(t.default),
                                duration: _(t.duration),
                                placeholder: _(t.placeholder),
                                unit: _(t.unit),
                            }}
                            onChange={setTtl}
                            onTtlHasError={setTtlHasError}
                            size='small'
                            ttlHasError={ttlHasError}
                            unitMenuPortalTarget={document.body}
                            unitMenuZIndex={32}
                        />
                    }
                    label={resource?.href || ''}
                    smallPadding={true}
                />

                <Spacer style={{ flex: '1 1 auto' }} type='pt-4'>
                    <Editor
                        dataTestId={dataTestId?.concat('-editor')}
                        disabled={disabled}
                        editorRef={(node: any) => {
                            editor.current = node
                        }}
                        height={modalCode ? '100%' : '350px'}
                        i18n={{
                            viewText: modalCode ? _(g.compactView) : _(g.fullView),
                        }}
                        json={jsonData || {}}
                        onChange={handleOnEditorChange}
                        onError={handleOnEditorError}
                        onViewChange={handleModalContentViewChange}
                    />
                </Spacer>
            </div>
        )
    }

    const handleSubmit = () => {
        isFunction(onSubmit) &&
            onSubmit({
                href,
                timeToLive: ttl,
                content: jsonData,
                id: resource?.id,
            })
    }

    const renderFooter = () => (
        <ModalFooter
            right={
                <div className='modal-buttons'>
                    <Button
                        className='modal-button'
                        dataTestId={dataTestId?.concat('-confirm-button')}
                        disabled={disabled || interfaceJsonError}
                        loading={loading}
                        onClick={handleSubmit}
                        variant='primary'
                    >
                        {isUpdateModal ? _(g.update) : _(g.create)}
                    </Button>
                </div>
            }
        />
    )

    return (
        <Modal
            appRoot={document.getElementById('root')}
            bodyStyle={{ display: 'flex', flexDirection: 'column', height: '100%' }}
            closeButton={!disabled}
            closeButtonText={_(g.close)}
            closeOnBackdrop={false}
            contentPadding={false}
            dataTestId={dataTestId?.concat('-modal')}
            fullSize={modalCode}
            minWidth={720}
            onClose={!disabled ? handleClose : undefined}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            renderFooter={renderFooter}
            show={show}
            title={isUpdateModal ? _(confT.editResource) : _(confT.createResource)}
            zIndex={30}
        />
    )
}

JsonConfigModal.displayName = 'JsonConfigModal'

export default JsonConfigModal
