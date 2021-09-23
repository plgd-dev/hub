import React, { PureComponent } from 'react'
import { injectIntl } from 'react-intl'

import { Button } from '@/components/button'
import { Modal } from '@/components/modal'
import { TextField } from '@/components/text-field'
import { Label } from '@/components/label'
import { showErrorToast } from '@/components/toast'
import { AppContext } from '@/containers/app/app-context'
import { CopyBox } from '@/components/copy-box'
import { isValidGuid } from '@/common/utils'

import { getDeviceAuthCode } from './rest'
import { messages as t } from './things-i18n'

const NOOP = () => {}

class _ProvisionNewDevice extends PureComponent {
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

    const providerName = this.context?.deviceOAuthClient?.providerName

    return (
      <>
        <Label title={_(t.authorizationProvider)} inline>
          <div id="auth-code-box">
            <span>{providerName || '-'}</span>
            {providerName && <CopyBox textToCopy={providerName} />}
          </div>
        </Label>

        <Label title={_(t.authorizationCode)} inline className="m-b-10">
          <div id="auth-code-box">
            <span>{code}</span>
            <CopyBox textToCopy={code} />
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

export const ProvisionNewDevice = injectIntl(_ProvisionNewDevice)
