import { useState, useMemo, FC } from 'react'
import classNames from 'classnames'
import Form from 'react-bootstrap/Form'
import omit from 'lodash/omit'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import { useIsMounted } from '@shared-ui/common/hooks'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { Props } from './DevicesDetailsTitle.types'
import { updateDevicesResourceApi } from '../../rest'
import { canChangeDeviceName, getDeviceChangeResourceHref } from '../../utils'
import { messages as t } from '../../Devices.i18n'

const DevicesDetailsTitle: FC<Props> = ({ className, deviceName, deviceId, updateDeviceName, isOnline, links, ttl, ...rest }) => {
    const { formatMessage: _ } = useIntl()
    const [inputTitle, setInputTitle] = useState('')
    const [edit, setEdit] = useState(false)
    const [saving, setSaving] = useState(false)
    const isMounted = useIsMounted()
    const canUpdate = useMemo(() => canChangeDeviceName(links) && isOnline, [links, isOnline])

    const onEditClick = () => {
        setInputTitle(deviceName || '')
        setEdit(true)
    }

    const onCloseClick = () => {
        setEdit(false)
    }

    const cancelSave = () => {
        setSaving(false)
        setEdit(false)
    }

    const onSave = async () => {
        if (inputTitle.trim() !== '' && inputTitle !== deviceName && canUpdate) {
            const href = getDeviceChangeResourceHref(links)

            setSaving(true)

            try {
                const { data } = await updateDevicesResourceApi(
                    { deviceId, href: href!, ttl },
                    {
                        n: inputTitle,
                    }
                )

                if (isMounted.current) {
                    cancelSave()
                    updateDeviceName(data?.n || inputTitle)
                }
            } catch (error) {
                if (error && isMounted.current) {
                    Notification.error({ title: _(t.deviceNameChangeFailed), message: getApiErrorMessage(error) })
                    cancelSave()
                }
            }
        } else {
            cancelSave()
        }
    }

    const handleKeyDown = (e: any) => {
        if (e.keyCode === 13) {
            // Enter
            onSave()
        } else if (e.keyCode === 27) {
            // Esc
            cancelSave()
        }
    }

    if (edit) {
        return (
            <div className='form-control-with-button h2-input'>
                <Form.Control
                    autoFocus
                    disabled={saving}
                    onChange={(e) => setInputTitle(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={`${_(t.enterDeviceName)}...`}
                    type='text'
                    value={inputTitle}
                />
                <Button disabled={saving} loading={saving} onClick={onSave} variant='primary'>
                    {saving ? _(t.saving) : _(t.save)}
                </Button>
                <Button className='close-button' disabled={saving} onClick={onCloseClick} variant='secondary'>
                    <i className='fas fa-times' />
                </Button>
            </div>
        )
    }

    return (
        <h2
            {...omit(rest, 'loading')}
            className={classNames(className, 'd-inline-flex align-items-center', {
                'title-with-icon': canUpdate,
            })}
            onClick={canUpdate ? onEditClick : undefined}
        >
            <span className={canUpdate ? 'link reveal-icon-on-hover icon-visible' : undefined}>{deviceName}</span>
            {canUpdate && <i className='fas fa-pen' />}
        </h2>
    )
}

DevicesDetailsTitle.displayName = 'DevicesDetailsTitle'

export default DevicesDetailsTitle
