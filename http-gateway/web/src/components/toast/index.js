import { useEffect } from 'react'
import PropTypes from 'prop-types'
import { ToastContainer as Toastr, toast } from 'react-toastify'
import classNames from 'classnames'
import { useIntl } from 'react-intl'

import { Emitter } from '@/common/services/emitter'
import {
  isBrowserTabActive,
  playFartSound,
  loadFartSound,
} from '@/common/utils'
import {
  toastTypes,
  browserNotificationPermissions,
  TOAST_HIDE_TIME,
  MAX_NUMBER_OF_VISIBLE_TOASTS,
  MAX_NUMBER_OF_ALL_TOASTS,
  MAX_NUMBER_OF_BROWSER_NOTIFICATIONS,
  BROWSER_NOTIFICATIONS_EVENT_KEY,
  BROWSER_NOTIFICATION_HIDE_TIME,
} from './constants'
import { translateToastString } from './utils'

const { ERROR, SUCCESS, WARNING, INFO } = toastTypes

// Globals
let dispatchedToasts = 0
let dispatchedBrowserNotifications = 0
let notification = null
let decrementTimer = null

// Container responsible for processing and dispatching the toast notifications
export const ToastContainer = () => {
  return (
    <Toastr
      closeButton={({ closeToast }) => (
        <i onClick={closeToast} className="fas fa-times close-toast" />
      )}
      pauseOnFocusLoss={false}
      limit={MAX_NUMBER_OF_VISIBLE_TOASTS}
      newestOnTop
      autoClose={TOAST_HIDE_TIME}
      hideProgressBar
    />
  )
}

// Single toast component
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

// Container responsible for processing and dispatching the browser notifications
export const BrowserNotificationsContainer = () => {
  const { formatMessage: _, locale } = useIntl()

  const decrementCounter = () => {
    if (dispatchedBrowserNotifications > 0) {
      dispatchedBrowserNotifications--
    }
  }

  const showBrowserNotification = (params, options) => {
    const { message, title } = params

    const toastMessage = translateToastString(message, _)
    const toastTitle = translateToastString(title, _)

    if (
      dispatchedBrowserNotifications < MAX_NUMBER_OF_BROWSER_NOTIFICATIONS &&
      Notification?.permission === browserNotificationPermissions.GRANTED
    ) {
      dispatchedBrowserNotifications++

      // Close the previous notification when showing a new one
      if (notification?.close) {
        decrementCounter()
        clearTimeout(decrementTimer)
        notification.close()
      }

      notification = new Notification(toastTitle, {
        body: toastMessage,
        icon: '/favicon.png',
        badge: '/favicon.png',
        lang: locale,
        silent: true,
      })

      // After approximately 5 seconds the notification disappears, so lets decrement the counter.
      decrementTimer = setTimeout(() => {
        decrementCounter()
      }, BROWSER_NOTIFICATION_HIDE_TIME)

      notification.onclick = () => {
        window.focus()
        decrementCounter()
        clearTimeout(decrementTimer)

        if (options?.onClick) {
          options.onClick()
        }
      }

      // Play fart sound :)
      playFartSound()
    }
  }

  const loadSounds = () => {
    loadFartSound()
    document.removeEventListener('click', loadSounds)
  }

  useEffect(() => {
    Emitter.on(BROWSER_NOTIFICATIONS_EVENT_KEY, showBrowserNotification)
    document.addEventListener('click', loadSounds)

    return () => {
      Emitter.off(BROWSER_NOTIFICATIONS_EVENT_KEY, showBrowserNotification)
      document.removeEventListener('click', loadSounds)
    }
  }, []) // eslint-disable-line

  return null
}

/**
 *
 * @param {*} message Can be a simple string/component, or an object of { message, title }
 * @param {*} options All available props from https://fkhadra.github.io/react-toastify/api/toast
 * @param {*} type [success, error, warning, info]
 */
export const showToast = (message, options = {}, type = ERROR) => {
  const toastMessage = message?.message || message
  const toastTitle = message?.title || null

  if (isBrowserTabActive() && dispatchedToasts < MAX_NUMBER_OF_ALL_TOASTS) {
    dispatchedToasts++

    const renderToast = (
      <ToastComponent message={toastMessage} title={toastTitle} type={type} />
    )

    const onToastClose = props => {
      if (dispatchedToasts > 0) {
        dispatchedToasts--
      }

      if (options?.onClose) {
        options.onClose(props)
      }
    }

    const toastOptions = { ...options, onClose: onToastClose }

    switch (type) {
      case SUCCESS:
        toast.success(renderToast, toastOptions)
        break
      case WARNING:
        toast.warning(renderToast, toastOptions)
        break
      case INFO:
        toast.info(renderToast, toastOptions)
        break
      default:
        toast.error(renderToast, toastOptions)
    }
  } else if (options?.isNotification) {
    // If it is a notification, try to push it to the browser if borwser notifications are enabled
    // - Emit a an event to be processed in the BrowserNotificationsContainer
    Emitter.emit(
      BROWSER_NOTIFICATIONS_EVENT_KEY,
      {
        message: toastMessage,
        title: toastTitle,
      },
      options
    )
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
