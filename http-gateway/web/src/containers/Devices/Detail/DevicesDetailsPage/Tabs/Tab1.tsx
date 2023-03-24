import React, { FC, useEffect, useRef, useState } from 'react'
import TileToggleRow from '@shared-ui/components/new/TileToggle/TileToggleRow'
import TileToggle from '@shared-ui/components/new/TileToggle'
import { Props } from './Tab1.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { useIntl } from 'react-intl'
import SimpleStripTable from '@shared-ui/components/new/SimpleStripTable'
import TagGroup from '@shared-ui/components/new/TagGroup'
import Tag from '@shared-ui/components/new/Tag'
import { Icon } from '@shared-ui/components/new/Icon'
import { getDeviceNotificationKey, getResourceRegistrationNotificationKey } from '@/containers/Devices/utils'
import { useDispatch, useSelector } from 'react-redux'
import { isNotificationActive, toggleActiveNotification } from '@/containers/Devices/slice'
import { eventFilters, WebSocketEventClient } from '@shared-ui/common/services'
import { deviceResourceRegistrationListener } from '@/containers/Devices/websockets'

const Tab1: FC<Props> = (props) => {
    const { isTwinEnabled, setTwinSynchronization, twinSyncLoading, deviceId, types, deviceName } = props
    const { formatMessage: _ } = useIntl()

    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const deviceNotificationKey = getDeviceNotificationKey(deviceId)
    const notificationsEnabled = useRef(false)
    notificationsEnabled.current = useSelector(isNotificationActive(deviceNotificationKey))
    const dispatch = useDispatch()

    const [state, setState] = useState({
        tile2: false,
        tile3: true,
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
                    name={_(t.subscribeNotify)}
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
                <TileToggle checked={state.tile3} name={_(t.logging)} onChange={() => setState({ ...state, tile3: !state.tile3 })} />
            </TileToggleRow>
            <div style={{ paddingTop: 16 }}>
                <SimpleStripTable
                    rows={[
                        { attribute: _(t.id), value: deviceId },
                        { attribute: _(t.model), value: 'TODO: doorbell-2020-11-03' },
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
                                    <span style={{ marginRight: 6 }}>0.22.1</span> <Icon icon='cloud-success' size={24} />
                                </div>
                            ),
                        },
                        { attribute: _(t.status), value: 'TODO: 3 pending commands' },
                    ]}
                />
            </div>
        </div>
    )
}

export default Tab1
