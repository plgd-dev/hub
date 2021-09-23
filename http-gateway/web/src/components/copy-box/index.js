import React, { PureComponent } from 'react'
import PropTypes from 'prop-types'
import { injectIntl } from 'react-intl'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'
import { v4 as uuidv4 } from 'uuid'

import { copyToClipboard } from '@/common/utils'
import { messages as t } from './copy-box-i18n'

class _CopyBox extends PureComponent {
  static propTypes = {
    text: PropTypes.string,
    textToCopy: PropTypes.string,
    copyToClipboardText: PropTypes.string,
  }

  static defaultProps = {
    copyToClipboardText: null,
    textToCopy: null,
    text: null,
  }

  copyTimer = null

  constructor(props) {
    super(props)

    this.state = {
      copiedToClipboard: false,
    }
  }

  componentWillUnmount() {
    clearTimeout(this.copyTimer)
  }

  handleCopyToClipboard = () => {
    const { text, textToCopy } = this.props

    if (copyToClipboard(textToCopy || text)) {
      this.setState({ copiedToClipboard: true })

      this.copyTimer = setTimeout(() => {
        this.setState({ copiedToClipboard: false })
      }, 3000)
    }
  }

  renderCopyToClipboardHintContent = () => {
    const { copiedToClipboard } = this.state
    const {
      copyToClipboardText,
      intl: { formatMessage: _ },
    } = this.props

    return copiedToClipboard ? (
      <div className="copy-success">
        <i className="fas fa-check-circle m-r-5" />
        {_(t.copied)}
      </div>
    ) : (
      copyToClipboardText || _(t.copyToClipboard)
    )
  }

  render() {
    const { text } = this.props

    return (
      <div className="copy-box">
        {text}
        <OverlayTrigger
          placement="right"
          overlay={
            <Tooltip
              id={`menu-item-tooltip-${uuidv4()}`}
              className="plgd-tooltip"
            >
              {this.renderCopyToClipboardHintContent()}
            </Tooltip>
          }
        >
          <div className="box m-l-10" onClick={this.handleCopyToClipboard}>
            <i className="far fa-copy" />
          </div>
        </OverlayTrigger>
      </div>
    )
  }
}

export const CopyBox = injectIntl(_CopyBox)
