import React, { useRef, useState } from 'react'
import { getDeviceAuthCode } from '@/containers/Devices/rest'
import { showErrorToast } from '@shared-ui/components/new/Toast/Toast'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { useIntl } from 'react-intl'
import Button from '@shared-ui/components/new/Button'
import { ProvisionDeviceModal } from '@shared-ui/components/new/Modal'
import { security } from '@shared-ui/common/services'
import { Icon } from '@shared-ui/components/new/Icon'

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
            showErrorToast({
                title: _(t.deviceAuthCodeError),
                message: e.message,
            })
            setFetching(false)
        }
    }

    const openModal = () => {
        setShow(true)
        inputRef?.current?.focus()
    }

    const onClose = () => {
        setShow(false)
        setCode(undefined)
        setDeviceId(null)
    }

    const { coapGateway: deviceEndpoint, id: hubId, certificateAuthorities } = security.getWellKnowConfig() || {}
    const { providerName } = (security.getDeviceOAuthConfig() as any) || {}

    return (
        <>
            <Button icon={<Icon icon='plus' />} onClick={openModal} variant='primary'>
                {_(t.addDevice)}
            </Button>
            <ProvisionDeviceModal
                closeButtonText={_(t.cancel)}
                deviceAuthCode={code}
                deviceAuthLoading={fetching}
                deviceInformation={
                    code
                        ? [
                              { attribute: _(t.hubId), value: hubId },
                              { attribute: _(t.deviceEndpoint), value: deviceEndpoint },
                              { attribute: _(t.authorizationCode), value: '******', copyValue: code },
                              { attribute: _(t.authorizationProvider), value: providerName },
                              { attribute: _(t.certificateAuthorities), value: '...', copyValue: certificateAuthorities, certFormat: true },
                          ]
                        : undefined
                }
                footerActions={[
                    {
                        label: _(t.cancel),
                        onClick: () => setShow(false),
                        variant: 'tertiary',
                    },
                    {
                        label: _(t.delete),
                        onClick: () => setShow(false),
                        variant: 'primary',
                    },
                ]}
                getDeviceAuthCode={handleFetch}
                i18n={{
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
