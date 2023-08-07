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
import { fetchApi, security } from '@shared-ui/common/services'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import CopyElement from '@shared-ui/components/Atomic/CopyElement'
import { IconCopy } from '@shared-ui/components/Atomic'
import { copyToClipboard } from '@shared-ui/common/utils'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'

import { Props, defaultProps, Inputs, ClientInformationLineType } from './AddRemoteClientModal.types'
import { messages as t } from '../../RemoteClients.i18n'

const AddRemoteClientModal: FC<Props> = (props) => {
    const { footerActions, defaultClientIP, defaultClientName, onClose, onFormSubmit, ...rest } = { ...defaultProps, ...props }
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
            clientIP: defaultClientIP || '',
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
                attribute: _(t.clientIP),
                attributeKey: 'clientIP',
                value: formValues.clientIP,
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

    const wellKnownConfig = security.getWellKnowConfig()

    const onSubmit: SubmitHandler<Inputs> = (data) => {
        setVersionLoading(true)
        const url = data.clientIP.endsWith('/') ? data.clientIP.slice(0, -1) : data.clientIP

        fetchApi(`${url}/.well-known/configuration`, {
            useToken: false,
        })
            .then((result) => {
                const version = result.data?.version
                // setVersionLoading(false)

                if (version) {
                    console.log(wellKnownConfig)
                    const { clientId, scopes = [], audience } = wellKnownConfig.deviceOauthClient
                    const audienceParam = audience ? `&audience=${audience}` : ''
                    const deviceId = '00000000-0000-0000-0000-000000000000'

                    const AuthUserManager = security.getUserManager()

                    AuthUserManager.metadataService.getAuthorizationEndpoint().then((authorizationEndpoint: string) => {
                        console.log(authorizationEndpoint)
                        console.log(
                            // https://auth.plgd.cloud/realms/shared/protocol/openid-connect/auth?response_type=code&client_id=cYN3p6lwNcNlOvvUhz55KvDZLQbJeDr5&scope=offline_access&redirect_uri=https://212.89.237.161:50080/devices&device_id=606bc01c-445a-4335-88f3-504b65c14f14
                            `${authorizationEndpoint}?response_type=code&client_id=${clientId}&scope=${scopes}${audienceParam}&redirect_uri=${url}/devices&device_id=${deviceId}`
                        )

                        fetchApi(
                            `${authorizationEndpoint}?response_type=code&client_id=${clientId}&scope=${scopes}${audienceParam}&redirect_uri=${url}/devices&device_id=${deviceId}`
                        )
                            .then((result) => {
                                console.log('result')
                                console.log(result)
                            })
                            .catch((e) => {
                                setVersionLoading(false)
                                console.log('!!!Error')
                                console.log(e)
                                // Notification.error({ title: _(t.error), message: _(t.clientError) })
                            })
                    })

                    // setClientInformation([
                    //     {
                    //         attribute: _(t.version),
                    //         attributeKey: 'version',
                    //         value: version,
                    //     },
                    // ])
                }

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
                        disabled={!!errors.clientIP || !!errors.clientName}
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
                    <FormGroup error={errors.clientName ? _(t.clientNameError) : undefined} id='device-name'>
                        <FormLabel text={_(t.clientName)} />
                        <FormInput
                            placeholder={_(t.clientName)}
                            {...register('clientName', { validate: (val) => val !== '' })}
                            disabled={!!versionLoading || !!clientInformation}
                        />
                    </FormGroup>
                </Column>
                <Column size={6}>
                    <FormGroup error={errors.clientIP ? _(t.clientIPError) : undefined} id='device-ip'>
                        <FormLabel text={_(t.clientIP)} />
                        <FormInput
                            placeholder={_(t.clientIP)}
                            {...register('clientIP', { validate: (val) => val !== '' })}
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
