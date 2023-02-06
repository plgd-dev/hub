import { FC } from 'react'
import { useIntl } from 'react-intl'

import ActionButton from '@shared-ui/components/new/ActionButton'
import { canCreateResource } from '../../utils'
import { messages as t } from '../../Devices.i18n'
import { Props, defaultProps } from './DevicesResourcesActionButton.types'

const DevicesResourcesActionButton: FC<Props> = props => {
  const { disabled, href, deviceId, interfaces, onCreate, onUpdate, onDelete } =
    { ...defaultProps, ...props }
  const { formatMessage: _ } = useIntl()

  return (
    <ActionButton
      disabled={disabled}
      menuProps={{
        align: 'end',
      }}
      items={[
        {
          onClick: () => onCreate(href),
          label: _(t.create),
          icon: 'fa-plus',
          hidden: !canCreateResource(interfaces),
        },
        {
          onClick: () => onUpdate({ deviceId, href }),
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

DevicesResourcesActionButton.displayName = 'DevicesResourcesActionButton'
DevicesResourcesActionButton.defaultProps = defaultProps

export default DevicesResourcesActionButton
