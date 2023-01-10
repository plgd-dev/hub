import React, { PureComponent } from 'react'
import { injectIntl } from 'react-intl'

import { Button } from '@/components/button'
import { Modal } from '@/components/modal'
import { TextField } from '@/components/text-field'
import { Label } from '@/components/label'
import { showErrorToast } from '@/components/toast'
import { AppContext } from '@/containers/App/AppContext'
import { CopyBox } from '@/components/copy-box'
import { isValidGuid } from '@/common/utils'

import { getDeviceAuthCode } from '../../rest'
import { messages as t } from '../../devices-i18n'

const NOOP = () => {}

class ProvisionNewDeviceCore extends PureComponent {
  static contextType = AppContext

  constructor(props) {
    super(props)

    this.state = {
      show: false,
      fetching: false,
      code: null,
      deviceId: '',
    }
  }

  componentDidMount() {
    this.isComponentMounted = true
  }

  componentWillUnmount() {
    this.isComponentMounted = false
  }

  handleFetch = async () => {
    const { deviceId } = this.state
    const {
      intl: { formatMessage: _ },
    } = this.props

    this.setState({ fetching: true })

    try {
      const code = await getDeviceAuthCode(deviceId)

      if (this.isComponentMounted) {
        this.setState({ fetching: false, code })
      }
    } catch (e) {
      showErrorToast({
        title: _(t.deviceAuthCodeError),
        message: e.message,
      })

      if (this.isComponentMounted) {
        this.setState({ fetching: false })
      }
    }
  }

  handleOnValueChange = event =>
    this.setState({ deviceId: event.target.value.trim() })

  handleRestart = () => {
    this.setState({ code: null, deviceId: '' }, () => {
      this?.input?.focus?.()
    })
  }

  renderFooter = () => {
    const { fetching, deviceId, code } = this.state
    const {
      intl: { formatMessage: _ },
    } = this.props

    return (
      <div className="w-100 d-flex justify-content-end align-items-center">
        {code && (
          <Button variant="secondary" onClick={this.handleRestart}>
            {_(t.back)}
          </Button>
        )}

        <Button
          variant={!code ? 'secondary' : 'primary'}
          onClick={this.onClose}
          disabled={fetching}
        >
          {code ? _(t.close) : _(t.cancel)}
        </Button>

        {!code && (
          <Button
            variant="primary"
            onClick={this.handleFetch}
            loading={fetching}
            disabled={fetching || !isValidGuid(deviceId.trim())}
          >
            {_(t.getCode)}
          </Button>
        )}
      </div>
    )
  }

  renderBody = () => {
    const { code, deviceId, fetching } = this.state
    const {
      intl: { formatMessage: _ },
    } = this.props

    if (!code) {
      return (
        <Label title={_(t.deviceId)}>
          <TextField
            value={deviceId}
            onChange={this.handleOnValueChange}
            placeholder={_(t.enterDeviceId)}
            disabled={fetching}
            inputRef={ref => {
              this.input = ref
            }}
          />
        </Label>
      )
    }

    const {
      coapGateway: deviceEndpoint,
      id: hubId,
      certificateAuthorities,
    } = this.context?.wellKnownConfig || {}
    const providerName = this.context?.deviceOauthClient?.providerName

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

  onOpen = () => {
    this.setState({ show: true }, () => {
      this?.input?.focus?.()
    })
  }

  onClose = () => this.setState({ show: false, code: null, deviceId: '' })

  render() {
    const { fetching, show } = this.state
    const {
      intl: { formatMessage: _ },
    } = this.props
    const deviceOauthClient = this.context?.deviceOauthClient

    if (!deviceOauthClient?.providerName || !deviceOauthClient?.clientId) {
      return null
    }

    return (
      <>
        <Button onClick={this.onOpen} className="m-r-30" icon="fa-plus">
          {_(t.device)}
        </Button>

        <Modal
          show={show}
          onClose={!fetching ? this.onClose : NOOP}
          title={_(t.provisionNewDevice)}
          renderBody={this.renderBody}
          renderFooter={this.renderFooter}
          closeButton={!fetching}
        />
      </>
    )
  }
}

export const ProvisionNewDevice = injectIntl(ProvisionNewDeviceCore)
