import { FC, useState } from 'react'
import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'
import { useEmitter } from '@shared-ui/common/hooks'
import Button from '@shared-ui/components/new/Button'
import Switch from '@shared-ui/components/new/Switch'
import ProvisionNewDevice from '../ProvisionNewDevice'
import { DEVICES_STATUS_WS_KEY, DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, RESET_COUNTER } from '../../constants'
import { isNotificationActive, toggleActiveNotification } from '../../slice'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesListHeader.types'
import * as styles from './DevicesListHeader.styles'

const DevicesListHeader: FC<Props> = ({ loading, refresh }) => {
    const { formatMessage: _ } = useIntl()
    const dispatch = useDispatch()
    const enabled = useSelector(isNotificationActive(DEVICES_STATUS_WS_KEY))
    const [numberOfChanges, setNumberOfChanges] = useState(0)

    useEmitter(DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, (numberOfNewChanges: number | string) => {
        typeof numberOfNewChanges === 'number' && setNumberOfChanges(numberOfChanges + numberOfNewChanges)
        numberOfNewChanges === RESET_COUNTER && setNumberOfChanges(0)
    })
    const refreshDevices = () => {
        // Re-fetch the devices list
        refresh()
    }

    return (
        <div css={styles.devicesListHeader}>
            <Button css={styles.item} disabled={loading || numberOfChanges === 0} loading={loading} onClick={refreshDevices}>
                {numberOfChanges > 0 && !loading && <span css={styles.circleNumber}>{numberOfChanges}</span>}
                {_(t.refresh)}
            </Button>
            <div css={styles.item}>
                <ProvisionNewDevice />
            </div>
            <Switch
                checked={enabled}
                css={styles.item}
                id='status-notifications'
                label={_(t.notifications)}
                onChange={(e) => {
                    if (e.target.checked) {
                        // Request browser notifications
                        // (browsers will explicitly disallow notification permission requests not triggered in response to a user gesture,
                        // so we must call it to make sure the user has received a notification request)
                        Notification?.requestPermission?.()
                    }

                    dispatch(toggleActiveNotification(DEVICES_STATUS_WS_KEY))
                }}
            />
        </div>
    )
}

DevicesListHeader.displayName = 'DevicesListHeader'

export default DevicesListHeader
