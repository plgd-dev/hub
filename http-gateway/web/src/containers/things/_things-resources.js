import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import classNames from 'classnames'

import { Switch } from '@/components/switch'
import { useLocalStorage } from '@/common/hooks'
import { ThingsResourcesList } from './_things-resources-list'
import { ThingsResourcesTree } from './_things-resources-tree'
import { thingsStatuses } from './constants'
import { thingResourceShape } from './shapes'
import { messages as t } from './things-i18n'

export const ThingsResources = ({
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
  const isUnregistered = thingsStatuses.UNREGISTERED === deviceStatus
  const greyedOutClassName = classNames({
    'grayed-out': isUnregistered,
  })

  return (
    <>
      <div className="d-flex justify-content-between align-items-center">
        <h2 className={classNames(greyedOutClassName)}>{_(t.resources)}</h2>
        <div className="d-flex justify-content-end align-items-center">
          <Switch
            id="toggle-tree-view"
            label={_(t.treeView)}
            checked={treeViewActive}
            onChange={() => setTreeViewActive(!treeViewActive)}
          />
        </div>
      </div>

      {treeViewActive ? (
        <ThingsResourcesTree
          data={data}
          onUpdate={onUpdate}
          onCreate={onCreate}
          onDelete={onDelete}
          deviceStatus={deviceStatus}
          loading={loading}
        />
      ) : (
        <ThingsResourcesList
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

ThingsResources.propTypes = {
  data: PropTypes.arrayOf(thingResourceShape),
  onCreate: PropTypes.func.isRequired,
  onUpdate: PropTypes.func.isRequired,
  onDelete: PropTypes.func.isRequired,
  loading: PropTypes.bool.isRequired,
  deviceStatus: PropTypes.oneOf(Object.values(thingsStatuses)),
}

ThingsResources.defaultProps = {
  data: null,
  deviceStatus: null,
}
