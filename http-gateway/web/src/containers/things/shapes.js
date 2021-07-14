import PropTypes from 'prop-types'

import { thingsStatuses } from './constants'

const { ONLINE, OFFLINE, REGISTERED, UNREGISTERED } = thingsStatuses

export const thingResourceShape = PropTypes.shape({
  di: PropTypes.string,
  href: PropTypes.string,
  rt: PropTypes.arrayOf(PropTypes.string),
  if: PropTypes.arrayOf(PropTypes.string),
})

export const thingsResourceLinkShape = PropTypes.shape({
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

export const thingShape = PropTypes.shape({
  id: PropTypes.string,
  types: PropTypes.arrayOf(PropTypes.string),
  name: PropTypes.string,
  metadata: PropTypes.shape({
    status: PropTypes.shape({
      value: PropTypes.oneOf([ONLINE, OFFLINE, REGISTERED, UNREGISTERED]),
    }),
  }),
})
