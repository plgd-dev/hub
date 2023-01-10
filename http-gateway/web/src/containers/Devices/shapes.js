import PropTypes from 'prop-types'

import { devicesStatuses } from './constants'

const { ONLINE, OFFLINE, REGISTERED, UNREGISTERED } = devicesStatuses

export const deviceResourceShape = PropTypes.shape({
  deviceId: PropTypes.string,
  href: PropTypes.string,
  resourceTypes: PropTypes.arrayOf(PropTypes.string),
  interfaces: PropTypes.arrayOf(PropTypes.string),
})

export const devicesResourceLinkShape = PropTypes.shape({
  href: PropTypes.string,
  deviceId: PropTypes.string,
  resourceTypes: PropTypes.arrayOf(PropTypes.string),
  interfaces: PropTypes.arrayOf(PropTypes.string),
  anchor: PropTypes.string,
  title: PropTypes.string,
  supportedContents: PropTypes.arrayOf(PropTypes.string),
  validUntil: PropTypes.string,
  policies: PropTypes.shape({
    bigFlags: PropTypes.number,
  }),
  endpointInformations: PropTypes.arrayOf(
    PropTypes.shape({
      endpoint: PropTypes.string,
      priority: PropTypes.string,
    })
  ),
})

export const deviceShape = PropTypes.shape({
  id: PropTypes.string,
  types: PropTypes.arrayOf(PropTypes.string),
  name: PropTypes.string,
  metadata: PropTypes.shape({
    connection: PropTypes.shape({
      status: PropTypes.oneOf([ONLINE, OFFLINE, REGISTERED, UNREGISTERED]),
    }),
    twinEnabled: PropTypes.bool,
  }),
})
