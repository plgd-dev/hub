import React, { useRef, useState } from 'react'
import { getDeviceAuthCode } from '@/containers/Devices/rest'
import { showErrorToast } from '@shared-ui/components/new/Toast/Toast'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { useIntl } from 'react-intl'
import Button from '@shared-ui/components/new/Button'
import Modal from '@shared-ui/components/new/Modal'
import { isValidGuid } from '@/common/utils'
import Label from '@shared-ui/components/new/Label'
import TextField from '@shared-ui/components/new/TextField'
import CopyBox from '@shared-ui/components/new/CopyBox'
import { security } from '@/common/services'

const ProvisionNewDeviceCore = () => {
  const [show, setShow] = useState(false)
  const [fetching, setFetching] = useState(false)
  const [deviceId, setDeviceId] = useState<null | string>(null)
  const [code, setCode] = useState<null | string>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { formatMessage: _ } = useIntl()

  const handleFetch = async () => {
    setFetching(true)
    try {
      const code = await getDeviceAuthCode(deviceId)
      setFetching(false)
      setCode(code)
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
    setCode(null)
    setDeviceId(null)
  }

  const handleRestart = () => {
    setCode(null)
    setDeviceId(null)
    inputRef?.current?.focus()
  }

  const renderFooter = () => {
    return (
      <div className="w-100 d-flex justify-content-end align-items-center">
        {code && (
          <Button variant="secondary" onClick={handleRestart}>
            {_(t.back)}
          </Button>
        )}
        <Button
          variant={!code ? 'secondary' : 'primary'}
          onClick={onClose}
          disabled={fetching}
        >
          {code ? _(t.close) : _(t.cancel)}
        </Button>
        {!code && (
          <Button
            variant="primary"
            onClick={handleFetch}
            loading={fetching}
            disabled={fetching || !isValidGuid(deviceId?.trim())}
          >
            {_(t.getCode)}
          </Button>
        )}
      </div>
    )
  }

  const renderBody = () => {
    if (!code) {
      return (
        <Label title={_(t.deviceId)}>
          <TextField
            value={deviceId || ''}
            onChange={e => setDeviceId(e.target.value.trim())}
            placeholder={_(t.enterDeviceId)}
            disabled={fetching}
            inputRef={inputRef}
          />
        </Label>
      )
    }

    const {
      coapGateway: deviceEndpoint,
      id: hubId,
      certificateAuthorities,
    } = security.getWellKnowConfig() || {}
    const { providerName } = (security.getDeviceOAuthConfig() as any) || {}

    return (
      <>
        <Label title={_(t.hubId)} inline>
          <div className="auth-code-box">
            <span>{hubId || '-'}</span>
            {hubId && <CopyBox textToCopy={hubId} />}
          </div>
        </Label>

        <Label title={_(t.deviceEndpoint)} inline>
          <div className="auth-code-box">
            <span className="text-overflow">{deviceEndpoint || '-'}</span>
            {deviceEndpoint && <CopyBox textToCopy={deviceEndpoint} />}
          </div>
        </Label>

        <Label title={_(t.authorizationCode)} inline>
          <div className="auth-code-box">
            <span className="text-overflow">********</span>
            <CopyBox textToCopy={code} />
          </div>
        </Label>

        <Label title={_(t.authorizationProvider)} inline>
          <div className="auth-code-box">
            <span className="text-overflow">{providerName || '-'}</span>
            {providerName && <CopyBox textToCopy={providerName} />}
          </div>
        </Label>

        <Label title={_(t.certificateAuthorities)} inline className="m-b-10">
          <div className="auth-code-box">
            <span>...</span>
            {certificateAuthorities && (
              <CopyBox textToCopy={certificateAuthorities} certFormat={true} />
            )}
          </div>
        </Label>
      </>
    )
  }

  return (
    <>
      <Button onClick={openModal} className="m-r-30" icon="fa-plus">
        {_(t.device)}
      </Button>
      <Modal
        show={show}
        onClose={!fetching ? onClose : () => {}}
        title={_(t.provisionNewDevice)}
        renderBody={renderBody}
        renderFooter={renderFooter}
        closeButton={!fetching}
      />
    </>
  )
}

export default ProvisionNewDeviceCore
