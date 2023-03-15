import { FC } from 'react'
import { useIntl } from 'react-intl'
import ActionButton from '@shared-ui/components/new/ActionButton'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesListActionButton.types'
import { useMediaQuery } from 'react-responsive'
import TableActions from '@plgd/shared-ui/src/components/new/TableNew/TableActions'
import Icon from '@shared-ui/components/new/Icon'

const DevicesListActionButton: FC<Props> = (props) => {
    const { deviceId, onView, onDelete } = props
    const { formatMessage: _ } = useIntl()
    const isDesktopOrLaptop = useMediaQuery({
        query: '(min-width: 1281px)',
    })

    if (isDesktopOrLaptop) {
        return (
            <TableActions
                items={[
                    { icon: 'trash', onClick: () => onDelete(deviceId), id: `delete-row-${deviceId}`, tooltipText: _(t.delete) },
                    { icon: 'arrow-right', onClick: () => onView(deviceId), id: `detail-row-${deviceId}`, tooltipText: _(t.details), size: 14 },
                ]}
            />
        )
    }

    return (
        <ActionButton
            items={[
                {
                    onClick: () => onDelete(deviceId),
                    label: _(t.delete),
                    icon: 'trash',
                },
                {
                    onClick: () => onView(deviceId),
                    label: _(t.view),
                    icon: 'icon-show-password',
                },
            ]}
            menuProps={{
                placement: 'bottom-end',
            }}
            type={undefined}
        />
    )
}

DevicesListActionButton.displayName = 'DevicesListActionButton'

export default DevicesListActionButton
