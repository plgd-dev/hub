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
                    { icon: 'icon-show-password', onClick: () => onView(deviceId), id: `detail-row-${deviceId}`, tooltipText: _(t.details) },
                ]}
            />
        )
    }

    return (
        <ActionButton
            items={[
                {
                    onClick: () => onView(deviceId),
                    label: _(t.details),
                    icon: <Icon icon='icon-show-password' />,
                },
                {
                    onClick: () => onDelete(deviceId),
                    label: _(t.delete),
                    icon: <Icon icon='trash' />,
                },
            ]}
            menuProps={{
                align: 'end',
            }}
            type={undefined}
        >
            <i className='fas fa-ellipsis-h' />
        </ActionButton>
    )
}

DevicesListActionButton.displayName = 'DevicesListActionButton'

export default DevicesListActionButton
