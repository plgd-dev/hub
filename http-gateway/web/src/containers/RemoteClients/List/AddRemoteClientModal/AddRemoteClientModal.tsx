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
import { IconCopy } from '@shared-ui/components/Atomic/Icon'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { copyToClipboard } from '@shared-ui/common/utils'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import { DEVICE_AUTH_MODE } from '@shared-ui/app/clientApp/constants'
import Alert from '@shared-ui/components/Atomic/Alert'

import { Props, defaultProps, Inputs, ClientInformationLineType } from './AddRemoteClientModal.types'
import { messages as t } from '../../RemoteClients.i18n'
import notificationId from '@/notificationId'

const AddRemoteClientModal: FC<Props> = (props) => {
    const {
        defaultAuthMode,
        defaultClientInformation,
        defaultClientName,
        defaultClientUrl,
        defaultPreSharedKey,
        defaultPreSharedSubjectId,
        footerActions,
        onClose,
        onFormSubmit,
        ...rest
    } = {
        ...defaultProps,
        ...props,
    }
    const { formatMessage: _ } = useIntl()
    const [versionLoading, setVersionLoading] = useState(false)
    const [clientInformation, setClientInformation] = useState<ClientInformationLineType[] | undefined>(defaultClientInformation)

    const options = useMemo(
        () => [
            { value: DEVICE_AUTH_MODE.X509, label: DEVICE_AUTH_MODE.X509 },
            { value: DEVICE_AUTH_MODE.PRE_SHARED_KEY, label: DEVICE_AUTH_MODE.PRE_SHARED_KEY },
        ],
        []
    )

    useEffect(() => {
        setClientInformation(defaultClientInformation)
    }, [defaultClientInformation])

    const defAuthMode = useMemo(() => options.find((o) => o.value === defaultAuthMode) || options[0], [defaultAuthMode, options])

    const {
        register,
        handleSubmit,
        formState: { errors },
        getValues,
        reset,
        watch,
        control,
        trigger,
        setValue,
    } = useForm<Inputs>({
        mode: 'all',
        reValidateMode: 'onSubmit',
        values: {
            clientName: defaultClientName || '',
            clientUrl: defaultClientUrl || '',
            authMode: defaultAuthMode ? defAuthMode : options[0],
            preSharedSubjectId: defaultPreSharedSubjectId || '',
            preSharedKey: defaultPreSharedKey || '',
        },
    })

    const isEditMode = useMemo(() => !!defaultClientInformation, [defaultClientInformation])

    const handleDone = useCallback(() => {
        const formValues = getValues()
        const finalData = clientInformation ? cloneDeep(clientInformation) : []

        const deviceAuthenticationModeIndex = finalData?.findIndex((i) => i.attributeKey === 'deviceAuthenticationMode')

        if (deviceAuthenticationModeIndex >= 0) {
            finalData[deviceAuthenticationModeIndex].value = formValues.authMode.value
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

        reset()
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
                        {
                            attribute: _(t.deviceAuthenticationMode),
                            attributeKey: 'deviceAuthenticationMode',
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
            .catch((e) => {
                setVersionLoading(false)
                Notification.error(
                    { title: _(t.error), message: _(t.clientError, { remoteClientUrl: url }) },
                    { notificationId: notificationId.HUB_ADD_REMOTE_CLIENT_MODAL_ON_SUBMIT, onClick: () => window.open(url, '_blank') }
                )
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
                            if (info.attributeKey !== 'deviceAuthenticationMode') {
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
        () =>
            (clientInformation &&
                clientInformation.find((item) => item.attributeKey === 'deviceAuthenticationMode')?.value === DEVICE_AUTH_MODE.UNINITIALIZED) ||
            isEditMode,
        [clientInformation, isEditMode]
    )

    const authMode = watch('authMode')

    useEffect(() => {
        trigger().then()

        if (authMode?.value === DEVICE_AUTH_MODE.X509) {
            setValue('preSharedSubjectId', '')
            setValue('preSharedKey', '')
        }

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [authMode])

    const renderBody = () => (
        <form onSubmit={handleSubmit(onSubmit)}>
            {!isEditMode && <Alert css={styles.alert}>{_(t.certificateAcceptDescription)}</Alert>}
            <Row>
                <Column size={6}>
                    <FormGroup error={errors.clientName ? _(t.clientNameError) : undefined} id='client-name'>
                        <FormLabel text={_(t.clientName)} />
                        <FormInput
                            placeholder={_(t.clientName)}
                            {...register('clientName', { validate: (val) => val !== '' })}
                            disabled={versionLoading || (!!clientInformation && !isEditMode)}
                        />
                    </FormGroup>
                </Column>
                <Column size={6}>
                    <FormGroup error={errors.clientUrl ? _(t.clientUrlError) : undefined} id='client-url'>
                        <FormLabel text={_(t.clientUrl)} />
                        <FormInput
                            placeholder={_(t.clientUrl)}
                            {...register('clientUrl', { validate: (val) => val !== '' })}
                            disabled={versionLoading || (!!clientInformation && !isEditMode)}
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
                                    render={({ field: { onChange, name, ref } }) => (
                                        <FormSelect
                                            defaultValue={defaultAuthMode ? defAuthMode : options[0]}
                                            name={name}
                                            onChange={onChange}
                                            options={options}
                                            ref={ref}
                                        />
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
            maxWidth={600}
            onClose={handleClose}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            title={isEditMode ? _(t.editClient) : _(t.addNewClient)}
        />
    )
}

AddRemoteClientModal.displayName = 'AddRemoteClientModal'

export default AddRemoteClientModal
