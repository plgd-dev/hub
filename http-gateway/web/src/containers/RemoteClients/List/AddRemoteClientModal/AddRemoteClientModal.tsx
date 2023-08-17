import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { Controller, SubmitHandler, useForm } from 'react-hook-form'
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
import { FormSelect, IconCopy } from '@shared-ui/components/Atomic'
import { copyToClipboard } from '@shared-ui/common/utils'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'

import { Props, defaultProps, Inputs, ClientInformationLineType } from './AddRemoteClientModal.types'
import { messages as t } from '../../RemoteClients.i18n'
import notificationId from '@/notificationId'

const AddRemoteClientModal: FC<Props> = (props) => {
    const { footerActions, defaultClientUrl, defaultClientName, onClose, onFormSubmit, ...rest } = { ...defaultProps, ...props }
    const { formatMessage: _ } = useIntl()
    const [versionLoading, setVersionLoading] = useState(false)
    const [clientInformation, setClientInformation] = useState<ClientInformationLineType[] | undefined>(undefined)

    const options = useMemo(
        () => [
            { value: DEVICE_AUTH_MODE.X509, label: DEVICE_AUTH_MODE.X509 },
            { value: DEVICE_AUTH_MODE.PRE_SHARED_KEY, label: DEVICE_AUTH_MODE.PRE_SHARED_KEY },
        ],
        []
    )

    const {
        register,
        handleSubmit,
        formState: { errors },
        getValues,
        reset,
        watch,
        control,
        trigger,
    } = useForm<Inputs>({
        mode: 'all',
        reValidateMode: 'onSubmit',
        values: {
            clientName: defaultClientName || '',
            clientUrl: defaultClientUrl || '',
            authMode: options[0],
            preSharedSubjectId: '',
            preSharedKey: '',
        },
    })

    const handleDone = useCallback(() => {
        const formValues = getValues()
        const finalData = clientInformation ? cloneDeep(clientInformation) : []

        const authenticationModeIndex = finalData?.findIndex((i) => i.attributeKey === 'authenticationMode')

        if (authenticationModeIndex >= 0) {
            finalData[authenticationModeIndex].value = formValues.authMode.value
        }

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
            },
            {
                attribute: _(t.subjectId),
                attributeKey: 'preSharedSubjectId',
                value: formValues.preSharedSubjectId,
            },
            {
                attribute: _(t.key),
                attributeKey: 'preSharedKey',
                value: formValues.preSharedKey || '',
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

                console.log(result.data)

                if (version) {
                    setClientInformation([
                        {
                            attribute: _(t.version),
                            attributeKey: 'version',
                            value: version,
                        },
                        {
                            attribute: _(t.deviceAuthenticationMode),
                            attributeKey: 'authenticationMode',
                            value: result.data?.deviceAuthenticationMode,
                        },
                    ])
                }

                setVersionLoading(false)
                Notification.success(
                    { title: _(t.success), message: _(t.clientSuccess) },
                    { notificationId: notificationId.HUB_ADD_REMOTE_CLIENT_MODAL_ON_SUBMIT }
                )
            })
            .catch(() => {
                setVersionLoading(false)
                Notification.error({ title: _(t.error), message: _(t.clientError) }, { notificationId: notificationId.HUB_ADD_REMOTE_CLIENT_MODAL_ON_SUBMIT })
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
                        {clientInformation?.map((info: ClientInformationLineType) => {
                            if (info.attributeKey !== 'authenticationMode') {
                                return <DeviceInformationLine key={info.attributeKey} {...info} />
                            }
                            return null
                        })}
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

    const isUninitialized = useMemo(
        () => clientInformation && clientInformation.find((item) => item.attributeKey === 'authenticationMode')?.value === DEVICE_AUTH_MODE.UNINITIALIZED,
        [clientInformation]
    )

    const authMode = watch('authMode')

    useEffect(() => {
        trigger().then()
    }, [authMode])

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
            {isUninitialized && (
                <>
                    <div css={styles.codeInfoHeader}>
                        <h3 css={styles.title}>{_(t.config)}</h3>
                    </div>
                    <Row>
                        <Column size={6}>
                            <FormGroup id='client-auth'>
                                <FormLabel text={_(t.deviceAuthenticationMode)} />
                                <Controller
                                    control={control}
                                    name='authMode'
                                    render={({ field: { onChange, onBlur, name, ref }, fieldState: { invalid, isTouched, isDirty, error } }) => (
                                        <FormSelect defaultValue={options[0]} name={name} onChange={onChange} options={options} ref={ref} />
                                    )}
                                />
                            </FormGroup>
                        </Column>
                    </Row>
                </>
            )}
            {isUninitialized && authMode?.value === DEVICE_AUTH_MODE.PRE_SHARED_KEY && (
                <Row>
                    <Column size={6}>
                        <FormGroup error={errors.preSharedSubjectId ? _(t.preSharedSubjectIdError) : undefined} id='subject-id'>
                            <FormLabel text={_(t.subjectId)} />
                            <FormInput placeholder={_(t.subjectId)} {...register('preSharedSubjectId', { validate: (val) => val !== '' })} />
                        </FormGroup>
                    </Column>
                    <Column size={6}>
                        <FormGroup error={errors.preSharedKey ? _(t.preSharedKeyError) : undefined} id='key'>
                            <FormLabel text={_(t.key)} />
                            <FormInput placeholder={_(t.key)} {...register('preSharedKey', { required: true, validate: (val) => val !== '' })} />
                        </FormGroup>
                    </Column>
                </Row>
            )}
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
                              disabled: Object.keys(errors).length > 0,
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
