import React, { useRef, useState } from 'react'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import { ProvisionDeviceModal } from '@shared-ui/components/Atomic/Modal'
import { security } from '@shared-ui/common/services'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'

import { getDeviceAuthCode } from '@/containers/Devices/rest'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import { DEVICE_AUTH_CODE_SESSION_KEY } from '@/constants'

const ProvisionNewDeviceCore = () => {
    const [show, setShow] = useState(false)
    const [fetching, setFetching] = useState(false)
    const [deviceId, setDeviceId] = useState<null | string>(null)
    const [code, setCode] = useState<undefined | string>(undefined)
    const inputRef = useRef<HTMLInputElement>(null)

    const { formatMessage: _ } = useIntl()

    const handleFetch = async () => {
        setFetching(true)
        try {
            const code = await getDeviceAuthCode(deviceId as string)
            setFetching(false)
            setCode(code as string)
        } catch (e: any) {
            Notification.error(
                {
                    title: _(t.deviceAuthCodeError),
                    message: getApiErrorMessage(e.message),
                },
                {
                    notificationId: notificationId.HUB_PROVISION_NEW_DEVICE_CORE_HANDLE_FETCH,
                }
            )

            setFetching(false)
        }
    }

    const openModal = () => {
        setShow(true)
        localStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
        sessionStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
        inputRef?.current?.focus()
    }

    const onClose = () => {
        setShow(false)
        setCode(undefined)
        localStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
        sessionStorage.removeItem(DEVICE_AUTH_CODE_SESSION_KEY)
        setDeviceId(null)
    }

    const { coapGateway: deviceEndpoint, id: hubId, certificateAuthorities } = security.getWellKnowConfig() || {}
    const { providerName } = (security.getDeviceOAuthConfig() as any) || {}

    return (
        <>
            <Button icon={<IconPlus />} onClick={openModal} variant='primary'>
                {_(t.addDevice)}
            </Button>
            <ProvisionDeviceModal
                closeButtonText={_(g.close)}
                deviceAuthCode={code}
                deviceAuthLoading={fetching}
                deviceInformation={
                    code
                        ? [
                              { attribute: _(t.hubId), value: hubId, attributeKey: 'hubId' },
                              { attribute: _(t.deviceEndpoint), value: deviceEndpoint, attributeKey: 'deviceEndpoint' },
                              { attribute: _(t.authorizationCode), value: '******', copyValue: code, attributeKey: 'authorizationCode' },
                              { attribute: _(t.authorizationProvider), value: providerName, attributeKey: 'authorizationProvider' },
                              {
                                  attribute: _(t.certificateAuthorities),
                                  value: '...',
                                  copyValue: certificateAuthorities,
                                  certFormat: true,
                                  attributeKey: 'certificateAuthorities',
                              },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: _(t.close),
                        onClick: onClose,
                        variant: 'primary',
                    },
                ]}
                getDeviceAuthCode={handleFetch}
                i18n={{
                    copy: _(t.copy),
                    deviceId: _(t.deviceId),
                    enterDeviceID: _(t.enterDeviceID),
                    invalidUuidFormat: _(t.invalidUuidFormat),
                    getTheCode: _(t.getTheCode),
                    deviceInformation: _(t.deviceInformation),
                }}
                onClose={!fetching ? onClose : () => {}}
                show={show}
                title={_(t.provisionNewDevice)}
            />
        </>
    )
}

export default ProvisionNewDeviceCore
