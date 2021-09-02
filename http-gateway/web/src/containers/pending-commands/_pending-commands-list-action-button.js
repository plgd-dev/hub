import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'

import { ActionButton } from '@/components/action-button'
import { messages as t } from './pending-commands-i18n'

// Component is currently not used
export const PendingCommandsListActionButton = ({
  deviceId,
  href,
  correlationId,
  onView,
  onCancel,
}) => {
  const { formatMessage: _ } = useIntl()

  return (
    <ActionButton
      menuProps={{
        align: 'right',
      }}
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
    >
      <i className="fas fa-ellipsis-h" />
    </ActionButton>
  )
}

PendingCommandsListActionButton.propTypes = {
  deviceId: PropTypes.string.isRequired,
  href: PropTypes.string.isRequired,
  correlationId: PropTypes.string.isRequired,
  onView: PropTypes.func.isRequired,
  onCancel: PropTypes.func.isRequired,
}
