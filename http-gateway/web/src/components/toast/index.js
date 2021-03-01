import PropTypes from 'prop-types'
import { ToastContainer as Toastr, toast } from 'react-toastify'
import classNames from 'classnames'
import { useIntl } from 'react-intl'

import { isBrowserTabActive } from '@/common/utils'
import { toastTypes } from './constants'
import { translateToastString } from './utils'

const { ERROR, SUCCESS, WARNING, INFO } = toastTypes

export const ToastContainer = () => {
  return (
    <Toastr
      closeButton={({ closeToast }) => (
        <i onClick={closeToast} className="fas fa-times close-toast" />
      )}
      pauseOnFocusLoss={false}
      limit={5}
      newestOnTop
      autoClose={8000}
    />
  )
}

const ToastComponent = props => {
  const { formatMessage: _ } = useIntl()
  const { message, title, type } = props

  const toastMessage = translateToastString(message, _)
  const toastTitle = translateToastString(title, _)

  return (
    <div className="toast-component">
      <div className="toast-icon">
        <i
          className={classNames('fas', {
            'fa-info-circle': type === INFO,
            'fa-check-circle': type === SUCCESS,
            'fa-exclamation-circle': type === WARNING,
            'fa-times-circle': type === ERROR,
          })}
        />
      </div>
      <div className="toast-content">
        {toastTitle && <div className="title">{toastTitle}</div>}
        <div className="message">{toastMessage}</div>
      </div>
    </div>
  )
}

ToastComponent.propTypes = {
  message: PropTypes.oneOfType([PropTypes.node, PropTypes.object]).isRequired,
  title: PropTypes.oneOfType([PropTypes.node, PropTypes.object]),
  type: PropTypes.oneOf([ERROR, SUCCESS, WARNING, INFO]),
}

ToastComponent.defaultProps = {
  title: null,
  type: ERROR,
}

/**
 *
 * @param {*} message Can be a simple string/component, or an object of { message, title }
 * @param {*} options All available props from https://fkhadra.github.io/react-toastify/api/toast
 * @param {*} type [success, error, warning, info]
 */
export const showToast = (message, options = {}, type = ERROR) => {
  if (isBrowserTabActive()) {
    const toastMessage = message?.message || message
    const toastTitle = message?.title || null

    const renderToast = (
      <ToastComponent message={toastMessage} title={toastTitle} type={type} />
    )

    switch (type) {
      case SUCCESS:
        toast.success(renderToast, options)
        break
      case WARNING:
        toast.warning(renderToast, options)
        break
      case INFO:
        toast.info(renderToast, options)
        break
      default:
        toast.error(renderToast, options)
    }
  }
}

export const showErrorToast = (message, options = {}) =>
  showToast(message, options, ERROR)
export const showSuccessToast = (message, options = {}) =>
  showToast(message, options, SUCCESS)
export const showInfoToast = (message, options = {}) =>
  showToast(message, options, INFO)
export const showWarningToast = (message, options = {}) =>
  showToast(message, options, WARNING)
