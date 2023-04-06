import React, { FC, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import classNames from 'classnames'
import { useHistory } from 'react-router-dom'
import { Props } from './DevicesDetailsHeader.types'

import { showSuccessToast } from '@shared-ui/components/new/Toast'
import Button from '@shared-ui/components/new/Button'
import { WebSocketEventClient } from '@shared-ui/common/services'
import { useIsMounted } from '@shared-ui/common/hooks'
import { canChangeDeviceName, getResourceRegistrationNotificationKey, handleDeleteDevicesErrors, sleep } from '../../utils'
import { deleteDevicesApi } from '../../rest'
import { messages as t } from '../../Devices.i18n'
import Icon from '@shared-ui/components/new/Icon'
import { DeleteModal } from '@shared-ui/components/new/Modal'

const DevicesDetailsHeader: FC<Props> = (props) => {
    const { deviceId, deviceName, isUnregistered, isOnline, handleOpenEditDeviceNameModal, links } = props
    const { formatMessage: _ } = useIntl()
    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const [deleteModalOpen, setDeleteModalOpen] = useState(false)
    const [deleting, setDeleting] = useState(false)
    const isMounted = useIsMounted()
    const history = useHistory()
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
        console.log('handleOpenDeleteDeviceModal')
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
            if (isMounted.current) {
                setDeleting(false)
            }
            handleDeleteDevicesErrors(error, _, true)
        }
    }

    return (
        <div className={classNames('d-flex align-items-center', greyedOutClassName)}>
            <Button disabled={isUnregistered} icon={<Icon icon='trash' />} onClick={handleOpenDeleteDeviceModal} variant='tertiary'>
                {_(t.delete)}
            </Button>

            {canUpdate && (
                <Button
                    disabled={isUnregistered}
                    icon={<Icon icon='edit' />}
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
                subTitle={_(t.deleteResourceMessageSubtitle)}
                title={_(t.deleteResourceMessage)}
            />
        </div>
    )
}

DevicesDetailsHeader.displayName = 'DevicesDetailsHeader'

export default DevicesDetailsHeader
