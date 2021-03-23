import { useState, useMemo } from 'react'
import PropTypes from 'prop-types'
import classNames from 'classnames'
import Form from 'react-bootstrap/Form'
import { useIntl } from 'react-intl'

import { Button } from '@/components/button'
import { showErrorToast } from '@/components/toast'
import { useIsMounted } from '@/common/hooks'
import { getApiErrorMessage } from '@/common/utils'
import { updateThingsResourceApi } from './rest'
import { canChangeDeviceName, getDeviceChangeResourceHref } from './utils'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsDetailsTitle = ({
  className,
  deviceName,
  deviceId,
  updateDeviceName,
  loading,
  isOnline,
  links,
  ...rest
}) => {
  const { formatMessage: _ } = useIntl()
  const [inputTitle, setInputTitle] = useState('')
  const [edit, setEdit] = useState(false)
  const [saving, setSaving] = useState(false)
  const isMounted = useIsMounted()
  const canUpdate = useMemo(() => canChangeDeviceName(links) && isOnline, [
    links,
    isOnline,
  ])

  const onEditClick = () => {
    setInputTitle(deviceName || '')
    setEdit(true)
  }

  const onCloseClick = () => {
    setEdit(false)
  }

  const cancel = () => {
    setSaving(false)
    setEdit(false)
  }

  const onSave = async () => {
    if (inputTitle.trim() !== '' && inputTitle !== deviceName && canUpdate) {
      const href = getDeviceChangeResourceHref(links)

      setSaving(true)

      try {
        const { data } = await updateThingsResourceApi(
          { deviceId, href },
          {
            n: inputTitle,
          }
        )

        if (isMounted.current) {
          cancel()
          updateDeviceName(data?.n || inputTitle)
        }
      } catch (error) {
        if (error && isMounted.current) {
          showErrorToast({
            title: _(t.thingNameChangeFailed),
            message: getApiErrorMessage(error),
          })
        }
      }
    } else {
      cancel()
    }
  }

  const handleKeyDown = e => {
    if (e.keyCode === 13) {
      // Enter
      onSave()
    } else if (e.keyCode === 27) {
      // Esc
      cancel()
    }
  }

  if (edit) {
    return (
      <div className="form-control-with-button h2-input">
        <Form.Control
          type="text"
          placeholder={`${_(t.enterThingName)}...`}
          value={inputTitle}
          onChange={e => setInputTitle(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={saving}
          autoFocus
        />
        <Button
          variant="primary"
          onClick={onSave}
          disabled={saving}
          loading={saving}
        >
          {saving ? _(t.saving) : _(t.save)}
        </Button>
        <Button
          className="close-button"
          variant="secondary"
          onClick={onCloseClick}
          disabled={saving}
        >
          <i className="fas fa-times" />
        </Button>
      </div>
    )
  }

  return (
    <h2
      {...rest}
      className={classNames(className, 'd-inline-flex align-items-center', {
        'title-with-icon': canUpdate,
      })}
      onClick={canUpdate ? onEditClick : null}
    >
      <span className={canUpdate ? 'link reveal-icon-on-hover icon-visible' : null}>
        {deviceName}
      </span>
      {canUpdate && <i className="fas fa-pen" />}
    </h2>
  )
}

ThingsDetailsTitle.propTypes = {
  className: PropTypes.string,
  deviceName: PropTypes.string,
  deviceId: PropTypes.string,
  loading: PropTypes.bool.isRequired,
  updateDeviceName: PropTypes.func.isRequired,
  isOnline: PropTypes.bool.isRequired,
  links: PropTypes.arrayOf(thingResourceShape),
}

ThingsDetailsTitle.defaultProps = {
  className: null,
  deviceName: null,
  deviceId: null,
  links: [],
}
