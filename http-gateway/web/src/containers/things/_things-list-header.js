import { useIntl } from 'react-intl'
import { useSelector, useDispatch } from 'react-redux'

import { Switch } from '@/components/switch'
import { THINGS_STATUS_WS_KEY } from './constants'
import { selectActiveNotifications, toggleActiveNotification } from './slice'
import { messages as t } from './things-i18n'

export const ThingsListHeader = () => {
  const { formatMessage: _ } = useIntl()
  const dispatch = useDispatch()
  const enabled = useSelector(selectActiveNotifications)?.includes(
    THINGS_STATUS_WS_KEY
  )

  return (
    <Switch
      id="status-notifications"
      label={_(t.notifications)}
      checked={enabled}
      onChange={() => dispatch(toggleActiveNotification(THINGS_STATUS_WS_KEY))}
    />
  )
}
