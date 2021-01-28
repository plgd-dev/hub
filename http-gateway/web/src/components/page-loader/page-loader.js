import { memo } from 'react'
import PropTypes from 'prop-types'

export const PageLoader = memo(({ loading, className }) => {
  return loading ? (
    <div id="page-loader" role="alert" aria-busy="true" className={className}>
      <div className="bar">
        <div className="progress">
          <div className="subline inc" />
          <div className="subline dec" />
        </div>
      </div>
    </div>
  ) : null
})

PageLoader.propTypes = {
  loading: PropTypes.bool.isRequired,
  className: PropTypes.string,
}

PageLoader.defaultProps = {
  className: null,
}
