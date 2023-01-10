import { FC } from 'react'
import { useIntl } from 'react-intl'
import { ActionButton } from '@/components/action-button'
import { messages as t } from '../../devices-i18n'
import { Props } from './DevicesListActionButton.types'

const DevicesListActionButton: FC<Props> = props => {
  const { deviceId, onView, onDelete } = props
  const { formatMessage: _ } = useIntl()

  return (
    <ActionButton
      menuProps={{
        align: 'right',
      }}
      items={[
        {
          onClick: () => onView(deviceId),
          label: _(t.details),
          icon: 'fa-eye',
        },
        {
          onClick: () => onDelete(deviceId),
          label: _(t.delete),
          icon: 'fa-trash-alt',
        },
      ]}
    >
      <i className="fas fa-ellipsis-h" />
    </ActionButton>
  )
}

DevicesListActionButton.displayName = 'DevicesListActionButton'

export default DevicesListActionButton
