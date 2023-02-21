import { useIntl } from 'react-intl'
import { devicesStatuses } from '@/containers/Devices/constants'
import { createElement, FC, memo } from 'react'
import Label from '@shared-ui/components/new/Label'
import classNames from 'classnames'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'
import { getValue } from '@shared-ui/common/utils'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import Badge from '@shared-ui/components/new/Badge'
import Switch from '@shared-ui/components/new/Switch'
import { Props } from './DevicesDetails.types'

const DevicesDetails: FC<Props> = memo(
  ({
    data,
    isTwinEnabled,
    loading,
    setTwinSynchronization,
    twinSyncLoading,
  }) => {
    const { formatMessage: _ } = useIntl()
    const deviceStatus = data?.metadata?.connection?.status
    const isOnline = devicesStatuses.ONLINE === deviceStatus
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus

    const LabelWithLoading = (p: any) =>
      createElement(Label, {
        ...p,
        inline: true,
        className: classNames({
          shimmering: loading,
          'grayed-out': isUnregistered,
        }),
      })

    return (
      <Row>
        <Col>
          <LabelWithLoading title="ID">{getValue(data?.id)}</LabelWithLoading>
          <LabelWithLoading title={_(t.types)}>
            <div className="align-items-end badges-box-vertical">
              {data?.types?.map?.((type: string) => (
                <Badge key={type}>{type}</Badge>
              )) || '-'}
            </div>
          </LabelWithLoading>
        </Col>
        <Col>
          <LabelWithLoading title={_(t.status)}>
            <Badge className={isOnline ? 'green' : 'red'}>
              {isOnline ? _(t.online) : _(t.offline)}
            </Badge>
          </LabelWithLoading>
          <LabelWithLoading title={_(t.twinSynchronization)}>
            <Switch
              className="text-left"
              id="toggle-twin-synchronization"
              label={isTwinEnabled ? _(t.enabled) : _(t.disabled)}
              checked={isTwinEnabled}
              onChange={setTwinSynchronization}
              disabled={twinSyncLoading || isUnregistered}
            />
          </LabelWithLoading>
        </Col>
      </Row>
    )
  }
)

DevicesDetails.displayName = 'DevicesDetails'

export default DevicesDetails
