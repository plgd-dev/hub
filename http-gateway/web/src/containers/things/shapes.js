import PropTypes from 'prop-types'

import { thingsStatuses } from './constants'

const { ONLINE, OFFLINE, REGISTERED, UNREGISTERED } = thingsStatuses

export const thingResourceShape = PropTypes.shape({
  di: PropTypes.string,
  href: PropTypes.string,
  rt: PropTypes.arrayOf(PropTypes.string),
  if: PropTypes.arrayOf(PropTypes.string),
})

export const thingShape = PropTypes.shape({
  device: PropTypes.shape({
    rt: PropTypes.arrayOf(PropTypes.string),
    di: PropTypes.string,
    n: PropTypes.string,
  }),
  status: PropTypes.oneOf([ONLINE, OFFLINE, REGISTERED, UNREGISTERED]),
  links: PropTypes.arrayOf(thingResourceShape),
})
