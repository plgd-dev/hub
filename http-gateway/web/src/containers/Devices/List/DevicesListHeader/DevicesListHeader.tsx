import { FC, useState } from 'react'
import { useIntl } from 'react-intl'

import { useEmitter } from '@shared-ui/common/hooks'
import Button from '@shared-ui/components/Atomic/Button'

import ProvisionNewDevice from '../ProvisionNewDevice'
import { DEVICES_REGISTERED_UNREGISTERED_COUNT_EVENT_KEY, RESET_COUNTER } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesListHeader.types'
import * as styles from './DevicesListHeader.styles'

const DevicesListHeader: FC<Props> = ({ loading, refresh }) => {
    const { formatMessage: _ } = useIntl()

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
        </div>
    )
}

DevicesListHeader.displayName = 'DevicesListHeader'

export default DevicesListHeader
