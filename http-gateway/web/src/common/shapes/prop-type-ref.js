import PropTypes from 'prop-types'

export const PropTypeRef = PropTypes.oneOfType([
  PropTypes.func,
  PropTypes.shape({ current: PropTypes.instanceOf(Element) }),
])
