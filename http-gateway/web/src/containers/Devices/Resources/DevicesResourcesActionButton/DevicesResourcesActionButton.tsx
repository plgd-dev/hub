import { FC } from 'react'
import { useIntl } from 'react-intl'
import { useMediaQuery } from 'react-responsive'
import ActionButton from '@shared-ui/components/new/ActionButton'
import { canCreateResource } from '../../utils'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesResourcesActionButton.types'
import TableActions from '@plgd/shared-ui/src/components/new/TableNew/TableActions'

const DevicesResourcesActionButton: FC<Props> = (props) => {
    const { disabled, href, deviceId, interfaces, onCreate, onUpdate, onDelete } = { ...defaultProps, ...props }
    const { formatMessage: _ } = useIntl()
    const isDesktopOrLaptop = useMediaQuery({
        query: '(min-width: 1281px)',
    })

    if (isDesktopOrLaptop) {
        const items = [
            { icon: 'trash', onClick: () => onDelete(href), id: `delete-row-${deviceId}`, tooltipText: _(t.delete) },
            { icon: 'edit', onClick: () => onUpdate({ deviceId, href }), id: `edit-row-${deviceId}`, tooltipText: _(t.update) },
        ]

        if (canCreateResource(interfaces)) {
            items.push({ icon: 'plus', onClick: () => onCreate(href), id: `create-row-${deviceId}`, tooltipText: _(t.create) })
        }

        return <TableActions items={items} />
    }

    return (
        <ActionButton
            disabled={disabled}
            items={[
                {
                    onClick: () => onCreate(href),
                    label: _(t.create),
                    icon: 'plus',
                    hidden: !canCreateResource(interfaces),
                },
                {
                    onClick: () => onUpdate({ deviceId, href }),
                    label: _(t.update),
                    icon: 'edit',
                },
                {
                    onClick: () => onDelete(href),
                    label: _(t.delete),
                    icon: 'trash',
                },
            ]}
            menuProps={{
                placement: 'bottom-end',
            }}
        />
    )
}

DevicesResourcesActionButton.displayName = 'DevicesResourcesActionButton'
DevicesResourcesActionButton.defaultProps = defaultProps

export default DevicesResourcesActionButton
