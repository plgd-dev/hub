import React, { FC, memo, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'
import { useNavigate } from 'react-router-dom'

import Button from '@shared-ui/components/Atomic/Button'
import { WebSocketEventClient } from '@shared-ui/common/services'
import { useIsMounted } from '@shared-ui/common/hooks'
import { IconEdit, IconTrash } from '@shared-ui/components/Atomic/Icon'
import { DeleteModal } from '@shared-ui/components/Atomic/Modal'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'

import { Props } from './DevicesDetailsHeader.types'
import { canChangeDeviceName, getResourceRegistrationNotificationKey, handleDeleteDevicesErrors, sleep } from '../../utils'
import { deleteDevicesApi } from '../../rest'
import { messages as t } from '../../Devices.i18n'
import notificationId from '@/notificationId'
import testId from '@/testId'

const DevicesDetailsHeader: FC<Props> = memo((props) => {
    const { deviceId, deviceName, isUnregistered, isOnline, handleOpenEditDeviceNameModal, links } = props
    const { formatMessage: _ } = useIntl()
    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [deleting, setDeleting] = useState(false)
    const isMounted = useIsMounted()
    const navigate = useNavigate()
    const canUpdate = useMemo(() => canChangeDeviceName(links) && isOnline, [links, isOnline])

    const greyedOutClassName = classNames({
        'grayed-out': isUnregistered,
    })

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
            setDeleting(true)
            await deleteDevicesApi([deviceId])
            await sleep(200)

            if (isMounted.current) {
                Notification.success(
                    { title: t.deviceDeleted, message: _(t.deviceWasDeleted, { name: deviceName }) },
                    { notificationId: notificationId.HUB_DEVICES_DETAILS_HEADER_HANDLE_DELETE_DEVICE }
                )

                // Unregister the WS when the device is deleted
                WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)

                // Redirect to the device page after a deletion
                navigate(`/device`)
            }
        } catch (error) {
            if (isMounted.current) {
                setDeleting(false)
            }
            handleDeleteDevicesErrors(error, _, true)
        }
    }

    return (
        <div className={classNames('d-flex align-items-center', greyedOutClassName)}>
            <Button disabled={isUnregistered} icon={<IconTrash />} onClick={handleOpenDeleteDeviceModal} variant='tertiary'>
                {_(t.delete)}
            </Button>

            {canUpdate && (
                <Button
                    dataTestId={testId.devices.detail.editNameButton}
                    disabled={isUnregistered}
                    icon={<IconEdit />}
                    onClick={handleOpenEditDeviceNameModal}
                    style={{ marginLeft: 8 }}
                    variant='tertiary'
                >
                    {_(t.editName)}
                </Button>
            )}

            <DeleteModal
                deleteInformation={[
                    { label: _(t.deviceName), value: deviceName },
                    { label: _(t.deviceId), value: deviceId },
                ]}
                footerActions={[
                    {
                        label: _(t.cancel),
                        onClick: handleCloseDeleteDeviceModal,
                        variant: 'tertiary',
                    },
                    {
                        label: _(t.delete),
                        loading: deleting,
                        loadingText: _(t.deleting),
                        onClick: handleDeleteDevice,
                        variant: 'primary',
                    },
                ]}
                onClose={handleCloseDeleteDeviceModal}
                show={deleteModalOpen}
                subTitle={_(t.deleteDeviceMessageSubTitle)}
                title={_(t.deleteDeviceMessage)}
            />
        </div>
    )
})

DevicesDetailsHeader.displayName = 'DevicesDetailsHeader'

export default DevicesDetailsHeader
