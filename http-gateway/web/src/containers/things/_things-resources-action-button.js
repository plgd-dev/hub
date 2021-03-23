import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'

import { ActionButton } from '@/components/action-button'
import { canCreateResource } from './utils'
import { messages as t } from './things-i18n'

export const ThingsResourcesActionButton = ({
  disabled,
  href,
  di,
  interfaces,
  onCreate,
  onUpdate,
  onDelete,
}) => {
  const { formatMessage: _ } = useIntl()

  return (
    <ActionButton
      disabled={disabled}
      menuProps={{
        align: 'right',
      }}
      items={[
        {
          onClick: () => onCreate(href),
          label: _(t.create),
          icon: 'fa-plus',
          hidden: !canCreateResource(interfaces),
        },
        {
          onClick: () => onUpdate({ di, href }),
          label: _(t.update),
          icon: 'fa-pen',
        },
        {
          onClick: () => onDelete(href),
          label: _(t.delete),
          icon: 'fa-trash-alt',
        },
      ]}
    >
      <i className="fas fa-ellipsis-h" />
    </ActionButton>
  )
}

ThingsResourcesActionButton.propTypes = {
  disabled: PropTypes.bool.isRequired,
  href: PropTypes.string.isRequired,
  di: PropTypes.string.isRequired,
  onCreate: PropTypes.func.isRequired,
  onUpdate: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
  interfaces: PropTypes.arrayOf(PropTypes.string),
}

ThingsResourcesActionButton.defaultProps = {
  interfaces: [],
}
