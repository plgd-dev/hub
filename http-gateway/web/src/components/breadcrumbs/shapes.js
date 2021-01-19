import { PropTypes } from 'prop-types'

export const breadcrumbShape = PropTypes.shape({
  to: PropTypes.string,
  label: PropTypes.string.isRequired,
})

export const breadcrumbsShape = PropTypes.arrayOf(breadcrumbShape)
