import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { Switch } from '@/components/switch'
import { useLocalStorage } from '@/common/hooks'
import { DevicesResourcesList } from './_devices-resources-list'
import { DevicesResourcesTree } from './_devices-resources-tree'
import { devicesStatuses } from './constants'
import { deviceResourceShape } from './shapes'
import { messages as t } from './devices-i18n'

export const DevicesResources = ({
  data,
  onUpdate,
  onCreate,
  onDelete,
  deviceStatus,
  loading,
}) => {
  const { formatMessage: _ } = useIntl()
  const [treeViewActive, setTreeViewActive] = useLocalStorage(
    'treeViewActive',
    false
  )
  const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
  const greyedOutClassName = classNames({
    'grayed-out': isUnregistered,
  })

  return (
    <>
      <div
        className={classNames(
          'd-flex justify-content-between align-items-center',
          greyedOutClassName
        )}
      >
        <h2>{_(t.resources)}</h2>
        <div className="d-flex justify-content-end align-items-center">
          <Switch
            id="toggle-tree-view"
            label={_(t.treeView)}
            checked={treeViewActive}
            onChange={() => setTreeViewActive(!treeViewActive)}
            disabled={isUnregistered}
          />
        </div>
      </div>

      {treeViewActive ? (
        <DevicesResourcesTree
          data={data}
          onUpdate={onUpdate}
          onCreate={onCreate}
          onDelete={onDelete}
          deviceStatus={deviceStatus}
          loading={loading}
        />
      ) : (
        <DevicesResourcesList
          data={data}
          onUpdate={onUpdate}
          onCreate={onCreate}
          onDelete={onDelete}
          deviceStatus={deviceStatus}
          loading={loading}
        />
      )}
    </>
  )
}

DevicesResources.propTypes = {
  data: PropTypes.arrayOf(deviceResourceShape),
  onCreate: PropTypes.func.isRequired,
  onUpdate: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  deviceStatus: PropTypes.oneOf(Object.values(devicesStatuses)),
}

DevicesResources.defaultProps = {
  data: null,
  deviceStatus: null,
}
