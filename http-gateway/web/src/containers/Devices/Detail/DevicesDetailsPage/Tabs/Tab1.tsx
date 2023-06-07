import React, { FC, useEffect, useRef } from 'react'
import { useIntl } from 'react-intl'
import { useDispatch, useSelector } from 'react-redux'

import TileToggleRow from '@shared-ui/components/Atomic/TileToggle/TileToggleRow'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import TagGroup from '@shared-ui/components/Atomic/TagGroup'
import Tag from '@shared-ui/components/Atomic/Tag'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import { convertSize, IconCloudSuccess, IconCloudWarning } from '@shared-ui/components/Atomic/Icon'
import { eventFilters, WebSocketEventClient } from '@shared-ui/common/services'

import { Props } from './Tab1.types'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { getDeviceNotificationKey, getResourceRegistrationNotificationKey } from '@/containers/Devices/utils'
import { isNotificationActive, toggleActiveNotification } from '@/containers/Devices/slice'
import { deviceResourceRegistrationListener } from '@/containers/Devices/websockets'

const Tab1: FC<Props> = (props) => {
    const { isTwinEnabled, setTwinSynchronization, twinSyncLoading, deviceId, types, deviceName, model, pendingCommandsData, firmware, softwareUpdateData } =
        props
    const { formatMessage: _ } = useIntl()

    const resourceRegistrationObservationWSKey = getResourceRegistrationNotificationKey(deviceId)
    const deviceNotificationKey = getDeviceNotificationKey(deviceId)
    const notificationsEnabled = useRef(false)
    notificationsEnabled.current = useSelector(isNotificationActive(deviceNotificationKey))
    const dispatch = useDispatch()

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
                {/* <TileToggle checked={state.tile3} name={_(t.logging)} onChange={() => setState({ ...state, tile3: !state.tile3 })} />*/}
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
                        /*
                        https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.r.softwareupdate.swagger.json
                        softwareUpdateData = {
                            "swupdateaction":"idle", https://github.com/plgd-dev/device/blob/2a60018de0639e7f225254ff9487bcf91bbb603f/schema/softwareupdate/swupdate.go#L49
                            "swupdateresult":0,  //
                            "swupdatestate":"nsa", https://github.com/plgd-dev/device/blob/2a60018de0639e7f225254ff9487bcf91bbb603f/schema/softwareupdate/swupdate.go#L58
                            "updatetime":"2023-06-02T07:37:00Z", // when the action will be performed
                            "lastupdate":"2023-06-02T07:37:02.330206Z",  // when the upgrade was performed
                            "nv":"0.0.12", // new available version
                            "purl":"https://hosted.mender.io?device_type=ocf&tenant_token=eyJhbuqBM",
                            "signed":"vendor"
                        }
                        // to upgrade send a POST request to swu resource with the following payload { swupdateaction: 'upgrade', updatetime: 'now+10sec', purl: 'same value as been received in the swupdate payload'}
                        */
                        {
                            attribute: _(t.firmware),
                            value: firmware ? (
                                <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                                    <span style={{ marginRight: 6 }}>{firmware}</span>{' '}
                                    {softwareUpdateData?.swupdatestate !== 'idle' ? (
                                        <Tooltip
                                            content={_(t.newDeviceFirmware, {
                                                newVersion: softwareUpdateData?.nv,
                                            })}
                                            delay={200}
                                        >
                                            <IconCloudWarning {...convertSize(24)} />
                                        </Tooltip>
                                    ) : (
                                        <Tooltip
                                            content={_(t.deviceFirmwareUpToDate, {
                                                lastUpdate: new Date(softwareUpdateData?.lastupdate!).toLocaleDateString('en-US'),
                                            })}
                                            delay={200}
                                        >
                                            <IconCloudSuccess {...convertSize(24)} />
                                        </Tooltip>
                                    )}
                                </div>
                            ) : (
                                <div>-</div>
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
