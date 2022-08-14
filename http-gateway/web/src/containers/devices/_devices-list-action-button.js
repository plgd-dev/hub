import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'

import { ActionButton } from '@shared-ui/components/old/action-button'
import { messages as t } from './devices-i18n'

export const DevicesListActionButton = ({ deviceId, onView, onDelete }) => {
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

DevicesListActionButton.propTypes = {
  deviceId: PropTypes.string.isRequired,
  onView: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
}
