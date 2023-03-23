import { FC } from 'react'
import { useIntl } from 'react-intl'

import classNames from 'classnames'
import Switch from '@shared-ui/components/new/Switch'
import { useLocalStorage } from '@shared-ui/common/hooks'
import DevicesResourcesList from '../DevicesResourcesList'
import DevicesResourcesTree from '../DevicesResourcesTree'
import { devicesStatuses } from '../../constants'
import { messages as t } from '../../Devices.i18n'
import { Props } from './DevicesResources.types'

const DevicesResources: FC<Props> = (props) => {
    const { data, onUpdate, onCreate, onDelete, deviceStatus, loading, pageSize } = props
    const { formatMessage: _ } = useIntl()
    const [treeViewActive, setTreeViewActive] = useLocalStorage('treeViewActive', false)
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const greyedOutClassName = classNames({
        'grayed-out': isUnregistered,
    })

    return (
        <>
            <div
                className={classNames('d-flex justify-content-between align-items-center', greyedOutClassName)}
                style={{
                    paddingBottom: 12,
                }}
            >
                <div></div>
                <div className='d-flex justify-content-end align-items-center'>
                    <Switch
                        checked={treeViewActive}
                        disabled={isUnregistered}
                        id='toggle-tree-view'
                        label={_(t.treeView)}
                        onChange={() => setTreeViewActive(!treeViewActive)}
                    />
                </div>
            </div>

            {treeViewActive ? (
                <DevicesResourcesTree data={data} deviceStatus={deviceStatus} loading={loading} onCreate={onCreate} onDelete={onDelete} onUpdate={onUpdate} />
            ) : (
                <DevicesResourcesList
                    data={data}
                    deviceStatus={deviceStatus}
                    loading={loading}
                    onCreate={onCreate}
                    onDelete={onDelete}
                    onUpdate={onUpdate}
                    pageSize={pageSize}
                />
            )}
        </>
    )
}

DevicesResources.displayName = 'DevicesResources'

export default DevicesResources
