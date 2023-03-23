import React, { FC, useEffect, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import classNames from 'classnames'
import { useHistory } from 'react-router-dom'
import { Props } from './DevicesDetailsHeader.types'

import { showSuccessToast } from '@shared-ui/components/new/Toast'
import Button from '@shared-ui/components/new/Button'
import { WebSocketEventClient, eventFilters } from '@shared-ui/common/services'
import Switch from '@shared-ui/components/new/Switch'
import { useIsMounted } from '@shared-ui/common/hooks'
import { canChangeDeviceName, getDeviceNotificationKey, getResourceRegistrationNotificationKey, handleDeleteDevicesErrors, sleep } from '../../utils'
import { isNotificationActive, toggleActiveNotification } from '../../slice'
import { deviceResourceRegistrationListener } from '../../websockets'
import { deleteDevicesApi } from '../../rest'
import { messages as t } from '../../Devices.i18n'
import Icon from '@shared-ui/components/new/Icon'
import { DeleteModal } from '@shared-ui/components/new/Modal'

const DevicesDetailsHeader: FC<Props> = (props) => {
    const { deviceId, deviceName, isUnregistered, isOnline, handleOpenEditDeviceNameModal, links } = props
    const { formatMessage: _ } = useIntl()
    const dispatch = useDispatch()
    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const deviceNotificationKey = getDeviceNotificationKey(deviceId)
    const notificationsEnabled = useRef(false)
    notificationsEnabled.current = useSelector(isNotificationActive(deviceNotificationKey))
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    // const [deleting, setDeleting] = useState(false)
    const isMounted = useIsMounted()
    const history = useHistory()
    const canUpdate = useMemo(() => canChangeDeviceName(links) && isOnline, [links, isOnline])

    const greyedOutClassName = classNames({
        'grayed-out': isUnregistered,
    })

    useEffect(() => {
        if (deviceId) {
            // Register the WS if not already registered
            WebSocketEventClient.subscribe(
                {
                    eventFilter: [eventFilters.RESOURCE_PUBLISHED, eventFilters.RESOURCE_UNPUBLISHED],
                    deviceIdFilter: [deviceId],
                },
                resourceRegistrationObservationWSKey,
                deviceResourceRegistrationListener({
                    deviceId,
                    deviceName,
                })
            )
        }

        return () => {
            // Unregister the WS if notification is off
            if (!notificationsEnabled.current) {
                WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
            }
        }
    }, [deviceId, deviceName, resourceRegistrationObservationWSKey])

    useEffect(() => {
        if (isUnregistered) {
            // Unregister the WS when the device is unregistered
            WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
        }
    }, [isUnregistered, resourceRegistrationObservationWSKey])

    const handleOpenDeleteDeviceModal = () => {
        if (isMounted.current) {
            setDeleteModalOpen(true)
        }
    }

    const handleCloseDeleteDeviceModal = () => {
        if (isMounted.current) {
            setDeleteModalOpen(false)
        }
    }

    const handleDeleteDevice = async () => {
        // api
        try {
            // setDeleting(true)
            await deleteDevicesApi([deviceId])
            await sleep(200)

            if (isMounted.current) {
                showSuccessToast({
                    title: _(t.deviceDeleted),
                    message: _(t.deviceWasDeleted, { name: deviceName }),
                })

                // Unregister the WS when the device is deleted
                WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)

                // Redirect to the device page after a deletion
                history.push(`/device`)
            }
        } catch (error) {
            // if (isMounted.current) {
            //     setDeleting(false)
            // }
            handleDeleteDevicesErrors(error, _, true)
        }
    }

    return (
        <div className={classNames('d-flex align-items-center', greyedOutClassName)}>
            {canUpdate && (
                <Button className='m-r-30' disabled={isUnregistered} icon={<Icon icon='edit' />} onClick={handleOpenEditDeviceNameModal} variant='tertiary'>
                    {_(t.editName)}
                </Button>
            )}

            <Button className='m-r-30' disabled={isUnregistered} icon={<Icon icon='trash' />} onClick={handleOpenDeleteDeviceModal} variant='secondary'>
                {_(t.delete)}
            </Button>

            <Switch
                checked={notificationsEnabled.current}
                className={classNames({ shimmering: !deviceId })}
                disabled={isUnregistered}
                id='status-notifications'
                label={_(t.notifications)}
                onChange={(e) => {
                    if (e.target.checked) {
                        // Request browser notifications
                        // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
                        // so we must call it to make sure the user has received a notification request)
                        Notification?.requestPermission?.()
                    }

                    dispatch(toggleActiveNotification(deviceNotificationKey))
                }}
            />

            <DeleteModal
                deleteInformation={[
                    { label: 'Device Name', value: deviceName },
                    { label: 'Device ID', value: deviceId },
                ]}
                footerActions={[
                    {
                        label: 'Cancel',
                        onClick: handleCloseDeleteDeviceModal,
                        variant: 'tertiary',
                    },
                    {
                        label: 'Delete',
                        onClick: handleDeleteDevice,
                        variant: 'primary',
                    },
                ]}
                onClose={handleCloseDeleteDeviceModal}
                show={deleteModalOpen}
                subTitle={_(t.deleteResourceMessageSubtitle)}
                title={_(t.deleteResourceMessage)}
            />
        </div>
    )
}

DevicesDetailsHeader.displayName = 'DevicesDetailsHeader'

export default DevicesDetailsHeader
