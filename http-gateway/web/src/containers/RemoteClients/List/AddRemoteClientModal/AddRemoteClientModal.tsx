import React, { FC, useCallback, useState } from 'react'
import { SubmitHandler, useForm } from 'react-hook-form'
import { useIntl } from 'react-intl'
import cloneDeep from 'lodash/cloneDeep'
import isFunction from 'lodash/isFunction'

import Modal from '@shared-ui/components/Atomic/Modal'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import { Column, Row } from '@shared-ui/components/Atomic/Grid'
import Button from '@shared-ui/components/Atomic/Button'
import * as styles from '@shared-ui/components/Atomic/Modal/components/ProvisionDeviceModal/ProvisionDeviceModal.styles'
import { fetchApi } from '@shared-ui/common/services'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import CopyElement from '@shared-ui/components/Atomic/CopyElement'
import { IconCopy } from '@shared-ui/components/Atomic'
import { copyToClipboard } from '@shared-ui/common/utils'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'

import { Props, defaultProps, Inputs, ClientInformationLineType } from './AddRemoteClientModal.types'
import { messages as t } from '../../RemoteClients.i18n'

const AddRemoteClientModal: FC<Props> = (props) => {
    const { footerActions, defaultClientUrl, defaultClientName, onClose, onFormSubmit, ...rest } = { ...defaultProps, ...props }
    const { formatMessage: _ } = useIntl()
    const [versionLoading, setVersionLoading] = useState(false)
    const [clientInformation, setClientInformation] = useState<ClientInformationLineType[] | undefined>(undefined)

    const {
        register,
        handleSubmit,
        formState: { errors },
        getValues,
        reset,
    } = useForm<Inputs>({
        mode: 'all',
        values: {
            clientName: defaultClientName || '',
            clientUrl: defaultClientUrl || '',
        },
    })

    const handleDone = useCallback(() => {
        const formValues = getValues()
        const finalData = cloneDeep(clientInformation)

        finalData?.push(
            {
                attribute: _(t.clientName),
                attributeKey: 'clientName',
                value: formValues.clientName,
            },
            {
                attribute: _(t.clientUrl),
                attributeKey: 'clientUrl',
                value: formValues.clientUrl,
            }
        )

        finalData && onFormSubmit(finalData)
        setClientInformation(undefined)
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [clientInformation, getValues])

    const handleClose = useCallback(() => {
        reset()
        isFunction(onClose) && onClose()
    }, [onClose, reset])

    const onSubmit: SubmitHandler<Inputs> = (data) => {
        setVersionLoading(true)
        const url = data.clientUrl.endsWith('/') ? data.clientUrl.slice(0, -1) : data.clientUrl

        fetchApi(`${url}/.well-known/configuration`, {
            useToken: false,
        })
            .then((result) => {
                const version = result.data?.version

                if (version) {
                    setClientInformation([
                        {
                            attribute: _(t.version),
                            attributeKey: 'version',
                            value: version,
                        },
                    ])
                }

                setVersionLoading(false)

                Notification.success({ title: _(t.success), message: _(t.clientSuccess) })
            })
            .catch((e) => {
                setVersionLoading(false)
                Notification.error({ title: _(t.error), message: _(t.clientError) })
            })
    }

    const DeviceInformationLine = (data: ClientInformationLineType) => (
        <div css={styles.line}>
            <div css={styles.attribute}>{data.attribute}</div>
            <div css={styles.value}>
                {data.value}
                <Tooltip content={_(t.copy)} css={styles.icon} id={`tooltip-group-${data.attribute}`} portalTarget={undefined}>
                    <IconCopy onClick={() => copyToClipboard(data?.copyValue || data.value, data.certFormat)} />
                </Tooltip>
            </div>
        </div>
    )

    const RenderBoxInfo = () => {
        if (clientInformation) {
            const dataForCopy = clientInformation.map((i) => ({ attribute: i.attribute, value: i.copyValue || i.value, attributeKey: i.attributeKey }))
            return (
                <div>
                    <div css={styles.codeInfoHeader}>
                        <h3 css={styles.title}>{_(t.clientInformation)}</h3>
                        <CopyElement textToCopy={JSON.stringify(dataForCopy)} />
                    </div>
                    <div css={[styles.getCodeBox, styles.codeBoxWithLines]}>
                        {clientInformation?.map((info: ClientInformationLineType) => (
                            <DeviceInformationLine key={info.attributeKey} {...info} />
                        ))}
                    </div>
                </div>
            )
        } else {
            return (
                <div css={styles.getCodeBox}>
                    <Button
                        disabled={!!errors.clientUrl || !!errors.clientName}
                        htmlType='submit'
                        loading={versionLoading}
                        loadingText={_(t.addClientButtonLoading)}
                        size='big'
                        variant='primary'
                    >
                        {_(t.addClientButton)}
                    </Button>
                </div>
            )
        }
    }

    const renderBody = () => (
        <form onSubmit={handleSubmit(onSubmit)}>
            <Row>
                <Column size={6}>
                    <FormGroup error={errors.clientName ? _(t.clientNameError) : undefined} id='client-name'>
                        <FormLabel text={_(t.clientName)} />
                        <FormInput
                            placeholder={_(t.clientName)}
                            {...register('clientName', { validate: (val) => val !== '' })}
                            disabled={!!versionLoading || !!clientInformation}
                        />
                    </FormGroup>
                </Column>
                <Column size={6}>
                    <FormGroup error={errors.clientUrl ? _(t.clientUrlError) : undefined} id='client-url'>
                        <FormLabel text={_(t.clientUrl)} />
                        <FormInput
                            placeholder={_(t.clientUrl)}
                            {...register('clientUrl', { validate: (val) => val !== '' })}
                            disabled={!!versionLoading || !!clientInformation}
                        />
                    </FormGroup>
                </Column>
            </Row>
            <RenderBoxInfo />
        </form>
    )

    return (
        <Modal
            {...rest}
            appRoot={document.getElementById('root')}
            footerActions={
                clientInformation
                    ? [
                          {
                              label: _(t.done),
                              onClick: handleDone,
                              variant: 'primary',
                          },
                      ]
                    : undefined
            }
            onClose={handleClose}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            title={_(t.addNewClient)}
        />
    )
}

AddRemoteClientModal.displayName = 'AddRemoteClientModal'
AddRemoteClientModal.defaultProps = defaultProps

export default AddRemoteClientModal
