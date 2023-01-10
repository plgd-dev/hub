import { createElement, memo } from 'react'
import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'
import classNames from 'classnames'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'

import { Switch } from '@/components/switch'
import { Badge } from '@/components/badge'
import { Label } from '@/components/label'
import { getValue } from '@/common/utils'

import { devicesStatuses } from '../constants'
import { deviceShape } from '../shapes'
import { messages as t } from '../devices-i18n'

export const DevicesDetails = memo(
  ({ data, loading, twinSyncLoading, setTwinSynchronization }) => {
    const { formatMessage: _ } = useIntl()
    const deviceStatus = data?.metadata?.connection?.status
    const isOnline = devicesStatuses.ONLINE === deviceStatus
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const isTwinEnabled = data?.metadata?.twinEnabled
    const LabelWithLoading = p =>
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
              {data?.types?.map?.(type => <Badge key={type}>{type}</Badge>) ||
                '-'}
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

DevicesDetails.propTypes = {
  data: deviceShape,
  loading: PropTypes.bool.isRequired,
  twinSyncLoading: PropTypes.bool.isRequired,
  setTwinSynchronization: PropTypes.func.isRequired,
}

DevicesDetails.defaultProps = {
  data: null,
}
