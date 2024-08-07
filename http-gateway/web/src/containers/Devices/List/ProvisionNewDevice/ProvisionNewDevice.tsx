import React, { useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { detect } from 'detect-browser'

import Button from '@shared-ui/components/Atomic/Button'
import { ProvisionDeviceModal } from '@shared-ui/components/Atomic/Modal'
import { security } from '@shared-ui/common/services'
import { IconPlus } from '@shared-ui/components/Atomic/Icon'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { isSafari } from '@shared-ui/components/Atomic/_utils/browser'

import { getDeviceAuthCode } from '@/containers/Devices/rest'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import { DEVICE_AUTH_CODE_SESSION_KEY } from '@/constants'

const ProvisionNewDeviceCore = () => {
    const [show, setShow] = useState(false)
    const [fetching, setFetching] = useState(false)
    const [code, setCode] = useState<undefined | string>(undefined)
    const inputRef = useRef<HTMLInputElement>(null)
    const [resetIndex, setResetIndex] = useState(0)

    const { formatMessage: _ } = useIntl()
    const isSafariBrowser = isSafari(detect())

    const handleFetch = async (deviceId: string) => {
        setFetching(true)
        try {
            // @ts-ignore
            let isBrave = await navigator?.brave?.isBrave()

            const code = await getDeviceAuthCode(deviceId, isBrave || isSafariBrowser)
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
        setResetIndex((prev) => prev + 1)
    }

    const { coapGateway: deviceEndpoint, id: hubId, certificateAuthorities } = security.getWellKnownConfig() || {}
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
                resetIndex={resetIndex}
                show={show}
                title={_(t.provisionNewDevice)}
            />
        </>
    )
}

export default ProvisionNewDeviceCore
