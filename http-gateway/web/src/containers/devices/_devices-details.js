import { createElement, memo } from 'react'
import { useIntl } from 'react-intl'
import PropTypes from 'prop-types'
import classNames from 'classnames'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'

import { Switch } from '@shared-ui/components/old/switch'
import { Badge } from '@shared-ui/components/old/badge'
import { Label } from '@shared-ui/components/old/label'
import { getValue } from '@shared-ui/common/utils'

import { devicesStatuses } from './constants'
import { deviceShape } from './shapes'
import { shadowSynchronizationEnabled } from './utils'
import { messages as t } from './devices-i18n'

export const DevicesDetails = memo(
  ({ data, loading, shadowSyncLoading, setShadowSynchronization }) => {
    const { formatMessage: _ } = useIntl()
    const deviceStatus = data?.metadata?.status?.value
    const isOnline = devicesStatuses.ONLINE === deviceStatus
    const isUnregistered = devicesStatuses.UNREGISTERED === deviceStatus
    const isShadowSynchronizationEnabled = shadowSynchronizationEnabled(
      data?.metadata?.shadowSynchronization
    )
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
          <LabelWithLoading title={_(t.shadowSynchronization)}>
            <Switch
              className="text-left"
              id="toggle-shadow-synchronization"
              label={
                isShadowSynchronizationEnabled ? _(t.enabled) : _(t.disabled)
              }
              checked={isShadowSynchronizationEnabled}
              onChange={setShadowSynchronization}
              disabled={shadowSyncLoading || isUnregistered}
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
  shadowSyncLoading: PropTypes.bool.isRequired,
  setShadowSynchronization: PropTypes.func.isRequired,
}

DevicesDetails.defaultProps = {
  data: null,
}
