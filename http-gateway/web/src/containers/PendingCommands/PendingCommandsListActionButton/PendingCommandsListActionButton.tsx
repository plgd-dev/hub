import { FC } from 'react'
import { useIntl } from 'react-intl'

import ActionButton from '@shared-ui/components/new/ActionButton'
import { messages as t } from '../PendingCommands.i18n'
import { Props } from './PendingCommandsListActionButton.types'

// Component is currently not used
export const PendingCommandsListActionButton: FC<Props> = ({ deviceId, href, correlationId, onView, onCancel }) => {
    const { formatMessage: _ } = useIntl()

    return (
        <ActionButton
            items={[
                {
                    onClick: () => onView(deviceId, href, correlationId),
                    label: _(t.details),
                    icon: 'fa-eye',
                },
                {
                    onClick: () => onCancel(deviceId, href, correlationId),
                    label: _(t.cancel),
                    icon: 'fa-times',
                },
            ]}
            menuProps={{
                placement: 'bottom-end',
            }}
        />
    )
}

PendingCommandsListActionButton.displayName = 'PendingCommandsListActionButton'

export default PendingCommandsListActionButton
