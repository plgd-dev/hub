import PropTypes from 'prop-types'

import { commandTypes } from './constants'

export const pendingCommandShape = PropTypes.shape({
  auditContext: PropTypes.shape({
    userId: PropTypes.string,
    correclationId: PropTypes.string,
  }),
  commandType: PropTypes.oneOf(Object.values(commandTypes)),
  content: PropTypes.object,
  eventMetadata: PropTypes.shape({
    connectionId: PropTypes.string,
    sequence: PropTypes.string,
    timestamp: PropTypes.string,
    version: PropTypes.string,
  }),
  resourceId: PropTypes.shape({
    deviceId: PropTypes.string,
    href: PropTypes.string,
  }),
  resourceInterface: PropTypes.string,
  validUntil: PropTypes.string,
})
