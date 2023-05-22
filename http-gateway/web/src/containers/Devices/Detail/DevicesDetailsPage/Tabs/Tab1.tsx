import React, { FC, useEffect, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'

import TileToggleRow from '@shared-ui/components/Atomic/TileToggle/TileToggleRow'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import { convertSize, IconCloudSuccess } from '@shared-ui/components/Atomic/Icon'
import { eventFilters, WebSocketEventClient } from '@shared-ui/common/services'

import { Props } from './Tab1.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { getDeviceNotificationKey, getResourceRegistrationNotificationKey } from '@/containers/Devices/utils'
import { isNotificationActive, toggleActiveNotification } from '@/containers/Devices/slice'
import { deviceResourceRegistrationListener } from '@/containers/Devices/websockets'

const Tab1: FC<Props> = (props) => {
    const { isTwinEnabled, setTwinSynchronization, twinSyncLoading, deviceId, types, deviceName, model, pendingCommandsData } = props
    const { formatMessage: _ } = useIntl()

    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const deviceNotificationKey = getDeviceNotificationKey(deviceId)
    const notificationsEnabled = useRef(false)
    notificationsEnabled.current = useSelector(isNotificationActive(deviceNotificationKey))
    const dispatch = useDispatch()

    const [state, setState] = useState({
        tile3: true,
    })

    useEffect(() => {
        if (deviceId && notificationsEnabled.current) {
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
    }, [deviceId, deviceName, resourceRegistrationObservationWSKey, notificationsEnabled])

    return (
        <div
            style={{
                paddingTop: 8,
                overflow: 'hidden',
            }}
        >
            <TileToggleRow>
                <TileToggle checked={isTwinEnabled} loading={twinSyncLoading} name={_(t.twinState)} onChange={() => setTwinSynchronization(!isTwinEnabled)} />
                <TileToggle
                    checked={notificationsEnabled.current}
                    name={_(t.notifications)}
                    onChange={(e) => {
                        if (e.target.checked) {
                            // Request browser notifications
                            // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
                            // so we must call it to make sure the user has received a notification request)
                            Notification?.requestPermission?.().then()

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
                        } else {
                            WebSocketEventClient.unsubscribe(resourceRegistrationObservationWSKey)
                        }

                        dispatch(toggleActiveNotification(deviceNotificationKey))
                    }}
                />
                <TileToggle checked={state.tile3} name={_(t.logging)} onChange={() => setState({ ...state, tile3: !state.tile3 })} />
            </TileToggleRow>
            <div style={{ paddingTop: 16 }}>
                <SimpleStripTable
                    rows={[
                        { attribute: _(t.id), value: deviceId },
                        { attribute: _(t.model), value: model || '-' },
                        {
                            attribute: _(t.types),
                            value: types ? (
                                <TagGroup>
                                    {types.map((t, key) => (
                                        <Tag key={t}>{t}</Tag>
                                    ))}
                                </TagGroup>
                            ) : (
                                <div>-</div>
                            ),
                        },
                        {
                            attribute: _(t.firmware),
                            value: (
                                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                    <span style={{ marginRight: 6 }}>0.22.1</span> <IconCloudSuccess {...convertSize(24)} />
                                </div>
                            ),
                        },
                        { attribute: _(t.status), value: pendingCommandsData ? `${pendingCommandsData.length} pending commands` : '-' },
                    ]}
                />
            </div>
        </div>
    )
}

export default Tab1
